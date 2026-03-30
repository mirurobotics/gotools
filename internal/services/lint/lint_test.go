package lint

import (
	"bytes"
	"io"
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
	_, err := BuildLinterConfig("nonexistent-rule", "", 88, 4, 50, 4, 5)
	if err == nil {
		t.Error("expected error for invalid exclusion")
	}
}

func TestBuildLinterConfig_UnknownSingleRule(t *testing.T) {
	_, err := BuildLinterConfig("", "nonexistent", 88, 4, 50, 4, 5)
	if err == nil {
		t.Error("expected error for unknown rule")
	}
}

func TestBuildLinterConfig_ValidSingleRule(t *testing.T) {
	cfg, err := BuildLinterConfig("", "errfmt", 88, 4, 50, 4, 5)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Exclude["errfmt"] {
		t.Error("errfmt should not be excluded")
	}
}
