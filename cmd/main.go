package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/SB-IM/skywalker/cmd/broadcast"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	if err := run(os.Args); err != nil {
		log.Fatal().Err(err)
	}
}

func run(args []string) error {
	app := &cli.App{
		Name:  "skywalker",
		Usage: "skywalker runs in cloud, currently includes broadcast sub-service",
		Flags: []cli.Flag{ // Global flags.
			&cli.BoolFlag{
				Name:        "debug",
				Value:       false,
				Usage:       "enable debug mod",
				DefaultText: "false",
				EnvVars:     []string{"DEBUG"},
			},
		},
		Commands: []*cli.Command{
			broadcast.Command(),
		},
	}

	return app.Run(args)
}
