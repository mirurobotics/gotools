package commands

import (
	"bytes"
	"testing"
)

func TestNewCovgateCommand_Metadata(t *testing.T) {
	cmd := NewCovgateCommand()

	if cmd.Use != "covgate" {
		t.Errorf("Use = %q, want %q", cmd.Use, "covgate")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCovgateCommand_Flags(t *testing.T) {
	assertFlagTypes(t, NewCovgateCommand().Flags(), map[string]string{
		"packages":          "string",
		"exclude":           "string",
		"src-prefix":        "string",
		"test-dir":          "string",
		"default-threshold": "float64",
		"parallelism":       "int",
	})
}

func TestNewCovgateCommand_FlagDefaults(t *testing.T) {
	fl := NewCovgateCommand().Flags()
	assertStringDefaults(t, fl, map[string]string{
		"packages": "./...", "exclude": "", "src-prefix": "pkg", "test-dir": "",
	})
	assertFloatDefaults(t, fl, map[string]float64{"default-threshold": 80.0})
	assertIntDefaults(t, fl, map[string]int{"parallelism": 0})
}

func TestNewCovgateCommand_Help(t *testing.T) {
	cmd := NewCovgateCommand()
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
