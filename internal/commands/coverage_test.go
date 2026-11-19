package commands

import (
	"bytes"
	"testing"
)

func TestNewCoverageCommand_Metadata(t *testing.T) {
	cmd := NewCoverageCommand()

	if cmd.Use != "coverage [sub-package]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "coverage [sub-package]")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCoverageCommand_Flags(t *testing.T) {
	assertFlagTypes(t, NewCoverageCommand().Flags(), map[string]string{
		"src-prefix": "string", "test-prefix": "string", "no-html": "bool",
	})
}

func TestNewCoverageCommand_FlagDefaults(t *testing.T) {
	fl := NewCoverageCommand().Flags()
	assertStringDefaults(t, fl,
		map[string]string{"src-prefix": "pkg", "test-prefix": "tests"})
	assertBoolDefaults(t, fl, map[string]bool{"no-html": false})
}

func TestNewCoverageCommand_AcceptsMaxOneArg(t *testing.T) {
	cmd := NewCoverageCommand()
	if cmd.Args == nil {
		t.Error("Args validator is nil; expected MaximumNArgs(1)")
	}
}

func TestNewCoverageCommand_Help(t *testing.T) {
	cmd := NewCoverageCommand()
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
