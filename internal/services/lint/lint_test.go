package lint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	//nolint:gosec // G306: test executable
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func fakeGolangci(t *testing.T, exitCode int) string {
	t.Helper()
	body := "echo \"GOOS=$GOOS GOWORK=$GOWORK args=$*\"\n"
	return writeScript(t, "golangci-lint", body+fmt.Sprintf("exit %d\n", exitCode))
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

func TestRunDeadcode_ToolCrash_ErrorWrittenToErrW(t *testing.T) {
	var out, errBuf bytes.Buffer
	err := RunDeadcode(&out, &errBuf, "")
	if err != nil && !strings.Contains(err.Error(), "deadcode found issues") {
		if errBuf.Len() == 0 {
			t.Errorf("RunDeadcode tool crash: errW was empty; err=%v", err)
		}
	}
	_ = out
}

func TestRunDeadcode_NonexistentPackage_ErrorWrittenToErrW(t *testing.T) {
	var out, errBuf bytes.Buffer
	err := RunExternal(&out, &errBuf,
		"go", "tool", "deadcode", "--bad-flag-that-does-not-exist")
	if err == nil {
		t.Skip("expected deadcode to reject unknown flag, but it did not")
	}
	if errBuf.Len() == 0 {
		t.Error("expected errW to contain deadcode error output, but it was empty")
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

func TestRunGolangci_PassesNewFromRev(t *testing.T) {
	fakeGoTool(t, fakeGolangci(t, 0))
	var out bytes.Buffer
	if err := RunGolangci(&out, io.Discard, "main"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Running golangci-lint...\n" +
		"go tool golangci-lint run --new-from-rev=main\n"
	if got := out.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunGolangciGOOS_Success(t *testing.T) {
	fakeGoTool(t, fakeGolangci(t, 0))
	var out, errBuf bytes.Buffer
	if err := RunGolangciGOOS(&out, &errBuf, "", "windows"); err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, errBuf.String())
	}
	want := "Running golangci-lint for windows...\n" +
		"GOOS=windows GOWORK=off args=run\n"
	if got := out.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunGolangciGOOS_UnsupportedGOOS(t *testing.T) {
	var errBuf bytes.Buffer
	err := RunGolangciGOOS(io.Discard, &errBuf, "", "notanos")
	if err == nil {
		t.Fatal("expected error for unsupported GOOS")
	}
	if !strings.Contains(errBuf.String(), "golangci-lint for notanos failed") {
		t.Errorf("expected failure surfaced to errW, got %q", errBuf.String())
	}
}

func TestRunGolangciGOOS_ResolveFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var errBuf bytes.Buffer
	err := RunGolangciGOOS(io.Discard, &errBuf, "", "windows")
	if err == nil {
		t.Fatal("expected error when go is not on PATH")
	}
	want := "golangci-lint for windows failed: resolve go tool golangci-lint"
	if !strings.Contains(errBuf.String(), want) {
		t.Errorf("errW = %q, want it to contain %q", errBuf.String(), want)
	}
}

func TestRunGolangciBin_PassesNewFromRev(t *testing.T) {
	var out bytes.Buffer
	bin := fakeGolangci(t, 0)
	if err := runGolangciBin(&out, io.Discard, bin, "main", "windows"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "GOOS=windows GOWORK=off args=run --new-from-rev=main\n"
	if got := out.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunGolangciBin_OverridesInheritedGOOS(t *testing.T) {
	t.Setenv("GOOS", "plan9")
	t.Setenv("GOWORK", "auto")
	var out bytes.Buffer
	bin := fakeGolangci(t, 0)
	if err := runGolangciBin(&out, io.Discard, bin, "", "windows"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "GOOS=windows GOWORK=off args=run\n"
	if got := out.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunGolangciBin_Failure(t *testing.T) {
	var errBuf bytes.Buffer
	err := runGolangciBin(io.Discard, &errBuf, fakeGolangci(t, 1), "", "windows")
	if err == nil {
		t.Fatal("expected error from failing golangci-lint")
	}
	if !strings.Contains(errBuf.String(), "golangci-lint for windows failed") {
		t.Errorf("expected failure surfaced to errW, got %q", errBuf.String())
	}
}

func TestGoosTargets(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"blanks", " , ,", nil},
		{"trims", " notanos ", []string{"notanos"}},
		{"keeps order", "b,a", []string{"b", "a"}},
		{"drops duplicates", "a,b,a, b", []string{"a", "b"}},
		{"drops host", runtime.GOOS + ",a," + runtime.GOOS, []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := goosTargets(tt.raw); !slices.Equal(got, tt.want) {
				t.Errorf("goosTargets(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRunGolangciTargets(t *testing.T) {
	tests := []struct {
		name string
		goos string
		want []string
	}{
		{"empty", "", nil},
		{"blanks", " , ", nil},
		{"two targets in order", "windows, plan9", []string{"windows", "plan9"}},
		{"duplicates and host", "windows,windows," + runtime.GOOS, []string{"windows"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeGoTool(t, fakeGolangci(t, 1))
			//nolint:exhaustruct // only testing target iteration
			opts := LintOpts{GOOS: tt.goos, Out: io.Discard, Err: io.Discard}
			failures, timings := runGolangciTargets(opts)
			steps := make([]string, 0, len(tt.want))
			for _, goos := range tt.want {
				steps = append(steps, "golangci-lint ("+goos+")")
			}
			if !slices.Equal(failures, steps) {
				t.Errorf("failures = %v, want %v", failures, steps)
			}
			if got := timingNames(timings); !slices.Equal(got, steps) {
				t.Errorf("timings = %v, want %v", got, steps)
			}
		})
	}
}

func TestRunGolangciTargets_PassesNewFromRev(t *testing.T) {
	fakeGoTool(t, fakeGolangci(t, 0))
	var out bytes.Buffer
	//nolint:exhaustruct // only testing NewFromRev wiring
	opts := LintOpts{GOOS: "windows", NewFromRev: "main", Out: &out, Err: io.Discard}
	if failures, _ := runGolangciTargets(opts); len(failures) != 0 {
		t.Fatalf("failures = %v, want none", failures)
	}
	want := "GOOS=windows GOWORK=off args=run --new-from-rev=main\n"
	if !strings.Contains(out.String(), want) {
		t.Errorf("stdout = %q, want it to contain %q", out.String(), want)
	}
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

func TestGolangciArgs(t *testing.T) {
	if got := strings.Join(golangciArgs(""), " "); got != "run" {
		t.Errorf("golangciArgs(\"\") = %q, want %q", got, "run")
	}
	want := "run --new-from-rev=main"
	if got := strings.Join(golangciArgs("main"), " "); got != want {
		t.Errorf("golangciArgs(\"main\") = %q, want %q", got, want)
	}
}

func TestHostToolPath_IgnoresInheritedTarget(t *testing.T) {
	want, err := hostToolPath("golangci-lint")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(want) {
		t.Fatalf("hostToolPath = %q, want an absolute path", want)
	}
	if info, err := os.Stat(want); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("hostToolPath = %q, want a regular file (stat error: %v)", want, err)
	}
	t.Setenv("GOOS", "windows")
	t.Setenv("GOARCH", "386")
	got, err := hostToolPath("golangci-lint")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("hostToolPath with GOOS/GOARCH set = %q, want %q", got, want)
	}
}

func TestHostToolPath_UnknownTool(t *testing.T) {
	if _, err := hostToolPath("definitely-not-a-go-tool"); err == nil {
		t.Fatal("expected error for unknown tool")
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
