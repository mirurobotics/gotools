package commands

import (
	"bytes"
	"testing"
)

func TestNewCovratchetCommand_Metadata(t *testing.T) {
	cmd := NewCovratchetCommand()

	if cmd.Use != "covratchet" {
		t.Errorf("Use = %q, want %q", cmd.Use, "covratchet")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCovratchetCommand_Flags(t *testing.T) {
	assertFlagTypes(t, NewCovratchetCommand().Flags(), map[string]string{
		"packages":    "string",
		"src-prefix":  "string",
		"test-dir":    "string",
		"parallelism": "int",
	})
}

func TestNewCovratchetCommand_FlagDefaults(t *testing.T) {
	flags := NewCovratchetCommand().Flags()
	assertStringDefaults(t, flags, map[string]string{
		"packages": "./...", "src-prefix": "pkg", "test-dir": "",
	})
	assertIntDefaults(t, flags, map[string]int{"parallelism": 0})
}

func TestNewCovratchetCommand_NoDefaultThreshold(t *testing.T) {
	cmd := NewCovratchetCommand()
	flag := cmd.Flags().Lookup("default-threshold")
	if flag != nil {
		t.Error("covratchet should not have a default-threshold flag")
	}
}

func TestNewCovratchetCommand_Help(t *testing.T) {
	cmd := NewCovratchetCommand()
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
