package publisher

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/SB-IM/skywalker/internal/broadcast/cfg"
	webrtcx "github.com/SB-IM/skywalker/internal/broadcast/webrtc"
)

// Publisher stands for a publisher webRTC peer.
type Publisher struct {
	client mqtt.Client
	logger zerolog.Logger
	config *cfg.PublisherConfigOptions

	// sessions must be created before used by publisher and is shared between publishers and subscribers.
	// It's mainly written and maintained by publishers
	sessions *sync.Map
}

// New returns a new Publisher.
func New(
	client mqtt.Client,
	sessions *sync.Map,
	logger *zerolog.Logger,
	config *cfg.PublisherConfigOptions,
) *Publisher {
	l := logger.With().Str("component", "Publisher").Logger()
	return &Publisher{
		client:   client,
		logger:   l,
		config:   config,
		sessions: sessions,
	}
}

// Signal performs webRTC signaling for all publisher peers.
func (p *Publisher) Signal() {
	// The receiving topic is the same for each edge device, but message payload is different.
	// The id and trackSource in payload determine the following publishing topic.
	// Receive remote SDP with MQTT.
	t := p.client.Subscribe(p.config.OfferTopic, byte(p.config.Qos), p.handleMessage())
	// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
	// in other handlers does cause problems its best to just assume we should not block
	go func() {
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not subscribe to %s", p.config.OfferTopic)
		} else {
			p.logger.Info().Msgf("subscribed to %s", p.config.OfferTopic)
		}
	}()
}

// sendCandidate sends candidate to remote webRTC peer via MQTT.
// The publish topic is unique to this edge device.
func (p *Publisher) sendCandidate(meta *pb.Meta) webrtcx.SendCandidateFunc {
	return func(candidate *webrtc.ICECandidate) error {
		payload, err := pb.EncodeCandidate(candidate)
		if err != nil {
			return fmt.Errorf("could not encode candidate: %w", err)
		}
		topic := p.config.CandidateSendTopicPrefix + "/" + meta.Id + "/" + strconv.Itoa(int(meta.TrackSource))
		t := p.client.Publish(topic, byte(p.config.Qos), p.config.Retained, payload)
		// Handle the token in a go routine so this loop keeps sending messages regardless of delivery status
		go func() {
			<-t.Done()
			if t.Error() != nil {
				p.logger.Err(t.Error()).Msgf("could not publish to %s", topic)
			}
		}()
		return nil
	}
}

// recvCandidate is not a one time subscriber.
// The caller must check if result in channel is nil.
// sendCandidate receive candidate from remote webRTC peer via MQTT.
// The subscription topic is unique to this edge device.
func (p *Publisher) recvCandidate(meta *pb.Meta) webrtcx.RecvCandidateFunc {
	return func() <-chan string {
		// TODO: Figure how to properly close channel.
		ch := make(chan string, 2) // Make buffer 2 because we have at least 2 sendings.
		topic := p.config.CandidateRecvTopicPrefix + "/" + meta.Id + "/" + strconv.Itoa(int(meta.TrackSource))
		// Receive remote ICE candidate with MQTT.
		t := p.client.Subscribe(topic, byte(p.config.Qos), func(c mqtt.Client, m mqtt.Message) {
			candidate, err := pb.DecodeCandidate(m.Payload())
			if err != nil {
				p.logger.Err(err).Msg("could not decode candidate")
				return
			}
			ch <- candidate
		})
		// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
		// in other handlers does cause problems its best to just assume we should not block
		go func() {
			<-t.Done()
			if t.Error() != nil {
				p.logger.Err(t.Error()).Msgf("could not subscribe to %s", topic)
			} else {
				p.logger.Info().Msgf("subscribed to %s", topic)
			}
		}()
		return ch
	}
}

// handleMessage handles MQTT subscription message.
func (p *Publisher) handleMessage() mqtt.MessageHandler {
	return func(c mqtt.Client, m mqtt.Message) {
		var offer pb.SessionDescription
		if err := proto.Unmarshal(m.Payload(), &offer); err != nil {
			p.logger.Err(err).Msg("could not unmarshal sdp")
			return
		}

		logger := p.logger.With().
			Str("offer_topic", p.config.OfferTopic).
			Str("id", offer.Meta.Id).
			Int32("track_source", int32(offer.Meta.TrackSource)).
			Logger()
		logger.Info().Msg("received offer from edge")

		answer, err := p.signalPeerConnection(&offer, &logger)
		if err != nil {
			logger.Err(err).Msg("failed to signal peer connection")
			return
		}
		logger.Info().Msg("Successfully signaled peer connection")

		payload, err := pb.EncodeSDP(answer, nil)
		if err != nil {
			logger.Err(err).Msg("could not encode sdp")
			return
		}

		// The publishing topic is unique to each edge device and is determined by above receiving message payload.
		answerTopic := p.config.AnswerTopicPrefix + "/" + offer.Meta.Id + "/" + strconv.Itoa(int(offer.Meta.TrackSource))
		t := c.Publish(answerTopic, byte(p.config.Qos), p.config.Retained, payload)
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not publish to %s", answerTopic)
			return
		}
		logger.Info().Str("answer_topic", answerTopic).Msg("sent answer to edge")
	}
}

// signalPeerConnection creates video track and performs webRTC signaling.
func (p *Publisher) signalPeerConnection(offer *pb.SessionDescription, logger *zerolog.Logger) (
	*webrtc.SessionDescription,
	error,
) {
	var sdp webrtc.SessionDescription
	if err := json.Unmarshal([]byte(offer.Sdp), &sdp); err != nil {
		return nil, err
	}

	videoTrack, err := webrtcx.CreateLocalTrack()
	if err != nil {
		return nil, fmt.Errorf("could not create webRTC local video track: %w", err)
	}
	logger.Info().Msg("created video track")

	w := webrtcx.New(
		p.config.WebRTCConfigOptions,
		logger,
		p.sendCandidate(offer.Meta),
		p.recvCandidate(offer.Meta),
		p.registerSession(offer.Meta, videoTrack),
		webrtcx.NoopHookStreamFunc,
	)

	// TODO: handle blocking case with timeout for channels.
	w.SignalChan <- &sdp
	if err := w.CreatePublisher(videoTrack); err != nil {
		return nil, fmt.Errorf("failed to create webRTC publisher: %w", err)
	}
	logger.Info().Msg("created publisher")

	return <-w.SignalChan, nil
}

func (p *Publisher) registerSession(
	meta *pb.Meta,
	videoTrack *webrtc.TrackLocalStaticRTP,
) webrtcx.RegisterSessionFunc {
	return func() {
		sessionID := meta.Id + strconv.Itoa(int(meta.TrackSource))
		_, ok := p.sessions.Load(sessionID)
		p.sessions.Store(sessionID, videoTrack)
		if ok {
			p.logger.Info().Str("key", sessionID).Int32("value", int32(meta.TrackSource)).Msg("re-registered old session")
		} else {
			p.logger.Info().Str("key", sessionID).Int32("value", int32(meta.TrackSource)).Msg("registered session")
		}
	}
}
