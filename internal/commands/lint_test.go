package commands

import (
	"bytes"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint"

	"github.com/spf13/cobra"
)

func TestNewLintCommand_Metadata(t *testing.T) {
	cmd := NewLintCommand()

	if cmd.Use != "lint" {
		t.Errorf("Use = %q, want %q", cmd.Use, "lint")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestBindLintFlags_BindsGOOSAndNewFromRev(t *testing.T) {
	var opts lint.LintOpts
	//nolint:exhaustruct // cobra uses partial initialization
	cmd := &cobra.Command{Use: "lint"}
	bindLintFlags(cmd, &opts)
	args := []string{"--goos=windows,darwin", "--new-from-rev=main"}
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	if opts.GOOS != "windows,darwin" {
		t.Errorf("GOOS = %q, want %q", opts.GOOS, "windows,darwin")
	}
	if opts.NewFromRev != "main" {
		t.Errorf("NewFromRev = %q, want %q", opts.NewFromRev, "main")
	}
}

func TestNewLintCommand_Help(t *testing.T) {
	cmd := NewLintCommand()
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

func TestSharedLinterFlags_ConsistentInCheckAndLint(t *testing.T) {
	check, lint := NewCheckCommand().Flags(), NewLintCommand().Flags()
	ints := linterIntDefaults()
	shared := make([]string, 0, len(ints)+2)
	shared = append(shared, "exclude", "rule")
	for name := range ints {
		shared = append(shared, name)
	}
	for _, name := range shared {
		checkFlag, lintFlag := check.Lookup(name), lint.Lookup(name)
		if checkFlag == nil || lintFlag == nil {
			t.Errorf(
				"flag %q missing in check=%v, lint=%v",
				name, checkFlag == nil, lintFlag == nil,
			)
			continue
		}
		if checkFlag.DefValue != lintFlag.DefValue {
			t.Errorf(
				"flag %q default mismatch: check=%q, lint=%q",
				name, checkFlag.DefValue, lintFlag.DefValue,
			)
		}
	}
}

func TestNewLintCommand_Flags(t *testing.T) {
	want := map[string]string{
		"paths":            "string",
		"fix":              "bool",
		"deadcode":         "bool",
		"deadcode-exclude": "string",
		"no-gofumpt":       "bool",
		"no-golangci":      "bool",
		"new-from-rev":     "string",
		"goos":             "string",
		"exclude":          "string",
		"rule":             "string",
	}
	for name := range linterIntDefaults() {
		want[name] = "int"
	}
	assertFlagTypes(t, NewLintCommand().Flags(), want)
}

func TestNewLintCommand_FlagDefaults(t *testing.T) {
	flags := NewLintCommand().Flags()
	assertStringDefaults(t, flags, map[string]string{
		"paths":            "",
		"deadcode-exclude": "",
		"new-from-rev":     "",
		"goos":             "",
		"exclude":          "",
		"rule":             "",
	})
	assertBoolDefaults(t, flags, map[string]bool{
		"fix":         true,
		"deadcode":    true,
		"no-gofumpt":  false,
		"no-golangci": false,
	})
	assertIntDefaults(t, flags, linterIntDefaults())
}
