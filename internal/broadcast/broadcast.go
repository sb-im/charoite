// broadcast is an event driven video broadcasting system.
// It's composed of a model of publisher, transfer and subscribers.
// The connection between local peer and remote publisher peer can be called a half session and is valid and should be established first.
// The connection between local peer and remote subscriber peer can also be called a half session but may not be valid.
// A whole session is made up of three peers. In fact, every publisher fires an event that starts a unique session.
// And every subscriber fires an event that starts their own half session.
package broadcast

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	mqttclient "github.com/SB-IM/mqtt-client"
	pb "github.com/SB-IM/pb/signal"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type subscriber map[pb.TrackSource]*subscriberChans

// Service consists of many sessions.
type Service struct {
	sessions map[machineID]subscriber
	client   mqtt.Client
	logger   zerolog.Logger
	config   ConfigOptions
}

func New(ctx context.Context, config ConfigOptions) *Service {
	return &Service{
		sessions: make(map[machineID]subscriber),
		client:   mqttclient.FromContext(ctx),
		logger:   *log.Ctx(ctx),
		config:   config,
	}
}

// Broadcast broadcasts video streams following publisher -> transfer -> subscribers flow direction.
func (svc *Service) Broadcast() error {
	// Register Websockets handler.
	http.HandleFunc(svc.config.WSServerConfigOptions.Path, svc.handleSubscription())
	go func() {
		// Start HTTP server.
		svc.logger.Info().Str("host",
			svc.config.WSServerConfigOptions.Host).Int("port",
			svc.config.WSServerConfigOptions.Port,
		).Msg("starting HTTP server for WebSocket")
		svc.logger.Fatal().Err(http.ListenAndServe(
			svc.config.WSServerConfigOptions.Host+":"+strconv.Itoa(svc.config.WSServerConfigOptions.Port),
			nil))
	}()

	// Start publisher signaling worker.
	publisher := newPublisher(svc.client, svc.logger, svc.config.TopicConfigOptions)
	publisher.Signal()

	// Use a loop to start endless broadcasting sessions.
	for {
		s := newSession(&publisher.publisherChans, svc.logger, svc.config.WebRTCConfigOptions)

		// You can get id and track source only when half of session (publisher session) completes.
		// Therefore, you must start session first.
		if err := s.start(func(id machineID, t pb.TrackSource, s *subscriberChans) {
			inner := make(subscriber)
			inner[t] = s
			svc.sessions[id] = inner
		}); err != nil {
			return fmt.Errorf("session failed: %w", err)
		}
	}
}

// handleSubscription is called every time after publisher session and at the start of subscriber session.
// If the running order is wrong, it blocks forever.
func (svc *Service) handleSubscription() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			panic(err)
		}
		defer c.Close(websocket.StatusInternalError, "")

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		var offer pb.SessionDescription
		if err := wsjson.Read(ctx, c, &offer); err != nil {
			panic(err)
		}
		fmt.Printf("Received: %+v\n", offer.Id)

		// The subscriber's sdp id must be equal to session's id.
		var offerChan chan *pb.SessionDescription
		var answerChan chan *pb.SessionDescription
		if inner, ok := svc.sessions[machineID(offer.Id)]; ok {
			// The subscriber's sdp track source must be equal to session's track source.
			if subscriber, ok := inner[offer.TrackSource]; ok {
				offerChan = subscriber.offerChan
				answerChan = subscriber.answerChan
			}
		}
		if offerChan == nil {
			if err := wsjson.Write(ctx, c, "wrong id"); err != nil {
				panic(err)
			}
		}
		offerChan <- &offer

		answer := <-answerChan
		if err := wsjson.Write(ctx, c, answer); err != nil {
			panic(err)
		}

		c.Close(websocket.StatusNormalClosure, "")
	}
}
