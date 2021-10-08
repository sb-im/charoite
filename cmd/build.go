//go:build turn

package main

import "github.com/SB-IM/charoite/cmd/internal/turn"

func init() {
	commands = append(commands, turn.Command())
}
