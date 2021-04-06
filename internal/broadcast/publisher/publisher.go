package publisher

import (
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
func (p *Publisher) sendCandidate(id string, trackSource pb.TrackSource) webrtcx.SendCandidateFunc {
	return func(candidate *webrtc.ICECandidate) error {
		payload, err := encodeCandidate(candidate)
		if err != nil {
			return fmt.Errorf("could not encode candidate: %w", err)
		}
		topic := p.config.CandidateSendTopicSuffix + "/" + id + "/" + strconv.Itoa(int(trackSource))
		t := p.client.Publish(topic, byte(p.config.Qos), p.config.Retained, payload)
		// Handle the token in a go routine so this loop keeps sending messages regardless of delivery status
		go func() {
			<-t.Done()
			if t.Error() != nil {
				p.logger.Err(t.Error()).Msgf("could not publish to %s", p.config.CandidateSendTopicSuffix)
			}
		}()
		return nil
	}
}

// recvCandidate is not a one time subscriber.
// The caller must check if result in channel is nil.
// sendCandidate receive candidate from remote webRTC peer via MQTT.
// The subscription topic is unique to this edge device.
func (p *Publisher) recvCandidate(id string, trackSource pb.TrackSource) webrtcx.RecvCandidateFunc {
	return func() <-chan *webrtc.ICECandidate {
		// TODO: Figure how to properly close channel.
		ch := make(chan *webrtc.ICECandidate, 2) // Make buffer 2 because we have at least 2 sendings.
		topic := p.config.CandidateRecvTopicSuffix + "/" + id + "/" + strconv.Itoa(int(trackSource))
		// Receive remote ICE candidate with MQTT.
		t := p.client.Subscribe(topic, byte(p.config.Qos), func(c mqtt.Client, m mqtt.Message) {
			candidate, err := decodeCandidate(m.Payload())
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
			Str("id", offer.Id).
			Int32("track_source", int32(offer.TrackSource)).
			Logger()
		logger.Debug().Msg("received offer from edge")

		answer, videoTrack, err := p.signalPeerConnection(&offer, &logger)
		if err != nil {
			logger.Err(err).Msg("failed to signal peer connection")
			return
		}
		logger.Debug().Msg("Successfully signaled peer connection")

		payload, err := proto.Marshal(answer)
		if err != nil {
			logger.Err(err).Msg("could not encode sdp")
			return
		}

		// The publishing topic is unique to each edge device and is determined by above receiving message payload.
		answerTopic := p.config.AnswerTopicSuffix + "/" + offer.Id + "/" + strconv.Itoa(int(offer.TrackSource))
		t := c.Publish(answerTopic, byte(p.config.Qos), p.config.Retained, payload)
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not publish to %s", answerTopic)
			return
		}
		logger.Debug().Str("answer_topic", answerTopic).Msg("sent answer to edge")

		// Register session on signaling success.
		p.registerSession(offer.Id, offer.TrackSource, videoTrack)
	}
}

// signalPeerConnection creates video track and performs webRTC signaling.
func (p *Publisher) signalPeerConnection(offer *pb.SessionDescription, logger *zerolog.Logger) (
	*pb.SessionDescription,
	*webrtc.TrackLocalStaticRTP,
	error,
) {
	videoTrack, err := webrtcx.CreateLocalTrack()
	if err != nil {
		return nil, nil, fmt.Errorf("could not create webRTC local video track: %w", err)
	}
	logger.Debug().Msg("created video track")

	w := webrtcx.New(
		p.config.WebRTCConfigOptions,
		logger,
		p.sendCandidate(offer.Id, offer.TrackSource),
		p.recvCandidate(offer.Id, offer.TrackSource),
	)

	// TODO: handle blocking case with timeout for channels.
	w.OfferChan <- offer
	if err := w.CreatePublisher(videoTrack); err != nil {
		return nil, nil, fmt.Errorf("failed to create webRTC publisher: %w", err)
	}
	logger.Debug().Msg("created publisher")

	return <-w.AnswerChan, videoTrack, nil
}

func (p *Publisher) registerSession(
	id string,
	trackSource pb.TrackSource,
	videoTrack *webrtc.TrackLocalStaticRTP,
) {
	sessionID := id + strconv.Itoa(int(trackSource))
	p.sessions.Store(sessionID, videoTrack)
	p.logger.Debug().Str("key", sessionID).Int32("value", int32(trackSource)).Msg("registered session")
}

func encodeCandidate(candidate *webrtc.ICECandidate) ([]byte, error) {
	msg := pb.ICECandidate{
		Foundation:     candidate.Foundation,
		Priority:       candidate.Priority,
		Address:        candidate.Address,
		Protocol:       int32(candidate.Protocol),
		Port:           uint32(candidate.Port),
		Type:           int32(candidate.Typ),
		Component:      uint32(candidate.Typ),
		RelatedAddress: candidate.RelatedAddress,
		RelatedPort:    uint32(candidate.RelatedPort),
		TcpType:        candidate.TCPType,
	}
	return proto.Marshal(&msg)
}

func decodeCandidate(payload []byte) (*webrtc.ICECandidate, error) {
	var candidate pb.ICECandidate
	if err := proto.Unmarshal(payload, &candidate); err != nil {
		return nil, err
	}
	return &webrtc.ICECandidate{
		Foundation:     candidate.Foundation,
		Priority:       candidate.Priority,
		Address:        candidate.Address,
		Protocol:       webrtc.ICEProtocol(candidate.Protocol),
		Port:           uint16(candidate.Port),
		Typ:            webrtc.ICECandidateType(candidate.Type),
		Component:      uint16(candidate.Component),
		RelatedAddress: candidate.RelatedAddress,
		RelatedPort:    uint16(candidate.RelatedPort),
		TCPType:        candidate.TcpType,
	}, nil
}
