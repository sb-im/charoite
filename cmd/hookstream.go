//go:build hookstream

package main

import "github.com/SB-IM/charoite/cmd/internal/hookstream"

func init() {
	commands = append(commands, hookstream.Command())
}
