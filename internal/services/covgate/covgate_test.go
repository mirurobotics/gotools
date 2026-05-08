package covgate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestPrintHeader(t *testing.T) {
	var buf bytes.Buffer
	printHeader(&buf)
	out := buf.String()

	cols := []string{"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE"}
	for _, col := range cols {
		if !strings.Contains(out, col) {
			t.Errorf("output missing column %q", col)
		}
	}
	if !strings.Contains(out, "------") {
		t.Error("output missing separator line")
	}
}

const (
	modName = "example.com/mod"
	pkgName = "example.com/mod/internal/foo"
	pkgRel  = "internal/foo"
)

func fakeMeasure(cov float64) func(string, []string) (float64, []byte, error) {
	return func(string, []string) (float64, []byte, error) { return cov, nil, nil }
}

func TestCheckPackage_Pass(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "75.0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{module: modName, threshold: 80.0})
	if !res.passed {
		t.Error("expected pass")
	}
	if !strings.Contains(res.output, "PASS") {
		t.Errorf("output missing PASS: %s", res.output)
	}
}

func TestCheckPackage_Fail_BelowThreshold(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "90.0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{module: modName, threshold: 80.0})
	if res.passed {
		t.Error("expected fail")
	}
	if !strings.Contains(res.output, "FAIL") {
		t.Errorf("output missing FAIL: %s", res.output)
	}
}

func TestCheckPackage_Fail_TestError(t *testing.T) {
	testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		measure: func(string, []string) (float64, []byte, error) {
			return 0, []byte("compile error\n"),
				fmt.Errorf("exit 1")
		},
	}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{module: modName, threshold: 80.0})
	if res.passed {
		t.Error("expected fail")
	}
	if !strings.Contains(res.output, "FAIL") {
		t.Errorf("output missing FAIL: %s", res.output)
	}
	if !strings.Contains(res.output, "tests failed") {
		t.Errorf("missing 'tests failed': %s", res.output)
	}
	if !strings.Contains(res.output, "compile error") {
		t.Errorf("missing test output: %s", res.output)
	}
}

func TestCheckPackage_DefaultThreshold(t *testing.T) {
	testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{module: modName, threshold: 80.0})
	if !res.passed {
		t.Error("expected pass with default threshold")
	}
	if !strings.Contains(res.output, "80.0%") {
		t.Errorf("missing default threshold 80.0%%: %s", res.output)
	}
}

func TestRun_AllPass(t *testing.T) {
	testutil.MakePkgDir(t, "pkg/a")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{"example.com/mod/pkg/a"}, nil
		},
		measure: fakeMeasure(90.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "All packages meet") {
		t.Errorf("missing success msg: %s", buf.String())
	}
}

func TestRun_WithFailure(t *testing.T) {
	dir := testutil.MakePkgDir(t, "pkg/a")
	testutil.WriteCovgateFile(t, dir, "95.0\n")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{"example.com/mod/pkg/a"}, nil
		},
		measure: fakeMeasure(50.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(buf.String(), "ERROR: One or more") {
		t.Errorf("missing error msg: %s", buf.String())
	}
}

func TestRun_Parallelism(t *testing.T) {
	// Use a single temp dir so all three packages share the same cwd.
	tmp := t.TempDir()
	t.Chdir(tmp)
	for _, rel := range []string{"pkg/a", "pkg/b", "pkg/c"} {
		//nolint:gosec // G301: test directory
		if err := os.MkdirAll(filepath.Join(tmp, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{
				modName + "/pkg/a",
				modName + "/pkg/b",
				modName + "/pkg/c",
			}, nil
		},
		measure: fakeMeasure(90.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "pkg/a") {
		t.Errorf("output missing pkg/a: %s", out)
	}
	if !strings.Contains(out, "pkg/b") {
		t.Errorf("output missing pkg/b: %s", out)
	}
	if !strings.Contains(out, "pkg/c") {
		t.Errorf("output missing pkg/c: %s", out)
	}
	// Verify output order is preserved: a before b before c.
	idxA := strings.Index(out, "pkg/a")
	idxB := strings.Index(out, "pkg/b")
	idxC := strings.Index(out, "pkg/c")
	if idxA >= idxB || idxB >= idxC {
		t.Errorf(
			"output order not preserved: a=%d b=%d c=%d\n%s",
			idxA, idxB, idxC, out,
		)
	}
}

func TestRun_Parallelism_DefaultsToGOMAXPROCS(t *testing.T) {
	testutil.MakePkgDir(t, "pkg/a")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{modName + "/pkg/a"}, nil
		},
		measure: fakeMeasure(90.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 0})
	if err != nil {
		t.Fatalf("unexpected error with Parallelism=0 (GOMAXPROCS): %v", err)
	}
}

func TestEffectiveParallelism(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	if got := effectiveParallelism(Opts{Parallelism: 4}); got != 4 {
		t.Errorf("effectiveParallelism(4) = %d, want 4", got)
	}
	want := runtime.GOMAXPROCS(0)
	//nolint:exhaustruct // test uses partial initialization
	if got := effectiveParallelism(Opts{Parallelism: 0}); got != want {
		t.Errorf("effectiveParallelism(0) = %d, want %d", got, want)
	}
}

func TestChildGOMAXPROCS(t *testing.T) {
	if got := childGOMAXPROCS(1 << 30); got != 1 {
		t.Errorf("childGOMAXPROCS(1<<30) = %d, want 1 (clamped)", got)
	}
	want := runtime.GOMAXPROCS(0)
	if got := childGOMAXPROCS(1); got != want {
		t.Errorf("childGOMAXPROCS(1) = %d, want %d", got, want)
	}
}

func TestMeasure_Pass(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	pkg := filepath.Join(tmp, "mypkg")
	//nolint:gosec // G301: test directory
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := "module testmod\n\ngo 1.23\n"
	//nolint:gosec // G306: test file
	err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	lib := "package mypkg\n\n" +
		"func Add(a, b int) int { return a + b }\n"
	//nolint:gosec // G306: test file
	err = os.WriteFile(filepath.Join(pkg, "lib.go"), []byte(lib), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	test := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestAdd(t *testing.T) {\n" +
		"\tif Add(1,2) != 3 { t.Fatal() }\n}\n"
	//nolint:gosec // G306: test file
	err = os.WriteFile(filepath.Join(pkg, "lib_test.go"), []byte(test), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cov, _, err := gocover.Measure("testmod/mypkg", []string{"testmod/mypkg"})
	if err != nil {
		t.Fatalf("Measure failed: %v", err)
	}
	if cov < 100.0 {
		t.Errorf("expected 100%% coverage, got %.1f%%", cov)
	}
}

func TestMeasure_TestFailure(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	pkg := filepath.Join(tmp, "mypkg")
	//nolint:gosec // G301: test directory
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := "module testmod\n\ngo 1.23\n"
	//nolint:gosec // G306: test file
	err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // G306: test file
	err = os.WriteFile(filepath.Join(pkg, "lib.go"), []byte("package mypkg\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	test := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestFail(t *testing.T) " +
		"{ t.Fatal(\"fail\") }\n"
	//nolint:gosec // G306: test file
	err = os.WriteFile(filepath.Join(pkg, "lib_test.go"), []byte(test), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	_, output, err := gocover.Measure("testmod/mypkg", []string{"testmod/mypkg"})
	if err == nil {
		t.Fatal("expected error from failing test")
	}
	if len(output) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestRun_NilWriter(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return "", fmt.Errorf("stop") },
	}
	//nolint:exhaustruct // test uses partial initialization
	_ = r.run(Opts{Out: nil})
}

func TestRun_PublicWrapper(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	// No go.mod → Run will fail at goModule, covering the
	// public wrapper.
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := Run(Opts{Out: &buf})
	if err == nil {
		t.Fatal("expected error (no go module)")
	}
}

func TestRun_PublicWrapper_HappyPath(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	goMod := "module testmod\n\ngo 1.23\n"
	//nolint:gosec // G306: test file
	err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(tmp, "mypkg")
	//nolint:gosec // G301: test directory
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	lib := "package mypkg\n\n" +
		"func Add(a, b int) int { return a + b }\n"
	//nolint:gosec // G306: test file
	err = os.WriteFile(filepath.Join(pkg, "lib.go"), []byte(lib), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	testSrc := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestAdd(t *testing.T) {\n" +
		"\tif Add(1, 2) != 3 { t.Fatal(\"Add broken\") }\n}\n"
	//nolint:gosec // G306: test file
	err = os.WriteFile(filepath.Join(pkg, "lib_test.go"), []byte(testSrc), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err = Run(Opts{
		Packages:         "testmod/...",
		DefaultThreshold: 80.0,
		Out:              &buf,
		Parallelism:      1,
	})
	if err != nil {
		t.Fatalf("Run: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "All packages meet") {
		t.Errorf("missing success msg: %s", buf.String())
	}
}

func TestRun_GoModuleError(t *testing.T) {
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return "", fmt.Errorf("no module") },
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no module") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_GoListError(t *testing.T) {
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return "mod", nil },
		goListPackages: func(string) ([]string, error) {
			return nil, fmt.Errorf("list failed")
		},
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0.0s"},
		{500 * time.Millisecond, "0.5s"},
		{3200 * time.Millisecond, "3.2s"},
		{60 * time.Second, "1m00s"},
		{7*time.Minute + 45*time.Second, "7m45s"},
	}
	for _, tt := range tests {
		if got := fmtDuration(tt.d); got != tt.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestRun_OutputContainsTiming(t *testing.T) {
	testutil.MakePkgDir(t, "pkg/a")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{"example.com/mod/pkg/a"}, nil
		},
		measure: fakeMeasure(90.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := buf.String()
	if !strings.Contains(s, "TIME") {
		t.Errorf("output missing TIME column header: %s", s)
	}
	if !strings.Contains(s, "Total time:") {
		t.Errorf("output missing total time: %s", s)
	}
}

func TestPrintResults_UsesWallTime(t *testing.T) {
	// Each result claims 5s duration, but the wall-clock parameter is only 3s.
	// The printed total must show 3.0s (wall time), not 15.0s (sum of durations).
	results := []checkResult{
		{output: "line1\n", passed: true, duration: 5 * time.Second},
		{output: "line2\n", passed: true, duration: 5 * time.Second},
		{output: "line3\n", passed: true, duration: 5 * time.Second},
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{}
	err := r.printResults(&buf, results, 3*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total time: 3.0s") {
		t.Errorf("expected wall-clock total 3.0s, got: %s", out)
	}
	if strings.Contains(out, "15.0s") {
		t.Errorf("total should not be sum of durations (15.0s): %s", out)
	}
}

func TestCheckPackage_Loose_Fires(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "10.0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if res.passed {
		t.Error("expected fail for loose threshold")
	}
	if !strings.Contains(res.output, "LOOSE") {
		t.Errorf("output missing LOOSE: %s", res.output)
	}
	if !strings.Contains(res.output, "70.0pp") {
		t.Errorf("output missing 70.0pp gap: %s", res.output)
	}
	if !strings.Contains(res.output, ">= 79.5") {
		t.Errorf("output missing recommended 79.5: %s", res.output)
	}
}

func TestCheckPackage_Loose_WithinTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "79.6\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass within tolerance: %s", res.output)
	}
	if !strings.Contains(res.output, "PASS") {
		t.Errorf("output missing PASS: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE in output: %s", res.output)
	}
}

func TestCheckPackage_Loose_AtExactTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "79.5\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass at exact tolerance boundary: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE at boundary: %s", res.output)
	}
}

func TestCheckPackage_Loose_JustOverTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "79.4\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if res.passed {
		t.Errorf("expected fail just over tolerance: %s", res.output)
	}
	if !strings.Contains(res.output, "LOOSE") {
		t.Errorf("output missing LOOSE: %s", res.output)
	}
}

func TestCheckPackage_Loose_ZeroCoverageAllowed(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(0.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass for 0/0: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE for zero coverage: %s", res.output)
	}
}

func TestCheckPackage_Loose_NoCovgateFile_UsesDefault(t *testing.T) {
	testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          10.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass when using default threshold: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE for default-fallback package: %s", res.output)
	}
}

func TestCheckPackage_Loose_Disabled(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "10.0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   false,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass when tightness disabled: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE when tightness disabled: %s", res.output)
	}
}

func TestCheckPackage_CustomTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "70.0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 15.0,
	})
	if !res.passed {
		t.Errorf("expected pass within custom tolerance: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE within custom tolerance: %s", res.output)
	}
}

func TestRun_LooseFailsOverall(t *testing.T) {
	dir := testutil.MakePkgDir(t, "pkg/a")
	testutil.WriteCovgateFile(t, dir, "10.0\n")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{"example.com/mod/pkg/a"}, nil
		},
		measure: fakeMeasure(80.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:                &buf,
		DefaultThreshold:   50.0,
		TightnessEnabled:   true,
		TightnessTolerance: 0.5,
	})
	if err == nil {
		t.Fatal("expected error from loose threshold")
	}
	out := buf.String()
	if !strings.Contains(out, "LOOSE") {
		t.Errorf("output missing LOOSE: %s", out)
	}
	if !strings.Contains(out, "have loose .covgate thresholds") {
		t.Errorf("output missing updated error message: %s", out)
	}
}

func TestCheckPackage_OutputContainsTime(t *testing.T) {
	testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := r.checkPackage(pkgName, checkPackageCtx{module: modName, threshold: 80.0})
	if !strings.Contains(res.output, "0.0s") {
		t.Errorf("output missing duration: %s", res.output)
	}
	if res.duration < 0 {
		t.Errorf("expected non-negative duration, got %v", res.duration)
	}
}
