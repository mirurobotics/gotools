package cmdutil

import (
	"testing"
)

func TestGoCommand_SetsGOWORK(t *testing.T) {
	cmd := GoCommand("version")
	found := false
	for _, e := range cmd.Env {
		if e == "GOWORK=off" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GoCommand did not set GOWORK=off")
	}
}

func TestGoCommand_Runs(t *testing.T) {
	cmd := GoCommand("version")
	if err := cmd.Run(); err != nil {
		t.Fatalf("GoCommand('version') failed: %v", err)
	}
}
