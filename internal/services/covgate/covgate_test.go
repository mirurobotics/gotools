package covgate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/gocover"
	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestPrintHeader(t *testing.T) {
	var buf bytes.Buffer
	printHeader(&buf)
	out := buf.String()

	cols := []string{"STATUS", "COVERAGE", "REQUIRED", "PACKAGE"}
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

func TestRun_Parallelism_DefaultsToNumCPU(t *testing.T) {
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
		t.Fatalf("unexpected error with Parallelism=0 (NumCPU): %v", err)
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
