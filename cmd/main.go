//go:build turn || build || broadcast

package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var commands = make([]*cli.Command, 0, 3)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	if err := run(os.Args); err != nil {
		log.Err(err).Send()
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
		Commands: commands,
	}

	return app.Run(args)
}
