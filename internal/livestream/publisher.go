package livestream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/randutil"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

const (
	signalTimeout = 3 * time.Second
)

// publisher implements Livestream interface.
type publisher struct {
	// meta contains id and track source of this publisher.
	meta *pb.Meta

	config broadcastConfigOptions
	client mqtt.Client

	createTrack  func() (webrtc.TrackLocal, error)
	streamSource func() string

	// liveStream blocks indefinitely if there no error.
	// It should listens to ctx.Done, and exit when done.
	liveStream func(ctx context.Context, address string, videoTrack webrtc.TrackLocal, logger *zerolog.Logger) error

	pendingCandidates []*webrtc.ICECandidate
	candidatesMux     sync.Mutex

	isLivestreamStarted bool
	stateMux            sync.Mutex

	logger zerolog.Logger
}

func (p *publisher) Publish() error {
	p.logger = p.logger.With().Str("id", p.meta.Id).Int32("track_source", int32(p.meta.TrackSource)).Logger()
	p.logger.Info().Msg("publishing stream")

	videoTrack, err := p.createTrack()
	if err != nil {
		return err
	}
	p.logger.Info().Msg("created video track")

	if err := p.createPeerConnection(videoTrack); err != nil {
		return fmt.Errorf("failed to create PeerConnection: %w", err)
	}
	p.logger.Info().Msg("created PeerConnection")

	p.logger.Info().Bool("consume_stream_on_demand", p.config.ConsumeStreamOnDemand).Send()
	if p.config.ConsumeStreamOnDemand {
		if err := <-p.listenSubscriber(videoTrack); err != nil {
			return fmt.Errorf("listening subscriber failed: %w", err)
		}
	} else {
		if err := p.liveStream(context.Background(), p.streamSource(), videoTrack, &p.logger); err != nil {
			p.logger.Err(err).Msg("live stream failed")
			return fmt.Errorf("live stream failed: %w", err)
		}
	}
	p.logger.Info().Msg("live stream is over")

	return nil
}

func (p *publisher) Meta() *pb.Meta {
	return p.meta
}

func (p *publisher) createPeerConnection(videoTrack webrtc.TrackLocal) error {
	answerChan := p.recvAnswer()
	candidateChan := p.recvCandidate()

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{p.config.ICEServer},
				Username:   p.config.Username,
				Credential: p.config.Credential,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not create PeerConnection: %w", err)
	}

	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return fmt.Errorf("could not add track to PeerConnection: %w", err)
	}
	go p.processRTCP(rtpSender)

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		if desc := peerConnection.RemoteDescription(); desc == nil {
			p.candidatesMux.Lock()
			p.pendingCandidates = append(p.pendingCandidates, c)
			p.candidatesMux.Unlock()

			return
		}

		if err = p.sendCandidate(c); err != nil {
			p.logger.Err(err).Msg("could not send candidate")
		}
		p.logger.Info().Msg("sent an ICEcandidate")
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(p.handleICEConnectionStateChange(peerConnection, videoTrack))

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("could not create offer: %w", err)
	}

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("could not set local description: %w", err)
	}

	if err := p.sendOffer(peerConnection.LocalDescription()); err != nil {
		return fmt.Errorf("could not send offer: %w", err)
	}
	p.logger.Info().Msg("sent local description offer")

	// Receiving answer.
	timer := time.NewTimer(signalTimeout)
	defer timer.Stop()
	select {
	case answer := <-answerChan:
		if err := peerConnection.SetRemoteDescription(*answer); err != nil {
			p.logger.Err(err).Msg("could not set remote description")
		}
		p.logger.Info().Msg("set remote description")
	case <-timer.C:
		p.logger.Warn().Dur("timeout", signalTimeout).Msg("timed out receiving answer")

		if err := closePeerConnection(peerConnection); err != nil {
			p.logger.Panic().Err(err).Msg("closing PeerConnection")
		}

		return errors.New("timed out receiving answer")
	}

	// Signal candidate after setting remote description.
	go p.signalCandidate(peerConnection, candidateChan)

	p.candidatesMux.Lock()
	defer func() {
		p.emptyPendingCandidate()
		p.candidatesMux.Unlock()
	}()

	for _, c := range p.pendingCandidates {
		if err := p.sendCandidate(c); err != nil {
			return fmt.Errorf("could not send candidate: %w", err)
		}
		p.logger.Info().Msg("sent an ICEcandidate")
	}

	return nil
}

func (p *publisher) handleICEConnectionStateChange(peerConnection *webrtc.PeerConnection, videoTrack webrtc.TrackLocal) func(connectionState webrtc.ICEConnectionState) {
	return func(connectionState webrtc.ICEConnectionState) {
		p.logger.Info().Str("state", connectionState.String()).Msg("connection state has changed")

		if connectionState == webrtc.ICEConnectionStateFailed {
			if err := closePeerConnection(peerConnection); err != nil {
				p.logger.Panic().Err(err).Msg("closing PeerConnection")
			}
		}

		if connectionState == webrtc.ICEConnectionStateClosed {
			// Retry creating peer connection only when network is ok.
			for {
				if !p.client.IsConnected() || !p.client.IsConnectionOpen() {
					continue
				}

				if err := p.createPeerConnection(videoTrack); err != nil {
					p.logger.Err(err).Msg("failed to create peer connection after retrying")
					continue
				}

				break
			}
		}

		if connectionState == webrtc.ICEConnectionStateDisconnected {
			p.stateMux.Lock()
			p.isLivestreamStarted = false
			p.candidatesMux.Unlock()
		}
	}
}

func (p *publisher) signalCandidate(peerConnection *webrtc.PeerConnection, ch <-chan string) {
	timer := time.NewTimer(signalTimeout)
	defer timer.Stop()

	select {
	case c := <-ch:
		if err := peerConnection.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: c,
		}); err != nil {
			p.logger.Err(err).Msg("could not add ICE candidate")
		}
		p.logger.Info().Str("candidate", c).Msg("successfully added an ICE candidate")
	case <-timer.C:
		p.logger.Debug().Dur("timeout", signalTimeout).Msg("timed out receiving candidate")
	}
}

// closePeerConnection tidies RTPSender and remvoes track from peer connection.
// It's used after peer connection fails.
func closePeerConnection(peerConnection *webrtc.PeerConnection) error {
	if peerConnection == nil {
		return nil
	}
	for _, sender := range peerConnection.GetSenders() {
		if err := sender.Stop(); err != nil {
			return fmt.Errorf("could not stop RTP sender: %w", err)
		}
		if err := peerConnection.RemoveTrack(sender); err != nil {
			return fmt.Errorf("could not remove track: %w", err)
		}
	}
	return peerConnection.Close()
}

// processRTCP reads incoming RTCP packets
// Before these packets are returned they are processed by interceptors.
// For things like NACK this needs to be called.
func (p *publisher) processRTCP(rtpSender *webrtc.RTPSender) {
	for {
		if _, _, rtcpErr := rtpSender.ReadRTCP(); rtcpErr != nil {
			if errors.Is(rtcpErr, io.EOF) || errors.Is(rtcpErr, io.ErrClosedPipe) {
				_ = rtpSender.Stop()
			} else {
				p.logger.Err(rtcpErr).Send()
			}
			return
		}
	}
}

// videoTrackRTP creates a RTP video track.
// The default MIME type is H.264
func videoTrackRTP() (webrtc.TrackLocal, error) {
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		fmt.Sprintf("video-%d", randutil.NewMathRandomGenerator().Uint32()),
		fmt.Sprintf("edge-%d", randutil.NewMathRandomGenerator().Uint32()),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create TrackLocalStaticRTP: %w", err)
	}
	return videoTrack, nil
}

// videoTrackSample creates a sample video track.
// The default MIME type is H.264
func videoTrackSample() (webrtc.TrackLocal, error) {
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		fmt.Sprintf("video-%d", randutil.NewMathRandomGenerator().Uint32()),
		fmt.Sprintf("edge-%d", randutil.NewMathRandomGenerator().Uint32()),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create TrackLocalStaticSample: %w", err)
	}
	return videoTrack, nil
}

// emptyPendingCandidate is called after all ICE candidates were sent to release resources.
func (p *publisher) emptyPendingCandidate() {
	p.pendingCandidates = p.pendingCandidates[:0]
}

func (p *publisher) listenSubscriber(videoTrack webrtc.TrackLocal) <-chan error {
	// Ctx should be renewed on every canceling.
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)
	topic := p.config.NotifyStreamTopicPrefix + "/" + p.meta.Id + "/" + strconv.Itoa(int(p.meta.TrackSource))

	p.client.Subscribe(topic, byte(p.config.Qos), func(_ mqtt.Client, m mqtt.Message) {
		p.stateMux.Lock()
		defer p.stateMux.Unlock()

		counter, err := strconv.Atoi(string(m.Payload()))
		if err != nil {
			p.logger.Err(err).Msg("could not parse message payload")
			return
		}
		p.logger.Info().Int("subscriber_counter", counter).Msg("received subscriber counter")

		if counter == 0 {
			cancel()
			p.logger.Info().Msg("cancel living stream now")

			ctx, cancel = context.WithCancel(context.Background())
			p.isLivestreamStarted = false

			return
		}

		if p.isLivestreamStarted {
			p.logger.Debug().Msg("live stream was already started")
			return
		}

		go func() {
			if err := p.liveStream(ctx, p.streamSource(), videoTrack, &p.logger); err != nil {
				p.logger.Err(err).Msg("live stream failed")
				errChan <- fmt.Errorf("live stream failed: %w", err)

				return
			}
		}()
		p.logger.Info().Msg("start living stream now")

		p.isLivestreamStarted = true
	})

	return errChan
}
