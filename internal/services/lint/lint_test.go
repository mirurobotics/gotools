package lint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

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
	s := out.String()
	if !strings.Contains(s, "Lint complete") {
		t.Errorf("expected 'Lint complete' in output, got %q", s)
	}
	if !strings.Contains(s, "Timings") {
		t.Errorf("expected timing summary in output, got %q", s)
	}
	if !strings.Contains(s, "total") {
		t.Errorf("expected total timing in output, got %q", s)
	}
}

func TestRunLint_NilWriters(t *testing.T) {
	//nolint:exhaustruct // testing nil writer defaults
	err := RunLint(LintOpts{NoGofumpt: true, NoGolangci: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLint_DeadcodeError_SurfacedToErrW(t *testing.T) {
	var out, errBuf bytes.Buffer
	//nolint:exhaustruct // only testing deadcode error-surfacing path
	err := RunLint(LintOpts{
		Deadcode:   true,
		NoGofumpt:  true,
		NoGolangci: true,
		Out:        &out,
		Err:        &errBuf,
	})
	if err != nil {
		if !strings.Contains(err.Error(), "deadcode") {
			t.Errorf("expected error to contain 'deadcode', got: %v", err)
		}
	}
}

func TestRunLint_EmptyPaths(t *testing.T) {
	var out bytes.Buffer
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
	s := out.String()
	if strings.Contains(s, "custom linter") {
		t.Error("custom linter should not have run with empty Paths")
	}
	if !strings.Contains(s, "Lint complete") {
		t.Errorf("expected 'Lint complete' in output, got %q", s)
	}
}

func TestFmtDuration_Seconds(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0.0s"},
		{500 * time.Millisecond, "0.5s"},
		{3200 * time.Millisecond, "3.2s"},
		{59*time.Second + 900*time.Millisecond, "59.9s"},
	}
	for _, tt := range tests {
		got := fmtDuration(tt.d)
		if got != tt.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFmtDuration_Minutes(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{60 * time.Second, "1m00s"},
		{7*time.Minute + 45*time.Second, "7m45s"},
	}
	for _, tt := range tests {
		got := fmtDuration(tt.d)
		if got != tt.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestRunLint_ParallelGolangciAndDeadcode(t *testing.T) {
	var out bytes.Buffer
	//nolint:exhaustruct // testing parallel path
	err := RunLint(LintOpts{
		Deadcode:   true,
		NoGofumpt:  true,
		NoGolangci: false,
		Out:        &out,
		Err:        io.Discard,
	})
	s := out.String()
	if !strings.Contains(s, "golangci-lint") {
		t.Error("expected golangci-lint in timing output")
	}
	if !strings.Contains(s, "deadcode") {
		t.Error("expected deadcode in timing output")
	}
	if !strings.Contains(s, "Timings") {
		t.Error("expected Timings header")
	}
	_ = err
}

func TestRunLintSteps_RunsGOOSTargetsLast(t *testing.T) {
	fakeGoTool(t, fakeGolangci(t, 1))
	//nolint:exhaustruct // only testing GOOS target wiring
	opts := LintOpts{NoGofumpt: true, GOOS: "windows", Out: io.Discard, Err: io.Discard}
	failures, timings, err := runLintSteps(opts)
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	target := "golangci-lint (windows)"
	if !slices.Equal(failures, []string{target}) {
		t.Errorf("failures = %v, want [%q]", failures, target)
	}
	want := []string{"golangci-lint", target}
	if got := timingNames(timings); !slices.Equal(got, want) {
		t.Errorf("timings = %v, want %v", got, want)
	}
}

func TestRunLint_NoGolangciSkipsGOOSTargets(t *testing.T) {
	var out bytes.Buffer
	//nolint:exhaustruct // only testing the skip path
	err := RunLint(LintOpts{
		NoGofumpt:  true,
		NoGolangci: true,
		GOOS:       "windows",
		Out:        &out,
		Err:        io.Discard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "windows") {
		t.Errorf("GOOS targets should not run with NoGolangci, got %q", out.String())
	}
}

func TestPrintTimings_WidensForLongStepNames(t *testing.T) {
	var buf bytes.Buffer
	name := "golangci-lint (windows)"
	printTimings(&buf, []stepTiming{{name, time.Second}}, 2*time.Second)
	s := buf.String()
	pad := strings.Repeat(" ", len(name)-len("total"))
	for _, want := range []string{name + "    1.0s\n", "total" + pad + "    2.0s\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in output, got %q", want, s)
		}
	}
}

func TestPrintTimings(t *testing.T) {
	var buf bytes.Buffer
	timings := []stepTiming{
		{"gofumpt", 400 * time.Millisecond},
		{"golangci-lint", 7*time.Minute + 45*time.Second},
	}
	printTimings(&buf, timings, 8*time.Minute+10*time.Second)
	s := buf.String()
	if !strings.Contains(s, "Timings") {
		t.Error("missing Timings header")
	}
	if !strings.Contains(s, "gofumpt") {
		t.Error("missing gofumpt timing")
	}
	if !strings.Contains(s, "golangci-lint") {
		t.Error("missing golangci-lint timing")
	}
	if !strings.Contains(s, "total") {
		t.Error("missing total timing")
	}
	for _, want := range []string{
		"gofumpt             0.4s\n",
		"golangci-lint      7m45s\n",
		"total              8m10s\n",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %q in output, got %q", want, s)
		}
	}
}

func fakeGolangci(t *testing.T, exitCode int) string {
	t.Helper()
	body := "echo \"GOOS=$GOOS GOWORK=$GOWORK args=$*\"\n"
	return writeScript(t, "golangci-lint", body+fmt.Sprintf("exit %d\n", exitCode))
}

func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	//nolint:gosec // G306: test executable
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeGoTool(t *testing.T, bin string) {
	t.Helper()
	body := "if [ \"$*\" = \"tool -n golangci-lint\" ]; then\n" +
		"\techo '" + bin + "'\n" +
		"else\n" +
		"\techo \"go $*\"\n" +
		"fi\n"
	goBin := writeScript(t, "go", body)
	t.Setenv("PATH", filepath.Dir(goBin))
}

func timingNames(timings []stepTiming) []string {
	names := make([]string, 0, len(timings))
	for _, timing := range timings {
		names = append(names, timing.name)
	}
	return names
}
