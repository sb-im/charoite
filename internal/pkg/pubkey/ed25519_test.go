package pubkey

import (
	"fmt"
	"testing"
)

func TestEd25519PubKey(t *testing.T) {
	key := Ed25519PubKey()
	if key == "" {
		t.Fatal("empty key")
	}
	fmt.Println(key)
}
