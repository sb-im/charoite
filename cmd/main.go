package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/SB-IM/sphinx/cmd/livestream"
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
		Name:  "sphinx",
		Usage: "sphinx runs in edge, currently includes livestream sub-service",
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
			livestream.Command(),
		},
	}

	return app.Run(args)
}
