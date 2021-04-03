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
	"github.com/SB-IM/skywalker/internal/broadcast/session"
	webrtcx "github.com/SB-IM/skywalker/internal/broadcast/webrtc"
)

// Publisher stands for a publisher webRTC peer.
type Publisher struct {
	client mqtt.Client
	logger zerolog.Logger
	config cfg.PublisherConfigOptions

	// sessions must be created before used by publisher and is shared between publishers and subscribers.
	// It's mainly written and maintained by publishers
	sessions *sync.Map
}

// New returns a new Publisher.
func New(
	client mqtt.Client,
	sessions *sync.Map,
	logger *zerolog.Logger,
	config cfg.PublisherConfigOptions,
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
	t := p.client.Subscribe(p.config.OfferTopic, 1, p.handleMessage())
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
		logger.Debug().Str("offer", offer.String()).Msg("received offer from edge")

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
		answerTopic := p.config.AnswerTopic + "/" + offer.Id + "/" + strconv.Itoa(int(offer.TrackSource))
		t := c.Publish(answerTopic, 1, true, payload)
		<-t.Done()
		if t.Error() != nil {
			p.logger.Err(t.Error()).Msgf("could not publish to %s", answerTopic)
			return
		}
		logger.Debug().Str("answer_topic", answerTopic).Str("answer", answer.String()).Msg("sent answer to edge")

		// Register session on signaling success.
		p.registerSession(session.MachineID(offer.Id), offer.TrackSource, videoTrack)
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

	w := webrtcx.New(videoTrack, p.config.WebRTCConfigOptions, logger)

	// TODO: handle blocking case with timeout for channels.
	w.OfferChan <- offer
	if err := w.CreatePublisher(); err != nil {
		return nil, nil, fmt.Errorf("failed to create webRTC publisher: %w", err)
	}
	logger.Debug().Msg("created publisher")

	return <-w.AnswerChan, videoTrack, nil
}

func (p *Publisher) registerSession(
	id session.MachineID,
	trackSource pb.TrackSource,
	videoTrack *webrtc.TrackLocalStaticRTP,
) {
	if s, ok := p.sessions.Load(id); ok {
		s.(session.Session)[trackSource] = videoTrack
	} else {
		s := make(session.Session)
		s[trackSource] = videoTrack
		p.sessions.Store(id, s)
	}
}
