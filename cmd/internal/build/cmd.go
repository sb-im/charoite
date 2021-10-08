package build

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

var (
	Branch    string
	Version   string
	Revision  string
	BuildUser string
	BuildDate string
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "info displays build information of this binary",
		Action: func(c *cli.Context) error {
			fmt.Printf(`Branch:		%s
Version:	%s
Revision:	%s
BuildUser:	%s
BuildDate:	%s`, Branch, Version, Revision, BuildUser, BuildDate)
			return nil
		},
	}
}
