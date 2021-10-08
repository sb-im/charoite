//go:build livestream

package main

import "github.com/SB-IM/charoite/cmd/internal/livestream"

func init() {
	commands = append(commands, livestream.Command())
}
