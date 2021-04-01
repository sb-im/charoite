package livestream

import (
	"fmt"

	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

// publisher implements Livestream interface.
type publisher struct {
	// id is an unique id for this publisher which is bound to OS's machine id.
	id string

	// trackSource is the source of video track, either drone or monitor.
	trackSource pb.TrackSource

	config       broadcastConfigOptions
	client       mqtt.Client
	createTrack  func() (webrtc.TrackLocal, error)
	streamSource func() string

	// liveStream blocks indefinitely if there no error.
	liveStream func(address string, videoTrack webrtc.TrackLocal, logger *zerolog.Logger) error

	logger zerolog.Logger
}

func (p *publisher) Publish() error {
	p.logger = p.logger.With().Str("id", p.id).Int32("track_source", int32(p.trackSource)).Logger()
	p.logger.Info().Msg("publishing stream")

	videoTrack, err := p.createTrack()
	if err != nil {
		return err
	}
	p.logger.Debug().Msg("created video track")

	if err := p.createPeerConnection(videoTrack); err != nil {
		return fmt.Errorf("failed to create PeerConnection: %w", err)
	}
	p.logger.Debug().Msg("created PeerConnection")

	if err := p.liveStream(p.streamSource(), videoTrack, &p.logger); err != nil {
		return fmt.Errorf("live stream failed: %w", err)
	}
	p.logger.Debug().Msg("live stream is over")

	return nil
}

func (p *publisher) ID() string {
	return p.id
}

func (p *publisher) TrackSource() pb.TrackSource {
	return p.trackSource
}

func (p *publisher) createPeerConnection(videoTrack webrtc.TrackLocal) error {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{p.config.ICEServer},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not create PeerConnection: %w", err)
	}

	// A signal for ICE connection.
	// iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return fmt.Errorf("could not add track to PeerConnection: %w", err)
	}
	go p.processRTCP(rtpSender)

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		p.logger.Debug().Str("state", connectionState.String()).Msg("connection state has changed")

		// if connectionState == webrtc.ICEConnectionStateConnected {
		// 	iceConnectedCtxCancel()
		// }

		if connectionState == webrtc.ICEConnectionStateFailed {
			if err = peerConnection.Close(); err != nil {
				p.logger.Panic().Err(err).Msg("closing PeerConnection")
			}
			p.logger.Info().Msg("PeerConnection has been closed")
		}
	})

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("could not create offer: %w", err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("could not set local description: %w", err)
	}
	<-gatherComplete

	if err := p.sendOffer(peerConnection.LocalDescription()); err != nil {
		return fmt.Errorf("could not send offer: %w", err)
	}
	p.logger.Debug().Msg("sent local description offer")

	answer := <-p.recvAnswer()
	if answer == nil {
		return nil
	}
	if err := peerConnection.SetRemoteDescription(*answer); err != nil {
		return fmt.Errorf("could not set remote description: %w", err)
	}
	p.logger.Debug().Msg("received remote answer from cloud")

	return nil
}

// processRTCP reads incoming RTCP packets
// Before these packets are returned they are processed by interceptors.
// For things like NACK this needs to be called.
func (p *publisher) processRTCP(rtpSender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			p.logger.Err(rtcpErr)
			return
		}
	}
}

// videoTrackRTP creates a RTP video track.
// The default MIME type is H.264
func videoTrackRTP() (webrtc.TrackLocal, error) {
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video",
		"edge_drone",
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
		"video_webcam",
		"edge_webcam",
	)
	if err != nil {
		return nil, fmt.Errorf("could not create TrackLocalStaticSample: %w", err)
	}
	return videoTrack, nil
}
