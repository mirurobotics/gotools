package commands

import (
	"bytes"
	"testing"
)

func TestNewRootCommand_Metadata(t *testing.T) {
	cmd := NewRootCommand()

	if cmd.Use != "miru" {
		t.Errorf("Use = %q, want %q", cmd.Use, "miru")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.Long == "" {
		t.Error("Long description is empty")
	}
	if !cmd.SilenceErrors {
		t.Error("SilenceErrors should be true")
	}
	if !cmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
}

func TestNewRootCommand_Subcommands(t *testing.T) {
	cmd := NewRootCommand()

	want := map[string]bool{
		"lint":       false,
		"check":      false,
		"covgate":    false,
		"covratchet": false,
		"coverage":   false,
		"test":       false,
		"deps":       false,
		"tag":        false,
	}

	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered on root", name)
		}
	}
}

func TestNewRootCommand_Help(t *testing.T) {
	cmd := NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

func TestNewTestCommand_Metadata(t *testing.T) {
	cmd := NewTestCommand()

	if cmd.Use != "test [-- extra args]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "test [-- extra args]")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewTestCommand_DisablesFlagParsing(t *testing.T) {
	cmd := NewTestCommand()
	if !cmd.DisableFlagParsing {
		t.Error("DisableFlagParsing should be true")
	}
}

func TestNewTestCommand_Help(t *testing.T) {
	cmd := NewTestCommand()
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
