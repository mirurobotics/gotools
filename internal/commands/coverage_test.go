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
	cmd := NewCoverageCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"src-prefix", "string"},
		{"test-prefix", "string"},
		{"no-html", "bool"},
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

func TestNewCoverageCommand_FlagDefaults(t *testing.T) {
	cmd := NewCoverageCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{"src-prefix": "pkg", "test-prefix": "tests"}
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

	got, err := fl.GetBool("no-html")
	if err != nil {
		t.Errorf("flag no-html: %v", err)
	} else if got != false {
		t.Errorf("flag no-html default = %v, want false", got)
	}
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
