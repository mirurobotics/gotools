package covgate

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestRun_OrphanTests_RunsUnclaimedPackage(t *testing.T) {
	testutil.MakePkgDir(t, "internal/foo")

	var ranPaths [][]string
	r, patterns := orphanRunner(func(paths []string) ([]byte, error) {
		ranPaths = append(ranPaths, paths)
		return nil, nil
	})

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		SrcPrefix:        "internal",
		TestDir:          "./tests",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*patterns) != 1 || (*patterns)[0] != "./tests/..." {
		t.Errorf("expected one test-dir listing of ./tests/..., got %v", *patterns)
	}
	if len(ranPaths) != 1 {
		t.Fatalf("expected exactly one orphan run, got %v", ranPaths)
	}
	if len(ranPaths[0]) != 1 || ranPaths[0][0] != "./tests/bar" {
		t.Errorf("expected orphan run of ./tests/bar, got %v", ranPaths[0])
	}
	assertOutput(t, buf.String(), []string{"tests/bar", "internal/foo"}, nil)
}

func TestRun_OrphanTests_FailureFailsGate(t *testing.T) {
	testutil.MakePkgDir(t, "internal/foo")

	r, _ := orphanRunner(func([]string) ([]byte, error) {
		return []byte("--- FAIL: TestBroken\n"), fmt.Errorf("exit status 1")
	})

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		SrcPrefix:        "internal",
		TestDir:          "./tests",
	})
	if err == nil {
		t.Fatal("expected error when an orphan test package fails")
	}
	out := buf.String()
	if !strings.Contains(out, "tests/bar (tests failed)") {
		t.Errorf("output missing orphan FAIL row: %s", out)
	}
	if !strings.Contains(out, "--- FAIL: TestBroken") {
		t.Errorf("output missing orphan test detail: %s", out)
	}
}

func TestRun_OrphanTests_NoTestDirSkipsLookup(t *testing.T) {
	testutil.MakePkgDir(t, "internal/foo")

	//nolint:exhaustruct // test uses partial initialization
	r := &runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{modName + "/internal/foo"}, nil
		},
		goListTestPackages: func(string) ([]string, error) {
			t.Error("goListTestPackages must not be called without --test-dir")
			return nil, nil
		},
		measure: fakeMeasure(90.0),
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, SrcPrefix: "internal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_OrphanTests_ClaimedPackageNotDuplicated(t *testing.T) {
	testutil.MakePkgDir(t, "internal/foo")

	r, _ := orphanRunner(func(paths []string) ([]byte, error) {
		if len(paths) == 1 && paths[0] == "./tests/foo" {
			t.Errorf("claimed test package scheduled as orphan: %v", paths)
		}
		return nil, nil
	})

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		SrcPrefix:        "internal",
		TestDir:          "./tests",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_OrphanTests_ExcludeApplies(t *testing.T) {
	testutil.MakePkgDir(t, "internal/foo")

	r, _ := orphanRunner(func(paths []string) ([]byte, error) {
		t.Errorf("excluded orphan must not run: %v", paths)
		return nil, nil
	})
	inner := r.goListPackages
	r.goListPackages = func(pattern string) ([]string, error) {
		if pattern == "./tests/bar" {
			return []string{modName + "/tests/bar"}, nil
		}
		return inner(pattern)
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		SrcPrefix:        "internal",
		TestDir:          "./tests",
		Exclude:          "./tests/bar",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "SKIPPED") {
		t.Errorf("expected excluded orphan to show as SKIPPED: %s", buf.String())
	}
}

func orphanRunner(runTests func([]string) ([]byte, error)) (*runner, *[]string) {
	var listedPatterns []string
	//nolint:exhaustruct // test uses partial initialization
	r := &runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{modName + "/internal/foo", modName + "/cmd/tool"}, nil
		},
		goListTestPackages: func(pattern string) ([]string, error) {
			listedPatterns = append(listedPatterns, pattern)
			return []string{modName + "/tests/foo", modName + "/tests/bar"}, nil
		},
		measure:  fakeMeasure(90.0),
		runTests: runTests,
	}
	return r, &listedPatterns
}

func TestFindOrphanTests_EmptySrcPrefixOrphansEverything(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := &runner{
		goListTestPackages: func(string) ([]string, error) {
			return []string{modName + "/tests/foo"}, nil
		},
	}

	//nolint:exhaustruct // test uses partial initialization
	orphans, err := r.findOrphanTests(
		[]string{modName + "/internal/foo"}, modName,
		Opts{TestDir: "tests", SrcPrefix: ""},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orphans) != 1 || orphans[0] != modName+"/tests/foo" {
		t.Errorf("expected tests/foo orphaned with no src-prefix, got %v", orphans)
	}
}

func TestRun_OrphanTests_ListError(t *testing.T) {
	testutil.MakePkgDir(t, "internal/foo")

	//nolint:exhaustruct // test uses partial initialization
	r := &runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{modName + "/internal/foo"}, nil
		},
		goListTestPackages: func(string) ([]string, error) {
			return nil, fmt.Errorf("go list failed")
		},
		measure: fakeMeasure(90.0),
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		SrcPrefix:        "internal",
		TestDir:          "./tests",
	})
	if err == nil {
		t.Fatal("expected error when the test-dir listing fails")
	}
}
