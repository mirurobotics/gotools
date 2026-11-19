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
	cmd := NewCovratchetCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"packages", "string"},
		{"src-prefix", "string"},
		{"test-dir", "string"},
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

func TestNewCovratchetCommand_FlagDefaults(t *testing.T) {
	cmd := NewCovratchetCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{
		"packages":   "./...",
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

	gotP, errP := fl.GetInt("parallelism")
	if errP != nil {
		t.Errorf("flag parallelism: %v", errP)
	} else if gotP != 0 {
		t.Errorf("flag parallelism default = %d, want 0", gotP)
	}
}

func TestNewCovratchetCommand_NoDefaultThreshold(t *testing.T) {
	cmd := NewCovratchetCommand()
	f := cmd.Flags().Lookup("default-threshold")
	if f != nil {
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
