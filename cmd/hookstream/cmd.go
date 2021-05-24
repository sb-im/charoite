package hookstream

import (
	"context"
	"time"

	"github.com/SB-IM/logging"
	mqttclient "github.com/SB-IM/mqtt-client"
	"github.com/SB-IM/sphinx/internal/hookstream"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const configFlagName = "config"

// Command returns a livestream command.
func Command() *cli.Command {
	ctx := context.Background()

	var (
		logger zerolog.Logger

		mc mqtt.Client

		uuid                    string
		mqttConfigOptions       mqttclient.ConfigOptions
		mqttClientConfigOptions hookstream.MQTTClientConfigOptions
		hookStreamConfigOptions hookstream.HookCommandLine
	)

	flags := func() (flags []cli.Flag) {
		for _, v := range [][]cli.Flag{
			loadConfigFlag(),
			uuidFlag(&uuid),
			mqttFlags(&mqttConfigOptions),
			mqttClientFlags(&mqttClientConfigOptions),
			hookCommandLineFlags(&hookStreamConfigOptions),
		} {
			flags = append(flags, v...)
		}
		return
	}()

	return &cli.Command{
		Name:  "hookstream",
		Usage: "Hookstream hooks a track source stream by executing hooking command line",
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
			logger = log.With().Str("service", "sphinx").Str("command", "hookstream").Logger()
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
			errCh := hookstream.Exec(ctx, hookstream.ConfigOptions{
				UUID:                    uuid,
				HookCommandLine:         hookStreamConfigOptions,
				MQTTClientConfigOptions: mqttClientConfigOptions,
			})
			return <-errCh
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

func uuidFlag(uuid *string) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "machine_id.uuid",
			Usage:       "UUID v4 for this Linux machine",
			Value:       "",
			DefaultText: "",
			Destination: uuid,
		}),
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

func mqttClientFlags(options *hookstream.MQTTClientConfigOptions) []cli.Flag {
	return []cli.Flag{
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

func hookCommandLineFlags(options *hookstream.HookCommandLine) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "hook_stream.service",
			Usage:       "Systemd service to be restarted of hooking stream",
			Value:       "sbcameradronemulti.service",
			DefaultText: "sbcameradronemulti.service",
			Destination: &options.Service,
		}),
		altsrc.NewDurationFlag(&cli.DurationFlag{
			Name:        "hook_stream.wait_timeout",
			Usage:       "Waiting timeout to hook stream in seconds",
			Value:       1 * time.Second,
			DefaultText: "1",
			Destination: &options.WaitTimeout,
		}),
	}
}
