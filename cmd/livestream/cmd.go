package livestream

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

	"github.com/SB-IM/sphinx/internal/livestream"
)

const configFlagName = "config"

// Command returns a livestream command.
func Command() *cli.Command {
	ctx := context.Background()

	var (
		logger zerolog.Logger

		mc mqtt.Client

		mqttConfigOptions         mqttclient.ConfigOptions
		mqttClientConfigOptions   livestream.MQTTClientConfigOptions
		webRTCConfigOptions       livestream.WebRTCConfigOptions
		droneStreamConfigOptions  livestream.StreamSource
		deportStreamConfigOptions livestream.StreamSource
	)

	flags := func() (flags []cli.Flag) {
		for _, v := range [][]cli.Flag{
			loadConfigFlag(),
			mqttFlags(&mqttConfigOptions),
			mqttClientFlags(&mqttClientConfigOptions),
			webRTCFlags(&webRTCConfigOptions),
			droneStreamFlags(&droneStreamConfigOptions),
			deportStreamFlags(&deportStreamConfigOptions),
		} {
			flags = append(flags, v...)
		}
		return
	}()

	return &cli.Command{
		Name:  "livestream",
		Usage: "livestream publishes live stream from edge to cloud",
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
			logger = log.With().Str("service", "sphinx").Str("command", "livestream").Logger()
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
			// Publish live stream.
			dronePublisher := livestream.NewDronePublisher(ctx, &livestream.DroneBroadcastConfigOptions{
				MQTTClientConfigOptions: mqttClientConfigOptions,
				WebRTCConfigOptions:     webRTCConfigOptions,
				StreamSource:            droneStreamConfigOptions,
			})
			deportPublisher := livestream.NewDeportPublisher(ctx, &livestream.DeportBroadcastConfigOptions{
				MQTTClientConfigOptions: mqttClientConfigOptions,
				WebRTCConfigOptions:     webRTCConfigOptions,
				StreamSource:            deportStreamConfigOptions,
			})

			errChan := make(chan error, 2)
			for _, s := range []livestream.Livestream{dronePublisher, deportPublisher} {
				s := s
				go func() {
					if err := s.Publish(); err != nil {
						logger.Err(err).
							Str("id", s.Meta().Id).
							Int32("track_source", int32(s.Meta().TrackSource)).
							Msg(
								"live stream publishing failed")
						errChan <- err
					}
				}()
			}
			return <-errChan
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

func mqttClientFlags(options *livestream.MQTTClientConfigOptions) []cli.Flag {
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
			Usage:       "MQTT topic prefix for WebRTC candidate sending, and the sending topic of cloud is /edge/livestream/signal/candidate/recv",
			Value:       "/edge/livestream/signal/candidate/send",
			DefaultText: "/edge/livestream/signal/candidate/send",
			Destination: &options.CandidateSendTopicPrefix,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_candidate_recv_prefix",
			Usage:       "MQTT topic prefix for WebRTC candidate receiving, and the receiving topic of cloud is /edge/livestream/signal/candidate/send", //nolint:lll
			Value:       "/edge/livestream/signal/candidate/recv",
			DefaultText: "/edge/livestream/signal/candidate/recv",
			Destination: &options.CandidateRecvTopicPrefix,
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
			Usage:       "MQTT client setting retainsion for WebRTC SDP signaling",
			Value:       true,
			DefaultText: "true",
			Destination: &options.Retained,
		}),
	}
}

func webRTCFlags(options *livestream.WebRTCConfigOptions) []cli.Flag {
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
			Usage:       "ICE server username for webRTC",
			Value:       "",
			DefaultText: "",
			Destination: &options.Username,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "webrtc.ice_server_credential",
			Usage:       "ICE server credential for webRTC",
			Value:       "",
			DefaultText: "",
			Destination: &options.Credential,
		}),
	}
}

func droneStreamFlags(options *livestream.StreamSource) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "drone_stream.protocol",
			Usage:       "Protocol of drone stream source",
			Value:       "rtp",
			DefaultText: "rtp",
			Destination: &options.Protocol,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "drone_stream.host",
			Usage:       "Host of RTP server",
			Value:       "0.0.0.0",
			DefaultText: "0.0.0.0",
			Destination: &options.Host,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "drone_stream.port",
			Usage:       "Port of RTP server",
			Value:       5004,
			DefaultText: "5004",
			Destination: &options.Port,
		}),
	}
}

func deportStreamFlags(options *livestream.StreamSource) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "deport_stream.protocol",
			Usage:       "Protocol of deport stream source",
			Value:       "rtsp",
			DefaultText: "rtsp",
			Destination: &options.Protocol,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "deport_stream.addr",
			Usage:       "Address of RTSP server",
			Value:       "",
			Destination: &options.Addr,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "deport_stream.host",
			Usage:       "Host of RTP server",
			Value:       "0.0.0.0",
			DefaultText: "0.0.0.0",
			Destination: &options.Host,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "deport_stream.port",
			Usage:       "Port of RTP server",
			Value:       5005,
			DefaultText: "5005",
			Destination: &options.Port,
		}),
	}
}
