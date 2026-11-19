package covratchet

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestRun_Success(t *testing.T) {
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
	err := r.run(Opts{Out: &buf})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Done.") {
		t.Errorf("missing summary: %s", out)
	}
	if !strings.Contains(out, "Updated: 1") {
		t.Errorf("expected 1 updated: %s", out)
	}
}

func TestRun_WithFailures(t *testing.T) {
	testutil.MakePkgDir(t, "pkg/a")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{modName + "/pkg/a"}, nil
		},
		measure: func(string, []string) (float64, []byte, error) {
			return 0, nil, fmt.Errorf("tests failed")
		},
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(buf.String(), "Failed: 1") {
		t.Errorf("missing failure count: %s", buf.String())
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

func TestRun_PublicWrapper_HappyPath(t *testing.T) {
	testutil.WriteTestModule(t,
		"package mypkg\n\nfunc Add(a, b int) int { return a + b }\n",
		"package mypkg\n\nimport \"testing\"\n\n"+
			"func TestAdd(t *testing.T) {\n"+
			"\tif Add(1, 2) != 3 { t.Fatal(\"Add broken\") }\n}\n")
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := Run(Opts{Packages: "testmod/...", Out: &buf, Parallelism: 1})
	if err != nil {
		t.Fatalf("Run: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "Done.") {
		t.Errorf("missing 'Done.' summary: %s", buf.String())
	}
}
