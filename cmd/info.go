//go:build info

package main

import "github.com/SB-IM/charoite/cmd/internal/info"

func init() {
	commands = append(commands, info.Command())
}
