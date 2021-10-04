//go:build build

package main

import "github.com/SB-IM/skywalker/cmd/build"

func init() {
	commands = append(commands, build.Command())
}
