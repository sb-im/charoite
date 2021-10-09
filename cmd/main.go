package main

import (
	stdlog "log"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // pprof
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
			&cli.BoolFlag{
				Name:        "profile",
				Value:       false,
				Usage:       "enable profiling",
				DefaultText: "false",
				EnvVars:     []string{"PROFILE"},
			},
		},
		Before: func(c *cli.Context) error {
			go func() {
				stdlog.Println("Starting pprof server at :6060")
				stdlog.Fatal(http.ListenAndServe(":6060", http.DefaultServeMux))
			}()
			return nil
		},
		Commands: commands,
	}

	return app.Run(args)
}
