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
	cmd := NewCovgateCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"packages", "string"},
		{"exclude", "string"},
		{"src-prefix", "string"},
		{"test-dir", "string"},
		{"default-threshold", "float64"},
		{"parallelism", "int"},
	}

	for _, tc := range tests {
		f := fl.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag %q not registered", tc.name)
			continue
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag %q type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

func TestNewCovgateCommand_FlagDefaults(t *testing.T) {
	cmd := NewCovgateCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{
		"packages":   "./...",
		"exclude":    "",
		"src-prefix": "pkg",
		"test-dir":   "",
	}
	for name, want := range stringDefaults {
		got, err := fl.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %q, want %q", name, got, want)
		}
	}

	got, err := fl.GetFloat64("default-threshold")
	if err != nil {
		t.Errorf("flag default-threshold: %v", err)
	} else if got != 80.0 {
		t.Errorf("flag default-threshold default = %f, want %f", got, 80.0)
	}

	gotP, errP := fl.GetInt("parallelism")
	if errP != nil {
		t.Errorf("flag parallelism: %v", errP)
	} else if gotP != 0 {
		t.Errorf("flag parallelism default = %d, want 0", gotP)
	}
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
