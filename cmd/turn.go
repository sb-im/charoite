//go:build build

package main

import "github.com/SB-IM/charoite/cmd/internal/build"

func init() {
	commands = append(commands, build.Command())
}
