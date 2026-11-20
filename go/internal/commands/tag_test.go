package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewTagCommand_Metadata(t *testing.T) {
	cmd := NewTagCommand()

	if cmd.Use != "tag" {
		t.Errorf("Use = %q, want %q", cmd.Use, "tag")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestNewTagCommand_Subcommands(t *testing.T) {
	cmd := NewTagCommand()

	want := map[string]bool{"previous": false, "latest": false, "current": false}

	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered on tag", name)
		}
	}
}

func TestNewTagCommand_PreviousAcceptsMaxOneArg(t *testing.T) {
	cmd := NewTagCommand()
	var prev *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "previous" {
			prev = sub
			break
		}
	}
	if prev == nil {
		t.Fatal("previous subcommand not found")
	}
	if prev.Args == nil {
		t.Error("previous Args validator is nil; expected MaximumNArgs(1)")
	}
}

func TestNewTagCommand_LatestAcceptsMaxOneArg(t *testing.T) {
	cmd := NewTagCommand()
	var latest *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "latest" {
			latest = sub
			break
		}
	}
	if latest == nil {
		t.Fatal("latest subcommand not found")
	}
	if latest.Args == nil {
		t.Error("latest Args validator is nil; expected MaximumNArgs(1)")
	}
}

func TestNewTagCommand_Help(t *testing.T) {
	cmd := NewTagCommand()
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
