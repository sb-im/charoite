package broadcast

import (
	"strconv"

	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

type publisherChans struct {
	OfferChan  chan *pb.SessionDescription
	AnswerChan chan *pb.SessionDescription
}

type publisher struct {
	client         mqtt.Client
	publisherChans publisherChans
	logger         zerolog.Logger
	config         TopicConfigOptions
}

func newPublisher(client mqtt.Client, logger *zerolog.Logger, config TopicConfigOptions) *publisher {
	return &publisher{
		client: client,
		publisherChans: publisherChans{ // The channel buffer size limits concurrency
			OfferChan:  make(chan *pb.SessionDescription, 1),
			AnswerChan: make(chan *pb.SessionDescription, 1),
		},
		logger: *logger,
		config: config,
	}
}

func (p *publisher) Signal() {
	p.logger = p.logger.With().Str("component", "publisher").Logger()

	// The receiving topic is the same for each edge device, but message payload is different.
	// The id and trackSource in payload determine the following publishing topic.
	// Receive remote description with MQTT.
	t := p.client.Subscribe(p.config.OfferTopic, 1, func(c mqtt.Client, msg mqtt.Message) {
		var offer pb.SessionDescription
		if err := proto.Unmarshal(msg.Payload(), &offer); err != nil {
			p.logger.Fatal().Err(err).Msg("could not unmarshal sdp")
		}
		p.logger.Debug().
			Str("id", offer.Id).
			Str("topic", p.config.OfferTopic).
			Int32("track_source", int32(offer.TrackSource)).
			Msg("received offer from edge")
		p.publisherChans.OfferChan <- &offer

		answer := <-p.publisherChans.AnswerChan
		payload, err := proto.Marshal(answer)
		if err != nil {
			p.logger.Fatal().Err(err).Msg("could not encode sdp")
		}

		answerTopic := p.config.AnswerTopic + "/" + offer.Id + "/" + strconv.Itoa(int(offer.TrackSource))
		// The publishing topic is unique to each edge device and is determined by above receiving message payload.
		t := p.client.Publish(answerTopic, 1, true, payload)
		// Handle the token in a go routine so this loop keeps sending messages regardless of delivery status
		go func() {
			<-t.Done()
			if t.Error() != nil {
				p.logger.Fatal().Err(t.Error()).Msgf("could not publish to %s", answerTopic)
			} else {
				p.logger.Debug().Str("topic", answerTopic).Msg("sent answer to edge")
			}
		}()
	})
	// the connection handler is called in a goroutine so blocking here would hot cause an issue. However as blocking
	// in other handlers does cause problems its best to just assume we should not block
	go func() {
		<-t.Done()
		if t.Error() != nil {
			p.logger.Fatal().Err(t.Error()).Msgf("could not subscribe to %s", p.config.OfferTopic)
		} else {
			p.logger.Info().Msgf("subscribed to %s", p.config.OfferTopic)
		}
	}()
}
