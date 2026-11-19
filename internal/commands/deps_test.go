package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewDepsCommand_Metadata(t *testing.T) {
	cmd := NewDepsCommand()

	if cmd.Use != "deps" {
		t.Errorf("Use = %q, want %q", cmd.Use, "deps")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestNewDepsCommand_Subcommands(t *testing.T) {
	cmd := NewDepsCommand()

	want := map[string]bool{"update": false}

	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered on deps", name)
		}
	}
}

func TestNewDepsCommand_UpdateHasRunE(t *testing.T) {
	cmd := NewDepsCommand()
	var update *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "update" {
			update = sub
			break
		}
	}
	if update == nil {
		t.Fatal("update subcommand not found")
	}
	if update.RunE == nil {
		t.Error("update RunE is nil")
	}
}

func TestNewDepsCommand_Help(t *testing.T) {
	cmd := NewDepsCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}
