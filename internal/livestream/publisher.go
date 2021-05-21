package livestream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

const (
	// maxBurstRetries is the maximum burst retry numbers.
	maxBurstRetries = 10

	// burstRetriesGroupInterval is the interval between burst retries group.
	burstRetriesGroupInterval = 1 * time.Minute
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

	logger zerolog.Logger

	// burstRetriesNo is an one time counter, will be reset to zero after peer connection success or before next burst retry.
	burstRetriesNo uint32

	// subscriberCounter is the front end subscriber counter.
	// subscriberCounter's initial value is 0.
	// The counter should be reset when edge disconnected and then reconnected again from mqtt broker.
	subscriberCounter uint32
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

	if err := <-p.listenSubscriber(videoTrack); err != nil {
		return fmt.Errorf("listening subscriber failed: %w", err)
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

		p.candidatesMux.Lock()
		defer p.candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			p.pendingCandidates = append(p.pendingCandidates, c)
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

	// TODO: Timeout channel receiving to avoid blocking.
	answer := <-answerChan
	if answer == nil {
		return nil
	}
	if err := peerConnection.SetRemoteDescription(*answer); err != nil {
		return fmt.Errorf("could not set remote description: %w", err)
	}
	p.logger.Info().Msg("received remote answer from cloud")

	// Signal candidate after setting remote description.
	go p.signalCandidate(peerConnection, candidateChan)

	// Signal candidate
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
			p.logger.Info().Msg("PeerConnection has been closed")

			n := atomic.LoadUint32(&p.burstRetriesNo)
			if n == maxBurstRetries {
				p.logger.Info().Dur("interval", burstRetriesGroupInterval).Msg("maximum burst retries reached, waiting for an interval time to continue next burst retries")
				<-time.NewTimer(burstRetriesGroupInterval).C
				// Reset counter after burst retries group interval.
				atomic.StoreUint32(&p.burstRetriesNo, 0)
			}
			// currying call function.
			p.logger.Info().Uint32("retry_no", n+1).Msg("retry creating peer connection")
			if err := p.createPeerConnection(videoTrack); err != nil {
				p.logger.Err(err).Msg("failed to create peer connection")
			}
			atomic.AddUint32(&p.burstRetriesNo, 1)
		}

		if connectionState == webrtc.ICEConnectionStateConnected {
			// If connection is successful, whether it's retried or not, counter should be reset to zero.
			atomic.StoreUint32(&p.burstRetriesNo, 0)

			if p.subscriberCounter > 0 {
				// Reset subscriberCounter upon reconnected.
				atomic.StoreUint32(&p.subscriberCounter, 0)
				p.logger.Info().Uint32("subscriber_counter", p.subscriberCounter).Msg("reset subscriber counter to 0 upon ice reconnected")
			}
		}
	}
}

func (p *publisher) signalCandidate(peerConnection *webrtc.PeerConnection, ch <-chan string) {
	// TODO: Stop adding ICE candidate when after signaling succeeded, that is, to exit the loop.
	// Just set a timer is not enough.
	for c := range ch {
		if err := peerConnection.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: c,
		}); err != nil {
			p.logger.Err(err).Msg("could not add ICE candidate")
		}
		p.logger.Info().Str("candidate", c).Msg("successfully added an ICE candidate")
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
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
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
	id := uuid.New().String()
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video-"+id,
		"edge-"+id,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create TrackLocalStaticRTP: %w", err)
	}
	return videoTrack, nil
}

// videoTrackSample creates a sample video track.
// The default MIME type is H.264
func videoTrackSample() (webrtc.TrackLocal, error) {
	id := uuid.New().String()
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video-"+id,
		"edge-"+id,
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

// listenSubscriber listens to subscribers' ice connection stats.
// When there are at least one streaming subscribers, live stream never stops,
// when there is not even one subscriber, live stream stops.
func (p *publisher) listenSubscriber(videoTrack webrtc.TrackLocal) <-chan error {
	// Ctx should be renewed on every canceling.
	ctx, cancel := context.WithCancel(context.Background())
	// errChan should never be closed when all subscribers disconnected,
	// It means always listens to subscribers' stat, only sends real error to errChan.
	errChan := make(chan error, 1)

	topic := p.config.HookStreamTopicPrefix + "/" + p.meta.Id + "/" + strconv.Itoa(int(p.meta.TrackSource))
	p.client.Subscribe(topic, byte(p.config.Qos), func(_ mqtt.Client, m mqtt.Message) {
		defer func() {
			p.logger.Info().Uint32("subscriber_counter", p.subscriberCounter).Msg("processed subscriber state")
		}()
		payload, err := strconv.Atoi(string(m.Payload()))
		if err != nil {
			p.logger.Err(err).Msg("could not parse message payload")
			return
		}

		state := webrtc.ICEConnectionState(payload)
		p.logger.Info().Str("state", state.String()).Uint32("subscriber_counter", p.subscriberCounter).Msg("received subscriber state")

		switch state {
		case webrtc.ICEConnectionStateConnected:
			defer func() {
				// always increment subscriberCounter by one when new subscribers connected at last.
				atomic.AddUint32(&p.subscriberCounter, 1)
			}()

			if p.subscriberCounter > 0 {
				// If there are already subscribers living stream, don't start the already started stream.
				return
			}
			// There is no subscriber yet, start streaming for this new subscriber now.
			go func() {
				if err := p.liveStream(ctx, p.streamSource(), videoTrack, &p.logger); err != nil {
					p.logger.Err(err).Msg("live stream failed")
					errChan <- fmt.Errorf("live stream failed: %w", err)
					return
				}
			}()
			p.logger.Info().Msg("started living stream")
		case webrtc.ICEConnectionStateDisconnected:
			if p.subscriberCounter == 0 {
				return
			}
			// always decrement subscriberCounter by one if it's larger than 0 when new subscribers disconnected.
			atomic.AddUint32(&p.subscriberCounter, ^uint32(0))
			if p.subscriberCounter > 0 {
				// If there are already subscribers living stream, just keep streaming.
				return
			}
			// The last subscriber is disconnected, now stop living stream.
			p.logger.Info().Uint32("subscriber_counter", p.subscriberCounter).Msg("the last subscriber is disconnected, cancel streaming now")
			cancel()
			// Renew context.
			ctx, cancel = context.WithCancel(context.Background())
			p.logger.Info().Msg("stream canceled")
		}
	})
	return errChan
}
