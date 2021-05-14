package turn

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/SB-IM/logging"
	"github.com/SB-IM/skywalker/internal/turn"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const configFlagName = "config"

func Command() *cli.Command {
	var (
		logger            zerolog.Logger
		turnConfigOptions turn.ConfigOptions
	)

	flags := func() (flags []cli.Flag) {
		for _, v := range [][]cli.Flag{
			loadConfigFlag(),
			turnConfigFlags(&turnConfigOptions),
		} {
			flags = append(flags, v...)
		}
		return
	}()

	return &cli.Command{
		Name:  "turn",
		Usage: "Start TURN server",
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
			logger = log.With().Str("service", "skywalker").Str("command", "turn").Logger()
			return nil
		},
		Action: func(c *cli.Context) error {
			s, err := turn.Serve(&logger, &turnConfigOptions)
			if err != nil {
				return err
			}

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs

			return s.Close()
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

func turnConfigFlags(options *turn.ConfigOptions) []cli.Flag {
	return []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "turn.public_ip",
			Usage:       "IP Address that TURN can be contacted by",
			Value:       "127.0.0.1",
			DefaultText: "127.0.0.1",
			Destination: &options.PublicIP,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "turn.port",
			Usage:       "Listening port",
			Value:       3478,
			DefaultText: "3478",
			Destination: &options.Port,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "turn.username",
			Usage:       "Username",
			Value:       "user",
			DefaultText: "user",
			Destination: &options.Username,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "turn.password",
			Usage:       "Password",
			Value:       "password",
			DefaultText: "password",
			Destination: &options.Password,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "turn.realm",
			Usage:       "Realm",
			Value:       "example.com",
			DefaultText: "example.com",
			Destination: &options.Realm,
		}),
		altsrc.NewUintFlag(&cli.UintFlag{
			Name:        "turn.relay_min_port",
			Usage:       "Minimum relay port",
			Value:       50000,
			DefaultText: "50000",
			Destination: &options.RelayMinPort,
		}),
		altsrc.NewUintFlag(&cli.UintFlag{
			Name:        "turn.relay_max_port",
			Usage:       "Maximum relay port",
			Value:       55000,
			DefaultText: "55000",
			Destination: &options.RelayMaxPort,
		}),
	}
}
