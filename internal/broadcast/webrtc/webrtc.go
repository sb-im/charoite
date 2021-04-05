package webrtc

import (
	"errors"
	"fmt"
	"io"
	"time"

	pb "github.com/SB-IM/pb/signal"
	"github.com/google/uuid"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"

	"github.com/SB-IM/skywalker/internal/broadcast/cfg"
)

const (
	rtcpPLIInterval = time.Second * 3
)

type WebRTC struct {
	logger     zerolog.Logger
	config     cfg.WebRTCConfigOptions
	OfferChan  chan *pb.SessionDescription
	AnswerChan chan *pb.SessionDescription
}

// New returns a new WebRTC.
func New(config cfg.WebRTCConfigOptions, logger *zerolog.Logger) *WebRTC {
	return &WebRTC{
		logger:     *logger,
		config:     config,
		OfferChan:  make(chan *pb.SessionDescription, 1), // Make 1 buffer so offer sending never blocks
		AnswerChan: make(chan *pb.SessionDescription, 1), // Make 1 buffer so answer sending never blocks
	}
}

// CreateLocalTrack creates a TrackLocalStaticRTP and is only used by publisher.
func CreateLocalTrack() (*webrtc.TrackLocalStaticRTP, error) {
	id := uuid.New().String()
	return webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video-"+id, "pion-"+id)
}

// CreatePublisher creates a webRTC publisher peer.
// Caller must send offer first by OfferChan or this function blocks waiting for receiving offer forever.
func (w *WebRTC) CreatePublisher(videoTrack *webrtc.TrackLocalStaticRTP) error {
	peerConnection, err := w.newPeerConnection()
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
		go w.sendRTCP(peerConnection, t)
		rtpBuf := make([]byte, 1400)
		for {
			i, _, readErr := t.Read(rtpBuf)
			if readErr != nil {
				w.logger.Panic().Err(err).Msg("could not read buffer")
			}
			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err = videoTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				w.logger.Panic().Err(err).Msg("could not write video track")
			}
		}
	})

	if err := w.signalPeerConnection(peerConnection); err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	w.logger.Debug().Msg("created peer connection for publisher")

	return nil
}

// CreateSubscriber creates a webRTC subscriber peer.
// Caller must send offer first by OfferChan or this function blocks waiting for receiving offer forever.
func (w *WebRTC) CreateSubscriber(videoTrack *webrtc.TrackLocalStaticRTP) error {
	peerConnection, err := w.newPeerConnection()
	if err != nil {
		return fmt.Errorf("could not create PeerConnection: %w", err)
	}

	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		return fmt.Errorf("could not add track: %w", err)
	}
	go w.processRTCP(rtpSender)

	if err := w.signalPeerConnection(peerConnection); err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	w.logger.Debug().Msg("created peer connection for subscriber")

	return nil
}

func (w *WebRTC) signalPeerConnection(peerConnection *webrtc.PeerConnection) error {
	offer := <-w.OfferChan

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		w.logger.Debug().Str("state", connectionState.String()).Msg("ICE connection state has changed")
		if connectionState == webrtc.ICEConnectionStateFailed {
			if err := peerConnection.Close(); err != nil {
				w.logger.Panic().Err(err).Msg("could not close peer connection")
			}
			w.logger.Debug().Msg("peer connection has been closed")
		}
	})

	if err := peerConnection.SetRemoteDescription(pbSdp2webrtcSdp(offer)); err != nil {
		return fmt.Errorf("could not set remote description: %w", err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("could not create answer: %w", err)
	}

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("could not set local description: %w", err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	// Send answer of local description.
	// This is a universal answer for both publisher and subscriber in protobuf format.
	sdp := webrtcSdp2pbSdp(peerConnection.LocalDescription())
	w.AnswerChan <- sdp

	return nil
}

func (w *WebRTC) newPeerConnection() (*webrtc.PeerConnection, error) {
	return webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{w.config.ICEServer},
				Username:   w.config.Username,
				Credential: w.config.Credential,
			},
		},
	})
}

// sendRTCP sends a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
// This can be less wasteful by processing incoming RTCP events, then we would emit a NACK/PLI when a viewer requests it
func (w *WebRTC) sendRTCP(peerConnection *webrtc.PeerConnection, remoteTrack *webrtc.TrackRemote) {
	ticker := time.NewTicker(rtcpPLIInterval)
	for range ticker.C {
		if rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{
			&rtcp.PictureLossIndication{
				MediaSSRC: uint32(remoteTrack.SSRC()),
			},
		}); rtcpSendErr != nil {
			w.logger.Panic().Err(rtcpSendErr).Send()
		}
	}
}

// processRTCP reads incoming RTCP packets
// Before these packets are returned they are processed by interceptors.
// For things like NACK this needs to be called.
func (w *WebRTC) processRTCP(rtpSender *webrtc.RTPSender) {
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
			w.logger.Err(rtcpErr).Send()
			return
		}
	}
}

func pbSdp2webrtcSdp(sdp *pb.SessionDescription) webrtc.SessionDescription {
	return webrtc.SessionDescription{
		Type: webrtc.NewSDPType(sdp.Sdp.Type),
		SDP:  sdp.Sdp.Sdp,
	}
}

func webrtcSdp2pbSdp(sdp *webrtc.SessionDescription) *pb.SessionDescription {
	return &pb.SessionDescription{
		Sdp: &pb.SDP{
			Type: sdp.Type.String(),
			Sdp:  sdp.SDP,
		},
	}
}
