package livestream

import (
	"fmt"
	"testing"
)

func TestMachineID(t *testing.T) {
	id, err := machineID()
	if err != nil || id == nil {
		t.Fatal(err)
	}
	fmt.Println(string(id))
}
