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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		logger zerolog.Logger

		mc mqtt.Client

		mqttConfigOptions   mqttclient.ConfigOptions
		topicConfigOptions  livestream.MQTTClientConfigOptions
		webRTCConfigOptions livestream.WebRTCConfigOptions
		rtpConfigOptions    livestream.RTPSourceConfigOptions
		rtspConfigOptions   livestream.RTSPSourceConfigOptions
	)

	flags := func() (flags []cli.Flag) {
		for _, v := range [][]cli.Flag{
			loadConfigFlag(),
			mqttFlags(&mqttConfigOptions),
			mqttClientFlags(&topicConfigOptions),
			webRTCFlags(&webRTCConfigOptions),
			rtpFlags(&rtpConfigOptions),
			rtspFlags(&rtspConfigOptions),
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
			rtpStream := livestream.NewRTPPublisher(ctx, &livestream.RTPBroadcastConfigOptions{
				MQTTClientConfigOptions: topicConfigOptions,
				WebRTCConfigOptions:     webRTCConfigOptions,
				RTPSourceConfigOptions:  rtpConfigOptions,
			})
			rtspStream := livestream.NewRTSPPublisher(ctx, &livestream.RTSPBroadcastConfigOptions{
				MQTTClientConfigOptions: topicConfigOptions,
				WebRTCConfigOptions:     webRTCConfigOptions,
				RTSPSourceConfigOptions: rtspConfigOptions,
			})

			errChan := make(chan error, 2)
			for _, s := range []livestream.Livestream{rtpStream, rtspStream} {
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

func mqttFlags(mqttConfigOptions *mqttclient.ConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.server",
			Usage:       "MQTT server address",
			Value:       "tcp://mosquitto:1883",
			DefaultText: "tcp://mosquitto:1883",
			Destination: &mqttConfigOptions.Server,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.clientID",
			Usage:       "MQTT client id",
			Value:       "mqtt_edge",
			DefaultText: "mqtt_edge",
			Destination: &mqttConfigOptions.ClientID,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.username",
			Usage:       "MQTT broker username",
			Value:       "",
			Destination: &mqttConfigOptions.Username,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt.password",
			Usage:       "MQTT broker password",
			Value:       "",
			Destination: &mqttConfigOptions.Password,
		}),
	}
}

func mqttClientFlags(topicConfigOptions *livestream.MQTTClientConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_offer",
			Usage:       "MQTT topic for WebRTC SDP offer signaling",
			Value:       "/edge/livestream/signal/offer",
			DefaultText: "/edge/livestream/signal/offer",
			Destination: &topicConfigOptions.OfferTopic,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_answer_prefix",
			Usage:       "MQTT topic prefix for WebRTC SDP answer signaling",
			Value:       "/edge/livestream/signal/answer",
			DefaultText: "/edge/livestream/signal/answer",
			Destination: &topicConfigOptions.AnswerTopicPrefix,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_candidate_send_prefix",
			Usage:       "MQTT topic prefix for WebRTC candidate sending, and the sending topic of cloud is /edge/livestream/signal/candidate/recv",
			Value:       "/edge/livestream/signal/candidate/send",
			DefaultText: "/edge/livestream/signal/candidate/send",
			Destination: &topicConfigOptions.CandidateSendTopicPrefix,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mqtt_client.topic_candidate_recv_prefix",
			Usage:       "MQTT topic prefix for WebRTC candidate receiving, and the receiving topic of cloud is /edge/livestream/signal/candidate/send", //nolint:lll
			Value:       "/edge/livestream/signal/candidate/recv",
			DefaultText: "/edge/livestream/signal/candidate/recv",
			Destination: &topicConfigOptions.CandidateRecvTopicPrefix,
		}),
		altsrc.NewUintFlag(&cli.UintFlag{
			Name:        "mqtt_client.qos",
			Usage:       "MQTT client qos for WebRTC SDP signaling",
			Value:       0,
			DefaultText: "0",
			Destination: &topicConfigOptions.Qos,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "mqtt_client.retained",
			Usage:       "MQTT client setting retainsion for WebRTC SDP signaling",
			Value:       false,
			DefaultText: "false",
			Destination: &topicConfigOptions.Retained,
		}),
	}
}

func webRTCFlags(webRTCConfigOptions *livestream.WebRTCConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "webrtc.ice_server",
			Usage:       "ICE server address for webRTC",
			Value:       "stun:stun.l.google.com:19302",
			DefaultText: "stun:stun.l.google.com:19302",
			Destination: &webRTCConfigOptions.ICEServer,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "webrtc.ice_server_username",
			Usage:       "ICE server username for webRTC",
			Value:       "",
			DefaultText: "",
			Destination: &webRTCConfigOptions.Username,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "webrtc.ice_server_credential",
			Usage:       "ICE server credential for webRTC",
			Value:       "",
			DefaultText: "",
			Destination: &webRTCConfigOptions.Credential,
		}),
	}
}

func rtpFlags(rtpConfigOptions *livestream.RTPSourceConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "rtp.host",
			Usage:       "Host of RTP server",
			Value:       "0.0.0.0",
			DefaultText: "0.0.0.0",
			Destination: &rtpConfigOptions.RTPHost,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "rtp.port",
			Usage:       "Port of RTP server",
			Value:       5004,
			DefaultText: "5004",
			Destination: &rtpConfigOptions.RTPPort,
		}),
	}
}

func rtspFlags(rtspConfigOptions *livestream.RTSPSourceConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "rtsp.addr",
			Usage:       "Address of RTSP server",
			Value:       "",
			Destination: &rtspConfigOptions.RTSPAddr,
		}),
	}
}
