//go:build build

package main

import "github.com/SB-IM/charoite/cmd/build"

func init() {
	commands = append(commands, build.Command())
}
