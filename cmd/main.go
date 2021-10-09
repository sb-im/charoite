package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/SB-IM/charoite/cmd/internal/info"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var commands = make([]*cli.Command, 0, 2)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	if err := run(os.Args); err != nil {
		log.Err(err).Send()
	}
}

func run(args []string) error {
	commands = append(commands, info.Command())

	app := &cli.App{
		Name:  "Charoite",
		Usage: "Charoite WebRTC SFU services",
		Flags: []cli.Flag{ // Global flags.
			&cli.BoolFlag{
				Name:        "debug",
				Value:       false,
				Usage:       "enable debug mod",
				DefaultText: "false",
				EnvVars:     []string{"DEBUG"},
			},
		},
		Commands: commands,
	}

	return app.Run(args)
}
