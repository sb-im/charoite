//go:build turn

package main

import "github.com/SB-IM/charoite/cmd/turn"

func init() {
	commands = append(commands, turn.Command())
}
