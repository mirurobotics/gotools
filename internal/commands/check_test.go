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
	cmd := NewCheckCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"path", "string"},
		{"fix", "bool"},
		{"max-line-width", "int"},
		{"tab-width", "int"},
		{"max-func-len", "int"},
		{"max-file-len", "int"},
		{"max-nest-depth", "int"},
		{"max-param-count", "int"},
		{"max-doc-lines", "int"},
		{"max-doc-file-lines", "int"},
		{"exclude", "string"},
		{"rule", "string"},
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
