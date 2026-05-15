package covratchet

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestReadCovgate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		exists  bool
		want    string
	}{
		{"valid value", "85.5\n", true, "85.5"},
		{"zero", "0\n", true, "0"},
		{"missing file", "", false, ""},
		{"empty file", "", true, ""},
		{"with trailing space", "90.0  \n", true, "90.0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".covgate")
			if tc.exists {
				//nolint:gosec // G306: test file
				err := os.WriteFile(path, []byte(tc.content), 0o644)
				if err != nil {
					t.Fatal(err)
				}
			}
			got := readCovgate(path)
			if got != tc.want {
				t.Errorf("readCovgate() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWriteCovgate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".covgate")

	if err := writeCovgate(path, 85.5); err != nil {
		t.Fatal(err)
	}

	//nolint:gosec // G304: test file read
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "85.5\n" {
		t.Errorf("wrote %q, want %q", got, "85.5\n")
	}
}

func TestWriteCovgate_NoLeftoverTempfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".covgate")
	if err := writeCovgate(path, 42.5); err != nil {
		t.Fatalf("writeCovgate: %v", err)
	}
	//nolint:gosec // G304: test file read
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .covgate: %v", err)
	}
	if string(content) != "42.5\n" {
		t.Fatalf("got %q, want %q", string(content), "42.5\n")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".covgate.tmp-") {
			t.Fatalf("tempfile leaked: %s", e.Name())
		}
	}
}

func TestWriteCovgate_PreservesExistingOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".covgate")
	//nolint:gosec // G306: test file
	if err := os.WriteFile(path, []byte("55.0\n"), 0o644); err != nil {
		t.Fatalf("seed .covgate: %v", err)
	}
	//nolint:gosec // G302: test file permissions
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	//nolint:gosec // G302: test file permissions
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if err := writeCovgate(path, 99.9); err == nil {
		t.Fatalf("expected error on read-only dir, got nil")
	}
	//nolint:gosec // G302: test file permissions
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("restore dir mode: %v", err)
	}
	//nolint:gosec // G304: test file read
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .covgate: %v", err)
	}
	if string(content) != "55.0\n" {
		t.Fatalf("original .covgate clobbered: got %q, want %q", string(content), "55.0\n")
	}
}

func TestPrintHeader(t *testing.T) {
	var buf bytes.Buffer
	printHeader(&buf)
	out := buf.String()

	cols := []string{"STATUS", "PREVIOUS", "CURRENT", "PACKAGE"}
	for _, col := range cols {
		if !strings.Contains(out, col) {
			t.Errorf("output missing column %q", col)
		}
	}
	if !strings.Contains(out, "------") {
		t.Error("output missing separator line")
	}
}

func TestWriteCovgate_Error(t *testing.T) {
	err := writeCovgate("/nonexistent/dir/.covgate", 50.0)
	if err == nil {
		t.Error("expected error writing to bad dir")
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

func TestRatchetPackage_MeasureError(t *testing.T) {
	testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		measure: func(string, []string) (float64, []byte, error) {
			return 0, nil, fmt.Errorf("tests failed")
		},
	}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 0 || res.unchanged != 0 || res.failed != 1 {
		t.Errorf("got (%d,%d,%d), want (0,0,1)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "FAIL") {
		t.Errorf("missing FAIL: %s", res.output)
	}
}

func TestRatchetPackage_New(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 1 || res.unchanged != 0 || res.failed != 0 {
		t.Errorf("got (%d,%d,%d), want (1,0,0)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "NEW") {
		t.Errorf("missing NEW: %s", res.output)
	}

	path := filepath.Join(dir, ".covgate")
	//nolint:gosec // G304: test file read
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "85.0\n" {
		t.Errorf(".covgate = %q, want %q", got, "85.0\n")
	}
}

func TestRatchetPackage_New_WriteError(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)

	//nolint:gosec // G302: test file permissions
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // G302: test file permissions
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 0 || res.unchanged != 0 || res.failed != 1 {
		t.Errorf("got (%d,%d,%d), want (0,0,1)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "FAIL") {
		t.Errorf("missing FAIL: %s", res.output)
	}
}

func TestRatchetPackage_FromZero(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 1 || res.unchanged != 0 || res.failed != 0 {
		t.Errorf("got (%d,%d,%d), want (1,0,0)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "UP") {
		t.Errorf("missing UP: %s", res.output)
	}

	path := filepath.Join(dir, ".covgate")
	//nolint:gosec // G304: test file read
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "85.0\n" {
		t.Errorf(".covgate = %q, want %q", got, "85.0\n")
	}
}

func TestRatchetPackage_Up(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "70.0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 1 || res.unchanged != 0 || res.failed != 0 {
		t.Errorf("got (%d,%d,%d), want (1,0,0)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "UP") {
		t.Errorf("missing UP: %s", res.output)
	}

	path := filepath.Join(dir, ".covgate")
	//nolint:gosec // G304: test file read
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "85.0\n" {
		t.Errorf(".covgate = %q, want %q", got, "85.0\n")
	}
}

func TestRatchetPackage_Up_WriteError(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "70.0\n")

	//nolint:gosec // G302: test file permissions
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // G302: test file permissions
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 0 || res.unchanged != 0 || res.failed != 1 {
		t.Errorf("got (%d,%d,%d), want (0,0,1)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "FAIL") {
		t.Errorf("missing FAIL: %s", res.output)
	}
}

func TestRatchetPackage_CorruptedCovgate(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "notanumber\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 0 || res.unchanged != 0 || res.failed != 1 {
		t.Errorf("got (%d,%d,%d), want (0,0,1)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "FAIL") {
		t.Errorf("missing FAIL in output: %s", res.output)
	}

	covFile := filepath.Join(dir, ".covgate")
	//nolint:gosec // G304: test file read
	data, err := os.ReadFile(covFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "notanumber\n" {
		t.Errorf(".covgate was overwritten: got %q, want %q", got, "notanumber\n")
	}
}

func TestRatchetPackage_Ok(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "90.0\n")

	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	res := r.ratchetPackage(pkgName, modName, "", "")
	if res.updated != 0 || res.unchanged != 1 || res.failed != 0 {
		t.Errorf("got (%d,%d,%d), want (0,1,0)", res.updated, res.unchanged, res.failed)
	}
	if !strings.Contains(res.output, "OK") {
		t.Errorf("missing OK: %s", res.output)
	}
}

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
	err := r.run(Opts{Out: &buf, Parallelism: 2})
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
	// Verify aggregate counts are accumulated correctly.
	if !strings.Contains(out, "Updated: 3") {
		t.Errorf("expected Updated: 3 in output: %s", out)
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
	err := r.run(Opts{Out: &buf, Parallelism: 0})
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
	err = Run(Opts{Packages: "testmod/...", Out: &buf, Parallelism: 1})
	if err != nil {
		t.Fatalf("Run: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "Done.") {
		t.Errorf("missing 'Done.' summary: %s", buf.String())
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
