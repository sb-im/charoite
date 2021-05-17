package broadcast

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	mqttclient "github.com/SB-IM/mqtt-client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/SB-IM/skywalker/internal/broadcast/cfg"
	"github.com/SB-IM/skywalker/internal/broadcast/publisher"
	"github.com/SB-IM/skywalker/internal/broadcast/subscriber"
)

// Service consists of many sessions.
type Service struct {
	client   mqtt.Client
	logger   zerolog.Logger
	config   cfg.ConfigOptions
	sessions sync.Map
}

func New(ctx context.Context, config *cfg.ConfigOptions) *Service {
	return &Service{
		client: mqttclient.FromContext(ctx),
		logger: *log.Ctx(ctx),
		config: *config,
	}
}

func (s *Service) Broadcast() error {
	pub := publisher.New(s.client, &s.sessions, &s.logger, &cfg.PublisherConfigOptions{
		MQTTClientConfigOptions: s.config.MQTTClientConfigOptions,
		WebRTCConfigOptions:     s.config.WebRTCConfigOptions,
	})
	pub.Signal()

	sub := subscriber.New(&s.sessions, &s.logger, s.config.WebRTCConfigOptions)
	handler := sub.Signal()

	server := s.newServer(handler)
	s.logger.Info().Str("host", s.config.Host).Int("port", s.config.Port).Msg("starting HTTP server")
	return server.ListenAndServe()
}

func (s *Service) newServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler: handler,
		Addr:    s.config.Host + ":" + strconv.Itoa(s.config.Port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}
