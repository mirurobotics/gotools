package covgate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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

func TestRun_Exclude(t *testing.T) {
	const (
		pkgA = "example.com/mod/pkg/a"
		pkgB = "example.com/mod/pkg/b"
		pkgC = "example.com/mod/pkg/c"
	)
	allThree := []string{pkgA, pkgB, pkgC}

	cases := []struct {
		name           string
		exclude        string
		lookup         map[string][]string
		wantContains   []string
		wantNotContain []string
		extra          func(t *testing.T, out string)
	}{
		{
			name:           "NoExclude",
			exclude:        "",
			lookup:         map[string][]string{"./...": allThree},
			wantContains:   []string{"pkg/a", "pkg/b", "pkg/c"},
			wantNotContain: []string{"Excluded", "SKIPPED"},
			extra:          nil,
		},
		{
			name:    "Subset",
			exclude: "./pkg/b",
			lookup:  map[string][]string{"./...": allThree, "./pkg/b": {pkgB}},
			wantContains: []string{
				"pkg/a",
				"pkg/b",
				"pkg/c",
				"SKIPPED",
				"Excluded 1 package(s) from coverage measurement",
				"All packages meet minimum coverage requirement",
			},
			wantNotContain: []string{"FAIL"},
			extra:          nil,
		},
		{
			name:    "NoOpPattern",
			exclude: "./does-not-exist/...",
			lookup: map[string][]string{
				"./...":                allThree,
				"./does-not-exist/...": {},
			},
			wantContains:   []string{"pkg/a", "pkg/b", "pkg/c"},
			wantNotContain: []string{"Excluded", "SKIPPED"},
			extra:          nil,
		},
		{
			// Includes empty entries before, between, and after
			// non-empty ones so the trim/skip-empty path is
			// exercised end-to-end.
			name:    "MultiplePatternsWithWhitespace",
			exclude: ", ./pkg/a, , ./pkg/c,",
			lookup: map[string][]string{
				"./...":   allThree,
				"./pkg/a": {pkgA},
				"./pkg/c": {pkgC},
			},
			wantContains: []string{
				"pkg/a",
				"pkg/b",
				"pkg/c",
				"SKIPPED",
				"Excluded 2 package(s) from coverage measurement",
			},
			wantNotContain: []string{"FAIL"},
			extra:          nil,
		},
		{
			name:    "AllPackages",
			exclude: "./...",
			lookup:  map[string][]string{"./...": allThree},
			wantContains: []string{
				"pkg/a",
				"pkg/b",
				"pkg/c",
				"SKIPPED",
				"Excluded 3 package(s) from coverage measurement",
				"All packages meet minimum coverage requirement",
				"Total time:",
			},
			wantNotContain: []string{"FAIL"},
			extra:          nil,
		},
		{
			name:           "OrderingAndCount",
			exclude:        "./pkg/b",
			lookup:         map[string][]string{"./...": allThree, "./pkg/b": {pkgB}},
			wantContains:   []string{"pkg/a", "pkg/b", "pkg/c", "SKIPPED"},
			wantNotContain: []string{"FAIL"},
			extra: func(t *testing.T, out string) {
				t.Helper()
				idxSkipped := strings.Index(out, "SKIPPED")
				idxA := strings.Index(out, "pkg/a")
				idxC := strings.Index(out, "pkg/c")
				if idxA < 0 || idxC < 0 || idxSkipped < 0 {
					t.Fatalf("expected pkg/a, pkg/c, and SKIPPED in output:\n%s", out)
				}
				if idxA >= idxSkipped {
					t.Errorf(
						"expected measured pkg/a row (%d) before SKIPPED block (%d):\n%s",
						idxA, idxSkipped, out,
					)
				}
				if idxC >= idxSkipped {
					t.Errorf(
						"expected measured pkg/c row (%d) before SKIPPED block (%d):\n%s",
						idxC, idxSkipped, out,
					)
				}
				if got := strings.Count(out, "SKIPPED"); got != 1 {
					t.Errorf("expected exactly 1 SKIPPED, got %d:\n%s", got, out)
				}
				var skippedLine string
				for line := range strings.SplitSeq(out, "\n") {
					if strings.Contains(line, "pkg/b") {
						skippedLine = line
						break
					}
				}
				if skippedLine == "" {
					t.Fatalf("no line contained pkg/b:\n%s", out)
				}
				if !strings.Contains(skippedLine, "SKIPPED") {
					t.Errorf("pkg/b line missing SKIPPED: %q", skippedLine)
				}
				if strings.Count(skippedLine, "---") != 3 {
					t.Errorf(
						"pkg/b SKIPPED line expected 3 '---' placeholders, got %d: %q",
						strings.Count(skippedLine, "---"), skippedLine,
					)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lookup := tc.lookup
			var buf bytes.Buffer
			//nolint:exhaustruct // test uses partial initialization
			r := runner{
				goModule: func() (string, error) { return modName, nil },
				goListPackages: func(pattern string) ([]string, error) {
					out, ok := lookup[pattern]
					if !ok {
						return nil, fmt.Errorf("unexpected pattern %q", pattern)
					}
					return out, nil
				},
				measure: fakeMeasure(90.0),
				prewarm: func([]string, []string) error { return nil },
			}

			//nolint:exhaustruct // test uses partial initialization
			err := r.run(Opts{
				Out:              &buf,
				DefaultThreshold: 80.0,
				Packages:         "./...",
				Exclude:          tc.exclude,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			out := buf.String()
			for _, want := range tc.wantContains {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
			for _, unwanted := range tc.wantNotContain {
				if strings.Contains(out, unwanted) {
					t.Errorf("output unexpectedly contains %q:\n%s", unwanted, out)
				}
			}
			if tc.extra != nil {
				tc.extra(t, out)
			}
		})
	}
}

func TestRun_Exclude_GoListError(t *testing.T) {
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(pattern string) ([]string, error) {
			if pattern == "./..." {
				return []string{"example.com/mod/pkg/a"}, nil
			}
			return nil, fmt.Errorf("list failed")
		},
		measure: fakeMeasure(90.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		Packages:         "./...",
		Exclude:          "./pkg/bogus",
	})
	if err == nil {
		t.Fatal("expected error from exclude pattern resolution")
	}
	msg := err.Error()
	if !strings.Contains(msg, "./pkg/bogus") {
		t.Errorf("error missing pattern %q: %v", "./pkg/bogus", err)
	}
	if !strings.Contains(msg, "list failed") {
		t.Errorf("error missing wrapped cause %q: %v", "list failed", err)
	}
	if !strings.Contains(msg, "resolve exclude") {
		t.Errorf("error missing context prefix %q: %v", "resolve exclude", err)
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
		prewarm: func([]string, []string) error { return nil },
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

func TestEffectiveParallelism(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	if got := effectiveParallelism(Opts{Parallelism: 4}); got != 4 {
		t.Errorf("effectiveParallelism(4) = %d, want 4", got)
	}
	want := runtime.NumCPU()
	//nolint:exhaustruct // test uses partial initialization
	if got := effectiveParallelism(Opts{Parallelism: 0}); got != want {
		t.Errorf("effectiveParallelism(0) = %d, want %d", got, want)
	}
}

func TestRun_EmitsProgress_WhenAutoParallelism(t *testing.T) {
	// Use a single temp dir so both packages share the same cwd.
	tmp := t.TempDir()
	t.Chdir(tmp)
	for _, rel := range []string{"pkg/a", "pkg/b"} {
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
			return []string{modName + "/pkg/a", modName + "/pkg/b"}, nil
		},
		measure:      fakeMeasure(90.0),
		emitProgress: true,
		prewarm:      func([]string, []string) error { return nil },
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !hasProgressLineWith(out, "[1/2]", "90.0%") {
		t.Errorf(
			"expected [1/2] progress line to include coverage "+
				"data (90.0%%):\n%s", out,
		)
	}
	if !hasProgressLineWith(out, "[2/2]", "90.0%") {
		t.Errorf(
			"expected [2/2] progress line to include coverage "+
				"data (90.0%%):\n%s", out,
		)
	}
	if !strings.Contains(out, "COUNT") {
		t.Errorf("expected progress header to include COUNT column:\n%s", out)
	}
	// STATUS header should appear exactly once — the progress
	// stream is the single canonical table.
	if got := strings.Count(out, "STATUS"); got != 1 {
		t.Errorf(
			"expected exactly one STATUS header (progress IS "+
				"the table); got %d:\n%s", got, out,
		)
	}
	// Leading announcement should appear before the COUNT header.
	idxAnnounce := strings.Index(out, "Running 2 packages with parallelism=")
	idxHeader := strings.Index(out, "COUNT")
	if idxAnnounce < 0 || idxHeader < 0 || idxAnnounce >= idxHeader {
		t.Errorf(
			"expected leading announcement before COUNT header "+
				"(announce=%d, header=%d):\n%s",
			idxAnnounce, idxHeader, out,
		)
	}
	// Both progress lines should appear before the final
	// "All packages meet" line that closes the table.
	idxProgress1 := strings.Index(out, "[1/2]")
	idxProgress2 := strings.Index(out, "[2/2]")
	idxFinal := strings.Index(out, "All packages meet")
	if idxProgress1 >= idxFinal || idxProgress2 >= idxFinal {
		t.Errorf(
			"expected progress lines before final summary "+
				"(p1=%d, p2=%d, final=%d):\n%s",
			idxProgress1, idxProgress2, idxFinal, out,
		)
	}
}

// hasProgressLineWith reports whether out contains a single line
// that matches both the count marker and an additional substring
// (used to verify progress lines carry full row data, not just the
// counter).
func hasProgressLineWith(out, marker, contains string) bool {
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, marker) && strings.Contains(line, contains) {
			return true
		}
	}
	return false
}

func TestRun_SuppressesProgress_WhenExplicitParallelism(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	for _, rel := range []string{"pkg/a", "pkg/b"} {
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
			return []string{modName + "/pkg/a", modName + "/pkg/b"}, nil
		},
		measure:      fakeMeasure(90.0),
		emitProgress: false,
		prewarm:      func([]string, []string) error { return nil },
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "[1/2]") || strings.Contains(out, "[2/2]") {
		t.Errorf("output unexpectedly contains progress prefix:\n%s", out)
	}
	if strings.Contains(out, "Running 2 packages with parallelism=") {
		t.Errorf("output unexpectedly contains leading announcement:\n%s", out)
	}
	if strings.Contains(out, "COUNT") {
		t.Errorf("output unexpectedly contains COUNT progress header:\n%s", out)
	}
	if got := strings.Count(out, "STATUS"); got != 1 {
		t.Errorf(
			"expected exactly one STATUS header (no progress "+
				"section); got %d:\n%s", got, out,
		)
	}
}

func TestRun_Prewarm_InvokedOnce_WhenParallelAndMultiPkg(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	for _, rel := range []string{"pkg/a", "pkg/b", "pkg/c"} {
		//nolint:gosec // G301: test directory
		if err := os.MkdirAll(filepath.Join(tmp, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	pkgs := []string{modName + "/pkg/a", modName + "/pkg/b", modName + "/pkg/c"}

	var (
		warmCalls      int
		warmCoverPaths []string
		warmPlainPaths []string
	)
	prewarm := func(coverPaths, plainPaths []string) error {
		warmCalls++
		warmCoverPaths = coverPaths
		warmPlainPaths = plainPaths
		return nil
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule:       func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) { return pkgs, nil },
		measure:        fakeMeasure(90.0),
		prewarm:        prewarm,
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if warmCalls != 1 {
		t.Fatalf("expected prewarm invoked exactly once, got %d", warmCalls)
	}
	// coverPaths is exactly the measured package import paths.
	if len(warmCoverPaths) != 3 {
		t.Fatalf(
			"expected 3 cover paths, got %d: %v",
			len(warmCoverPaths), warmCoverPaths,
		)
	}
	for _, pkg := range pkgs {
		if !slices.Contains(warmCoverPaths, pkg) {
			t.Errorf("cover paths missing %q: %v", pkg, warmCoverPaths)
		}
	}
	// With empty SrcPrefix/TestDir, BuildTestPaths returns just the
	// package path (no ./tests/... dirs), so plainPaths is empty.
	if len(warmPlainPaths) != 0 {
		t.Errorf(
			"expected no plain paths, got %d: %v",
			len(warmPlainPaths), warmPlainPaths,
		)
	}
	out := buf.String()
	wants := []string{
		"pkg/a", "pkg/b", "pkg/c", "Total time:",
		// Visibility: the warm pass announces itself and reports its
		// duration so its cost is observable separately from "Total time".
		"Pre-warming build cache (3 packages)...",
		"Pre-warm complete in ",
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestCollectWarmPaths_NoTestDirs(t *testing.T) {
	// With empty SrcPrefix/TestDir, BuildTestPaths returns just the
	// package path, so coverPaths is exactly pkgs and plainPaths is
	// empty (no ./tests/... dirs to compile plain).
	pkgA := modName + "/pkg/a"
	pkgB := modName + "/pkg/b"
	pkgs := []string{pkgA, pkgB}

	//nolint:exhaustruct // only module is needed for path collection
	coverPaths, plainPaths := collectWarmPaths(pkgs, checkPackageCtx{module: modName})

	if len(coverPaths) != len(pkgs) {
		t.Fatalf(
			"expected %d cover paths, got %d: %v",
			len(pkgs), len(coverPaths), coverPaths,
		)
	}
	for i := range pkgs {
		if coverPaths[i] != pkgs[i] {
			t.Errorf("cover path %d: expected %q, got %q", i, pkgs[i], coverPaths[i])
		}
	}
	if len(plainPaths) != 0 {
		t.Errorf("expected no plain paths, got %d: %v", len(plainPaths), plainPaths)
	}
}

func TestCollectWarmPaths_SplitsCoverFromPlainTestDirs(t *testing.T) {
	// When SrcPrefix/TestDir are set and a package's ./tests dir
	// exists, the measured package goes to coverPaths (instrumented)
	// and its external tests dir goes to plainPaths (non-instrumented).
	tmp := t.TempDir()
	t.Chdir(tmp)

	const srcPrefix = "internal"
	pkg := modName + "/internal/foo"
	relPkg := gocover.RelPkg(pkg, modName) // internal/foo
	// The package's source dir.
	//nolint:gosec // G301: test directory
	if err := os.MkdirAll(filepath.Join(tmp, relPkg), 0o755); err != nil {
		t.Fatal(err)
	}
	// The external integration-test dir: tests/<relPkg without srcPrefix>.
	testSub := strings.TrimPrefix(relPkg, srcPrefix+"/") // foo
	extDir := filepath.Join("tests", testSub)
	//nolint:gosec // G301: test directory
	if err := os.MkdirAll(filepath.Join(tmp, extDir), 0o755); err != nil {
		t.Fatal(err)
	}

	//nolint:exhaustruct // only the path-deriving fields are needed
	coverPaths, plainPaths := collectWarmPaths(
		[]string{pkg},
		checkPackageCtx{module: modName, srcPrefix: srcPrefix, testDir: "tests"},
	)

	wantCover := []string{pkg}
	if !slices.Equal(coverPaths, wantCover) {
		t.Errorf("coverPaths = %v, want %v", coverPaths, wantCover)
	}
	wantPlain := []string{"./" + extDir}
	if !slices.Equal(plainPaths, wantPlain) {
		t.Errorf("plainPaths = %v, want %v", plainPaths, wantPlain)
	}

	// A duplicated package yields the same external tests dir twice,
	// which the seen-set must collapse to a single plainPaths entry.
	//nolint:exhaustruct // only the path-deriving fields are needed
	_, dedupPlain := collectWarmPaths(
		[]string{pkg, pkg},
		checkPackageCtx{module: modName, srcPrefix: srcPrefix, testDir: "tests"},
	)
	if !slices.Equal(dedupPlain, wantPlain) {
		t.Errorf("deduplicated plainPaths = %v, want %v", dedupPlain, wantPlain)
	}
}

func TestRun_Prewarm_Skipped_WhenSinglePackageOrSerial(t *testing.T) {
	cases := []struct {
		name        string
		pkgs        []string
		parallelism int
	}{
		{
			name: "SingleParallelism",
			pkgs: []string{
				modName + "/pkg/a",
				modName + "/pkg/b",
				modName + "/pkg/c",
			},
			parallelism: 1,
		},
		{
			name:        "SinglePackage",
			pkgs:        []string{modName + "/pkg/a"},
			parallelism: 4,
		},
		{
			name:        "SinglePackageSerial",
			pkgs:        []string{modName + "/pkg/a"},
			parallelism: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Chdir(tmp)
			for _, pkg := range tc.pkgs {
				rel := gocover.RelPkg(pkg, modName)
				//nolint:gosec // G301: test directory
				if err := os.MkdirAll(filepath.Join(tmp, rel), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			warmCalls := 0
			var buf bytes.Buffer
			//nolint:exhaustruct // test uses partial initialization
			r := runner{
				goModule:       func() (string, error) { return modName, nil },
				goListPackages: func(string) ([]string, error) { return tc.pkgs, nil },
				measure:        fakeMeasure(90.0),
				prewarm: func([]string, []string) error {
					warmCalls++
					return nil
				},
			}

			//nolint:exhaustruct // test uses partial initialization
			err := r.run(Opts{
				Out:              &buf,
				DefaultThreshold: 80.0,
				Parallelism:      tc.parallelism,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if warmCalls != 0 {
				t.Errorf("expected prewarm skipped, got %d calls", warmCalls)
			}
			out := buf.String()
			rel := gocover.RelPkg(tc.pkgs[0], modName)
			if !strings.Contains(out, rel) {
				t.Errorf("output missing per-package row %q:\n%s", rel, out)
			}
			// When the warm pass is gated off, it must not announce itself.
			if strings.Contains(out, "Pre-warming build cache") {
				t.Errorf("unexpected pre-warm output when skipped:\n%s", out)
			}
		})
	}
}

func TestRun_Prewarm_ErrorPropagates(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	for _, rel := range []string{"pkg/a", "pkg/b", "pkg/c"} {
		//nolint:gosec // G301: test directory
		if err := os.MkdirAll(filepath.Join(tmp, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	measureCalls := 0
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
		measure: func(string, []string) (float64, []byte, error) {
			measureCalls++
			return 90.0, nil, nil
		},
		prewarm: func([]string, []string) error {
			return fmt.Errorf("prewarm build: boom")
		},
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 3})
	if err == nil {
		t.Fatal("expected error from prewarm build")
	}
	if !strings.Contains(err.Error(), "prewarm build") {
		t.Errorf("error missing 'prewarm build': %v", err)
	}
	if measureCalls != 0 {
		t.Errorf("expected measure never invoked, got %d calls", measureCalls)
	}
	if strings.Contains(buf.String(), "Total time:") {
		t.Errorf("expected run to return before printing results:\n%s", buf.String())
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
	err := r.printResults(&buf, results, nil, modName, 3*time.Second)
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
