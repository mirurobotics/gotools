package covgate

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/gocover"
	"github.com/mirurobotics/gotools/internal/testutil"
)

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

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := Run(Opts{Out: &buf})
	if err == nil {
		t.Fatal("expected error (no go module)")
	}
}

func TestRun_PublicWrapper_HappyPath(t *testing.T) {
	testutil.WriteTestModule(t,
		"package mypkg\n\nfunc Add(a, b int) int { return a + b }\n",
		"package mypkg\n\nimport \"testing\"\n\n"+
			"func TestAdd(t *testing.T) {\n"+
			"\tif Add(1, 2) != 3 { t.Fatal(\"Add broken\") }\n}\n")
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := Run(Opts{
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

func assertOutput(t *testing.T, out string, want, unwanted []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q:\n%s", w, out)
		}
	}
	for _, u := range unwanted {
		if strings.Contains(out, u) {
			t.Errorf("output unexpectedly contains %q:\n%s", u, out)
		}
	}
}

func TestMeasure_Pass(t *testing.T) {
	testutil.WriteTestModule(t,
		"package mypkg\n\nfunc Add(a, b int) int { return a + b }\n",
		"package mypkg\n\nimport \"testing\"\n\n"+
			"func TestAdd(t *testing.T) {\n\tif Add(1,2) != 3 { t.Fatal() }\n}\n")
	cov, _, err := gocover.Measure("testmod/mypkg", []string{"testmod/mypkg"})
	if err != nil {
		t.Fatalf("Measure failed: %v", err)
	}
	if cov < 100.0 {
		t.Errorf("expected 100%% coverage, got %.1f%%", cov)
	}
}

func TestMeasure_TestFailure(t *testing.T) {
	testutil.WriteTestModule(t,
		"package mypkg\n",
		"package mypkg\n\nimport \"testing\"\n\n"+
			"func TestFail(t *testing.T) { t.Fatal(\"fail\") }\n")
	_, output, err := gocover.Measure("testmod/mypkg", []string{"testmod/mypkg"})
	if err == nil {
		t.Fatal("expected error from failing test")
	}
	if len(output) == 0 {
		t.Error("expected non-empty output")
	}
}
