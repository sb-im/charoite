package webrtc

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"

	"github.com/SB-IM/skywalker/internal/broadcast/cfg"
)

// SendCandidateFunc sends a candidate to remote webRTC peer.
type SendCandidateFunc func(candidate *webrtc.ICECandidate) error

// RecvCandidateFunc receives a candidate from remote webRTC peer.
type RecvCandidateFunc func() <-chan string

// RegisterSessionFunc registers a edge WebRTC session. Only used for publisher.
// For subscriber, it should use NoopRegisterSessionFunc instead.
type RegisterSessionFunc func()

// HookStreamFunc hooks the stream seeding source on peer connection established.
type HookStreamFunc func()

const (
	rtcpPLIInterval = time.Second * 3
)

type WebRTC struct {
	logger zerolog.Logger
	config cfg.WebRTCConfigOptions

	// SignalChan is a bi-direction channel.
	SignalChan chan *webrtc.SessionDescription

	pendingCandidates []*webrtc.ICECandidate
	candidatesMux     sync.Mutex

	sendCandidate SendCandidateFunc
	recvCandidate RecvCandidateFunc

	registerSession RegisterSessionFunc

	hookStream HookStreamFunc
}

// New returns a new WebRTC.
func New(
	config cfg.WebRTCConfigOptions,
	logger *zerolog.Logger,
	sendCandidate SendCandidateFunc,
	recvCandidate RecvCandidateFunc,
	registerSession RegisterSessionFunc,
	hookStream HookStreamFunc,
) *WebRTC {
	return &WebRTC{
		logger:          *logger,
		config:          config,
		SignalChan:      make(chan *webrtc.SessionDescription, 1), // Make 1 buffer so SDP signaling never blocks
		sendCandidate:   sendCandidate,
		recvCandidate:   recvCandidate,
		registerSession: registerSession,
		hookStream:      hookStream,
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
				w.logger.Err(err).Msg("could not read buffer")
				return
			}
			// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
			if _, err = videoTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				w.logger.Err(err).Msg("could not write video track")
				return
			}
		}
	})

	if err := w.signalPeerConnection(peerConnection); err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}
	w.logger.Info().Msg("created peer connection for publisher")

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
	w.logger.Info().Msg("created peer connection for subscriber")

	return nil
}

func (w *WebRTC) signalPeerConnection(peerConnection *webrtc.PeerConnection) error {
	offer := <-w.SignalChan
	candidateChan := w.recvCandidate()

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		w.candidatesMux.Lock()
		defer w.candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			w.pendingCandidates = append(w.pendingCandidates, c)
			return
		}
		if err := w.sendCandidate(c); err != nil {
			w.logger.Err(err).Msg("could not send candidate")
		}
		w.logger.Info().Msg("sent an ICE candidate")
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		w.logger.Info().Str("state", connectionState.String()).Msg("ICE connection state has changed")

		switch connectionState {
		case webrtc.ICEConnectionStateFailed:
			if err := closePeerConnection(peerConnection); err != nil {
				w.logger.Panic().Err(err).Msg("could not close peer connection")
			}
			w.logger.Info().Msg("peer connection has been closed")
		case webrtc.ICEConnectionStateConnected:
			// Hook video seeding source here.
			w.hookStream()
			// Register session after ICE state is connected.
			w.registerSession()
		default:
		}
	})

	if err := peerConnection.SetRemoteDescription(*offer); err != nil {
		return fmt.Errorf("could not set remote description: %w", err)
	}

	// Signal candidate after setting remote description.
	go w.signalCandidate(peerConnection, candidateChan)

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("could not create answer: %w", err)
	}

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("could not set local description: %w", err)
	}

	// Send answer of local description.
	// This is a universal answer for both publisher and subscriber in protobuf format.
	w.SignalChan <- peerConnection.LocalDescription()

	// Signal candidate
	w.candidatesMux.Lock()
	defer w.candidatesMux.Unlock()

	for _, c := range w.pendingCandidates {
		if err := w.sendCandidate(c); err != nil {
			return fmt.Errorf("could not send candidate: %w", err)
		}
		w.logger.Info().Msg("sent an ICE candidate")
	}

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

func (w *WebRTC) signalCandidate(peerConnection *webrtc.PeerConnection, ch <-chan string) {
	// TODO: Stop adding ICE candidate when after signaling succeeded, that is, to exit the loop.
	// Just set a timer is not enough.
	for c := range ch {
		if err := peerConnection.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: c,
		}); err != nil {
			w.logger.Err(err).Msg("could not add ICE candidate")
		}
		w.logger.Info().Str("candidate", c).Msg("successfully added an ICE candidate")
	}
}

// closePeerConnection tidies RTPSender and remvoes track from peer connection.
// It's used after a subscriber peer connection fails.
// A publisher calls this has no effect.
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
			w.logger.Err(rtcpSendErr).Send()
			return
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
			if errors.Is(rtcpErr, io.EOF) || errors.Is(rtcpErr, io.ErrClosedPipe) {
				_ = rtpSender.Stop()
			} else {
				w.logger.Err(rtcpErr).Send()
			}
			return
		}
	}
}

// NoopSendCandidateFunc does nothing.
func NoopSendCandidateFunc(_ *webrtc.ICECandidate) error {
	return nil
}

// NoopRecvCandidateFunc does nothing.
func NoopRecvCandidateFunc() <-chan string {
	ch := make(chan string)
	close(ch)
	return ch
}

// NoopRegisterSessionFunc does nothing.
func NoopRegisterSessionFunc() {}

// NoopHookStreamFunc does nothing.
func NoopHookStreamFunc() {}
