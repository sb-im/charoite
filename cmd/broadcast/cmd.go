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
	"github.com/SB-IM/skywalker/internal/broadcast/cfg"
)

const configFlagName = "config"

// Command returns a broadcast command.
func Command() *cli.Command {
	ctx := context.Background()

	var (
		logger zerolog.Logger

		mc mqtt.Client

		mqttConfigOptions       mqttclient.ConfigOptions
		mqttClientConfigOptions cfg.MQTTClientConfigOptions
		webRTCConfigOptions     cfg.WebRTCConfigOptions
		serverConfigOptions     cfg.ServerConfigOptions
	)

	flags := func() (flags []cli.Flag) {
		for _, v := range [][]cli.Flag{
			loadConfigFlag(),
			mqttFlags(&mqttConfigOptions),
			mqttClientFlags(&mqttClientConfigOptions),
			webRTCFlags(&webRTCConfigOptions),
			serverFlags(&serverConfigOptions),
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
				altsrc.NewTomlSourceFromFlagFunc(configFlagName),
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
			svc := broadcast.New(ctx, &cfg.ConfigOptions{
				WebRTCConfigOptions:     webRTCConfigOptions,
				MQTTClientConfigOptions: mqttClientConfigOptions,
				ServerConfigOptions:     serverConfigOptions,
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
			Name:        configFlagName,
			Aliases:     []string{"c"},
			Usage:       "Config file path",
			Value:       "config/config.toml",
			DefaultText: "config/config.toml",
		},
	}
}

func mqttFlags(options *mqttclient.ConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.server",
			Usage:       "MQTT server address",
			Value:       "tcp://mosquitto:1883",
			DefaultText: "tcp://mosquitto:1883",
			Destination: &options.Server,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.clientID",
			Usage:       "MQTT client id",
			Value:       "mqtt_edge",
			DefaultText: "mqtt_edge",
			Destination: &options.ClientID,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.username",
			Usage:       "MQTT broker username",
			Value:       "",
			Destination: &options.Username,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.password",
			Usage:       "MQTT broker password",
			Value:       "",
			Destination: &options.Password,
		}),
	}
}

func mqttClientFlags(options *cfg.MQTTClientConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_offer_prefix",
			Usage:       "MQTT topic prefix for WebRTC SDP offer signaling",
			Value:       "/edge/livestream/signal/offer",
			DefaultText: "/edge/livestream/signal/offer",
			Destination: &options.OfferTopicPrefix,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_answer_prefix",
			Usage:       "MQTT topic prefix for WebRTC SDP answer signaling",
			Value:       "/edge/livestream/signal/answer",
			DefaultText: "/edge/livestream/signal/answer",
			Destination: &options.AnswerTopicPrefix,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_candidate_send_prefix",
			Usage:       "MQTT topic prefix for WebRTC candidate sending, and the sending topic of edge is /edge/livestream/signal/candidate/send",
			Value:       "/edge/livestream/signal/candidate/recv",
			DefaultText: "/edge/livestream/signal/candidate/recv",
			Destination: &options.CandidateSendTopicPrefix,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_candidate_recv_prefix",
			Usage:       "MQTT topic prefix for WebRTC candidate receiving, and the receiving topic of edge is /edge/livestream/signal/candidate/recv", //nolint:lll
			Value:       "/edge/livestream/signal/candidate/send",
			DefaultText: "/edge/livestream/signal/candidate/send",
			Destination: &options.CandidateRecvTopicPrefix,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_hook_stream_prefix",
			Usage:       "MQTT topic prefix for hooking of seeding stream",
			Value:       "/edge/livestream/hook",
			DefaultText: "/edge/livestream/hook",
			Destination: &options.HookStreamTopicPrefix,
		}),
		altsrc.NewUintFlag(&cli.UintFlag{
			Name:        "mqtt_client.qos",
			Usage:       "MQTT client qos for WebRTC SDP signaling",
			Value:       0,
			DefaultText: "0",
			Destination: &options.Qos,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "mqtt_client.retained",
			Usage:       "MQTT client setting retention for WebRTC SDP signaling",
			Value:       false,
			DefaultText: "false",
			Destination: &options.Retained,
		}),
	}
}

func webRTCFlags(options *cfg.WebRTCConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "webrtc.ice_server",
			Usage:       "ICE server address for webRTC",
			Value:       "stun:stun.l.google.com:19302",
			DefaultText: "stun:stun.l.google.com:19302",
			Destination: &options.ICEServer,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "webrtc.ice_server_username",
			Usage:       "ICE server username",
			Value:       "",
			DefaultText: "",
			Destination: &options.Username,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "webrtc.ice_server_credential",
			Usage:       "ICE server credential",
			Value:       "",
			DefaultText: "",
			Destination: &options.Credential,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "webrtc.enable_frontend",
			Usage:       "Enable webRTC frontend server",
			Value:       false,
			DefaultText: "false",
			Destination: &options.EnableFrontend,
		}),
	}
}

func serverFlags(options *cfg.ServerConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "signal_server.host",
			Usage:       "Host of webRTC signaling server",
			Value:       "0.0.0.0",
			DefaultText: "0.0.0.0",
			Destination: &options.Host,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "signal_server.port",
			Usage:       "Port of webRTC signaling server",
			Value:       8080,
			DefaultText: "8080",
			Destination: &options.Port,
		}),
	}
}
