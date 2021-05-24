package pubkey

import (
	"bytes"
	"os"
	"os/user"
	"path/filepath"
)

const ed25519PubKeyFile = ".ssh/id_ed25519.pub"

func Ed25519PubKey() string {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	path := filepath.Join(currentUser.HomeDir, ed25519PubKeyFile)
	id, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	subBytes := bytes.Split(id, []byte(" "))
	if len(subBytes) != 3 {
		panic("invalid id_ed25519 pub key")
	}
	return string(subBytes[1])
}
