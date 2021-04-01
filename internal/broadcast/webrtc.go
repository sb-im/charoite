package broadcast

import (
	"errors"
	"fmt"
	"io"
	"time"

	pb "github.com/SB-IM/pb/signal"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
)

// actor distinguishes between publisher and subscriber.
type actor int

const (
	peerPublisher actor = iota
	peerSubscriber

	rtcpPLIInterval = time.Second * 3
)

func (a actor) string() string {
	if a == peerPublisher {
		return "publisher"
	}
	return "subscriber"
}

// createLocalTrack creates a local video track shared between publisher and subscriber peers.
// localTrack is a transfer that transfers video track between publisher and subscriber peers.
// For localTrack, there can only be one publisher peer, but subscribers can be many.
func createLocalTrack() (*webrtc.TrackLocalStaticRTP, error) {
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		return nil, fmt.Errorf("coult not create TrackLocalStaticRTP: %w", err)
	}
	return videoTrack, nil
}

// createPublisher creates a session between local and remote webRTC peer which is a publisher.
// It receives and transfers video track from remote publisher peer to local peer and is fired by remote publisher peer by RPC call.
func (s *session) createPublisher(videoTrack *webrtc.TrackLocalStaticRTP) error {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{s.config.ICEServer},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not create PeerConnection: %w", err)
	}

	// Allow us to receive 1 video track
	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		return fmt.Errorf("could not add tranceiver from kind: %w", err)
	}

	// Set a handler for when a new remote track starts, this just distributes all our packets
	// to connected peers
	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		go sendRTCP(peerConnection, t, &s.logger)

		rtpBuf := make([]byte, 1400)
		for {
			i, _, readErr := t.Read(rtpBuf)
			if readErr != nil {
				s.logger.Panic().Err(err).Msg("could not read buffer")
			}

			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err = videoTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				s.logger.Panic().Err(err).Msg("could not write video track")
			}
		}
	})

	if err := s.createPeerConnection(peerConnection, peerPublisher); err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	s.logger.Debug().Str("id", string(s.id)).Int32("track_source", int32(s.trackSource)).Msg("created PeerConnection for publisher")

	return nil
}

// createSubscriber creates a session between local and remote webRTC peer which is a subscriber.
// It broadcasts video track from local to remote subscriber peer and is fired by remote subscriber peer by RCP call.
func (s *session) createSubscriber(videoTrack *webrtc.TrackLocalStaticRTP) error {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{s.config.ICEServer},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not create PeerConnection: %w", err)
	}

	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return fmt.Errorf("could not add track: %w", err)
	}
	go processRTCP(rtpSender, &s.logger)

	if err := s.createPeerConnection(peerConnection, peerSubscriber); err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	s.logger.Debug().Msg("created PeerConnection for subscriber")

	return nil
}

// createPeerConnection creates peer connection to the remote webRTC peer who sends SDP offer.
// It's a generic function for both publisher and subscriber.
func (s *session) createPeerConnection(peerConnection *webrtc.PeerConnection, actor actor) error {
	// Receive remote offer.
	var offer *pb.SessionDescription
	if actor == peerPublisher {
		offer = <-s.publisherChans.OfferChan

		// Set up session identity.
		s.id = machineID(offer.Id)
		s.trackSource = offer.TrackSource
	} else {
		offer = <-s.subscriberChans.offerChan
	}
	logger := s.logger.With().
		Str("actor", actor.string()).
		Str("id", offer.Id).
		Int32("track_source",
			int32(offer.TrackSource)).
		Logger()

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		logger.Debug().Str("state", connectionState.String()).Msg("ICE connection state has changed")
		if connectionState == webrtc.ICEConnectionStateFailed {
			if err := peerConnection.Close(); err != nil {
				logger.Panic().Err(err).Msg("could not close PeerConnection")
			}
			logger.Debug().Msg("PeerConnection has been closed")
		}
	})

	if err := peerConnection.SetRemoteDescription(pbSdp2webrtcSdp(offer)); err != nil {
		return fmt.Errorf("could not set remote describption: %w", err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("could not create answer: %w", err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("could not set local description: %w", err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	// Send answer local description.
	sdp := webrtcSdp2pbSdp(peerConnection.LocalDescription())
	if actor == peerPublisher {
		s.publisherChans.AnswerChan <- sdp
	} else {
		s.subscriberChans.answerChan <- sdp
	}

	return nil
}

// sendRTCP sends a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it
func sendRTCP(peerConnection *webrtc.PeerConnection, remoteTrack *webrtc.TrackRemote, logger *zerolog.Logger) {
	ticker := time.NewTicker(rtcpPLIInterval)
	for range ticker.C {
		if rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{
			&rtcp.PictureLossIndication{
				MediaSSRC: uint32(remoteTrack.SSRC()),
			},
		}); rtcpSendErr != nil {
			logger.Panic().Err(rtcpSendErr).Send()
		}
	}
}

// processRTCP reads incoming RTCP packets
// Before these packets are returned they are processed by interceptors.
// For things like NACK this needs to be called.
func processRTCP(rtpSender *webrtc.RTPSender, logger *zerolog.Logger) {
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			logger.Err(rtcpErr).Send()
			return
		}
	}
}

func pbSdp2webrtcSdp(sdp *pb.SessionDescription) webrtc.SessionDescription {
	return webrtc.SessionDescription{
		Type: webrtc.SDPType(sdp.Type),
		SDP:  string(sdp.Description),
	}
}

func webrtcSdp2pbSdp(sdp *webrtc.SessionDescription) *pb.SessionDescription {
	return &pb.SessionDescription{
		Type:        int32(sdp.Type),
		Description: []byte(sdp.SDP),
	}
}
