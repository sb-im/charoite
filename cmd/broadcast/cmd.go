package broadcast

import (
	"context"
	"time"

	"github.com/SB-IM/logging"
	mqttclient "github.com/SB-IM/mqtt-client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/SB-IM/skywalker/internal/broadcast"
)

// Command returns a broadcast command.
func Command() *cli.Command {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		logger zerolog.Logger

		mc mqtt.Client

		mqttConfigOptions     mqttclient.ConfigOptions
		topicConfigOptions    broadcast.TopicConfigOptions
		webRTCConfigOptions   broadcast.WebRTCConfigOptions
		wsServerConfigOptions broadcast.WSServerConfigOptions
	)

	flags := func() (flags []cli.Flag) {
		for _, v := range [][]cli.Flag{
			loadConfigFlag(),
			mqttFlags(&mqttConfigOptions),
			topicFlags(&topicConfigOptions),
			webRTCFlags(&webRTCConfigOptions),
			wsFlags(&wsServerConfigOptions),
		} {
			flags = append(flags, v...)
		}
		return
	}()

	return &cli.Command{
		Name:  "broadcast",
		Usage: "broadcast live stream from Sphinx edge to users",
		Flags: flags,
		Before: func(c *cli.Context) error {
			if err := altsrc.InitInputSourceWithContext(
				flags,
				altsrc.NewYamlSourceFromFlagFunc("config"),
			)(c); err != nil {
				return err
			}

			// Set up logger.
			debug := c.Bool("debug")
			logging.SetDebugMod(debug)
			logger = log.With().Str("service", "skywalker").Str("command", "broadcast").Logger()
			ctx = logger.WithContext(ctx)

			// Initializes MQTT client.
			mc = mqttclient.NewClient(ctx, mqttConfigOptions)
			if err := mqttclient.CheckConnectivity(mc, 3*time.Second); err != nil {
				return err
			}
			ctx = mqttclient.WithContext(ctx, mc)
			return nil
		},
		Action: func(c *cli.Context) error {
			svc := broadcast.New(ctx, &broadcast.ConfigOptions{
				WebRTCConfigOptions:   webRTCConfigOptions,
				TopicConfigOptions:    topicConfigOptions,
				WSServerConfigOptions: wsServerConfigOptions,
			})
			err := svc.Broadcast()
			if err != nil {
				logger.Err(err).Msg("broadcast failed")
			}
			return err
		},
		After: func(c *cli.Context) error {
			logger.Info().Msg("exits")
			return nil
		},
	}
}

// loadConfigFlag sets a config file path for app command.
// Note: you can't set any other flags' `Required` value to `true`,
// As it conflicts with this flag. You can set only either this flag or specifically the other flags but not both.
func loadConfigFlag() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Usage:       "Config file path",
			Value:       "config/config.yaml",
			DefaultText: "config/config.yaml",
		},
	}
}

func mqttFlags(mqttConfigOptions *mqttclient.ConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt-server",
			Usage:       "MQTT server address",
			Value:       "tcp://mosquitto:1883",
			DefaultText: "tcp://mosquitto:1883",
			Destination: &mqttConfigOptions.Server,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt-clientID",
			Usage:       "MQTT client id",
			Value:       "mqtt_edge",
			DefaultText: "mqtt_edge",
			Destination: &mqttConfigOptions.ClientID,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt-username",
			Usage:       "MQTT broker username",
			Value:       "",
			Destination: &mqttConfigOptions.Username,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt-password",
			Usage:       "MQTT broker password",
			Value:       "",
			Destination: &mqttConfigOptions.Password,
		}),
	}
}

func topicFlags(topicConfigOptions *broadcast.TopicConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "topic-offer",
			Usage:       "MQTT topic for WebRTC SDP offer signaling",
			Value:       "/edge/livestream/signal/offer",
			DefaultText: "/edge/livestream/signal/offer",
			Destination: &topicConfigOptions.OfferTopic,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "topic-answer",
			Usage:       "MQTT topic for WebRTC SDP answer signaling",
			Value:       "/edge/livestream/signal/answer",
			DefaultText: "/edge/livestream/signal/answer",
			Destination: &topicConfigOptions.AnswerTopic,
		}),
	}
}

func webRTCFlags(webRTCConfigOptions *broadcast.WebRTCConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "ice-server",
			Usage:       "ICE server address for webRTC",
			Value:       "stun:stun.l.google.com:19302",
			DefaultText: "stun:stun.l.google.com:19302",
			Destination: &webRTCConfigOptions.ICEServer,
		}),
	}
}

func wsFlags(wsServerConfigOptions *broadcast.WSServerConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "ws-host",
			Usage:       "Host of WebSocket server",
			Value:       "0.0.0.0",
			DefaultText: "0.0.0.0",
			Destination: &wsServerConfigOptions.Host,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "ws-port",
			Usage:       "Port of WebSocket server",
			Value:       8080,
			DefaultText: "8080",
			Destination: &wsServerConfigOptions.Port,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "ws-path",
			Usage:       "HTTP path of broadcast service",
			Value:       "/ws/webrtc",
			DefaultText: "/ws/webrtc",
			Destination: &wsServerConfigOptions.Path,
		}),
	}
}
