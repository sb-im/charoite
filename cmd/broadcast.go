//go:build broadcast

package main

import "github.com/SB-IM/skywalker/cmd/broadcast"

func init() {
	commands = append(commands, broadcast.Command())
}
