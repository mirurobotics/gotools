package lint

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestFilterDeadcodeOutput(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		exclude string
		want    int
	}{
		{"empty", "", "", 0},
		{"filters mod path", "/go/pkg/mod/foo.go:1: unused", "", 0},
		{"keeps local", "internal/foo.go:1: unused", "", 1},
		{
			"exclude pattern",
			"internal/foo.go:1: Unused\n" +
				"internal/bar.go:1: Other", "Unused", 1,
		},
		{"mixed", "/go/pkg/mod/x.go:1: a\ninternal/y.go:1: b\n", "", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterDeadcodeOutput(tt.raw, tt.exclude)
			if len(got) != tt.want {
				t.Errorf("len = %d, want %d: %v", len(got), tt.want, got)
			}
		})
	}
}

func TestRunExternal_Success(t *testing.T) {
	var out, errBuf bytes.Buffer
	err := RunExternal(&out, &errBuf, "echo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "hello\n" {
		t.Errorf("stdout = %q, want %q", got, "hello\n")
	}
}

func TestRunExternal_Failure(t *testing.T) {
	err := RunExternal(io.Discard, io.Discard, "false")
	if err == nil {
		t.Error("expected error from false command")
	}
}

func TestBuildLinterConfig_InvalidExclude(t *testing.T) {
	//nolint:exhaustruct // only testing exclude
	_, err := BuildLinterConfig(LinterFlags{Exclude: "nonexistent-rule"})
	if err == nil {
		t.Error("expected error for invalid exclusion")
	}
}

func TestBuildLinterConfig_UnknownSingleRule(t *testing.T) {
	//nolint:exhaustruct // only testing rule
	_, err := BuildLinterConfig(LinterFlags{Rule: "nonexistent"})
	if err == nil {
		t.Error("expected error for unknown rule")
	}
}

func TestBuildLinterConfig_ValidSingleRule(t *testing.T) {
	//nolint:exhaustruct // only testing rule
	cfg, err := BuildLinterConfig(LinterFlags{Rule: "errfmt"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Exclude["errfmt"] {
		t.Error("errfmt should not be excluded")
	}
}

func TestRunLint_AllSkipped(t *testing.T) {
	var out bytes.Buffer
	//nolint:exhaustruct // testing skip-all path
	err := RunLint(LintOpts{
		NoGofumpt:  true,
		NoGolangci: true,
		Out:        &out,
		Err:        io.Discard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Lint complete") {
		t.Errorf("expected 'Lint complete' in output, got %q", out.String())
	}
}

func TestRunLint_NilWriters(t *testing.T) {
	// Ensure nil Out/Err don't panic (they default to os.Stdout/os.Stderr).
	//nolint:exhaustruct // testing nil writer defaults
	err := RunLint(LintOpts{NoGofumpt: true, NoGolangci: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLint_EmptyPaths(t *testing.T) {
	var out bytes.Buffer
	// Empty Paths should skip the custom linter and succeed.
	//nolint:exhaustruct // testing empty Paths path
	err := RunLint(LintOpts{
		Paths:      "",
		NoGofumpt:  true,
		NoGolangci: true,
		Out:        &out,
		Err:        io.Discard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "custom linter") {
		t.Error("custom linter should not have run with empty Paths")
	}
	if !strings.Contains(out.String(), "Lint complete") {
		t.Errorf("expected 'Lint complete' in output, got %q", out.String())
	}
}
