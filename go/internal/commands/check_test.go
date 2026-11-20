package commands

import (
	"bytes"
	"testing"
)

func TestNewCheckCommand_Metadata(t *testing.T) {
	cmd := NewCheckCommand()

	if cmd.Use != "check" {
		t.Errorf("Use = %q, want %q", cmd.Use, "check")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCheckCommand_Flags(t *testing.T) {
	want := map[string]string{"path": "string", "fix": "bool"}
	for name := range linterIntDefaults() {
		want[name] = "int"
	}
	want["exclude"], want["rule"] = "string", "string"
	assertFlagTypes(t, NewCheckCommand().Flags(), want)
}

func TestNewCheckCommand_Help(t *testing.T) {
	cmd := NewCheckCommand()
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

func TestNewCheckCommand_FlagDefaults(t *testing.T) {
	fl := NewCheckCommand().Flags()
	assertStringDefaults(t, fl,
		map[string]string{"path": ".", "exclude": "", "rule": ""})
	assertBoolDefaults(t, fl, map[string]bool{"fix": false})
	assertIntDefaults(t, fl, linterIntDefaults())
}
