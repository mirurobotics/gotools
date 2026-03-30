package covratchet

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		measure: func(string, []string) (float64, []byte, error) {
			return 0, nil, fmt.Errorf("tests failed")
		},
	}

	u, unch, f := r.ratchetPackage(pkgName, modName, "", "", &buf)
	if u != 0 || unch != 0 || f != 1 {
		t.Errorf("got (%d,%d,%d), want (0,0,1)", u, unch, f)
	}
	if !strings.Contains(buf.String(), "FAIL") {
		t.Errorf("missing FAIL: %s", buf.String())
	}
}

func TestRatchetPackage_New(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	u, unch, f := r.ratchetPackage(pkgName, modName, "", "", &buf)
	if u != 1 || unch != 0 || f != 0 {
		t.Errorf("got (%d,%d,%d), want (1,0,0)", u, unch, f)
	}
	if !strings.Contains(buf.String(), "NEW") {
		t.Errorf("missing NEW: %s", buf.String())
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

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	u, unch, f := r.ratchetPackage(pkgName, modName, "", "", &buf)
	if u != 0 || unch != 0 || f != 1 {
		t.Errorf("got (%d,%d,%d), want (0,0,1)", u, unch, f)
	}
	if !strings.Contains(buf.String(), "FAIL") {
		t.Errorf("missing FAIL: %s", buf.String())
	}
}

func TestRatchetPackage_FromZero(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "0\n")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	u, unch, f := r.ratchetPackage(pkgName, modName, "", "", &buf)
	if u != 1 || unch != 0 || f != 0 {
		t.Errorf("got (%d,%d,%d), want (1,0,0)", u, unch, f)
	}
	if !strings.Contains(buf.String(), "UP") {
		t.Errorf("missing UP: %s", buf.String())
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

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	u, unch, f := r.ratchetPackage(pkgName, modName, "", "", &buf)
	if u != 1 || unch != 0 || f != 0 {
		t.Errorf("got (%d,%d,%d), want (1,0,0)", u, unch, f)
	}
	if !strings.Contains(buf.String(), "UP") {
		t.Errorf("missing UP: %s", buf.String())
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

	covFile := filepath.Join(dir, ".covgate")
	//nolint:gosec // G302: test file permissions
	if err := os.Chmod(covFile, 0o444); err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // G302: test file permissions
	t.Cleanup(func() { _ = os.Chmod(covFile, 0o644) })

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	u, unch, f := r.ratchetPackage(pkgName, modName, "", "", &buf)
	if u != 0 || unch != 0 || f != 1 {
		t.Errorf("got (%d,%d,%d), want (0,0,1)", u, unch, f)
	}
	if !strings.Contains(buf.String(), "FAIL") {
		t.Errorf("missing FAIL: %s", buf.String())
	}
}

func TestRatchetPackage_Ok(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "90.0\n")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{measure: fakeMeasure(85.0)}

	u, unch, f := r.ratchetPackage(pkgName, modName, "", "", &buf)
	if u != 0 || unch != 1 || f != 0 {
		t.Errorf("got (%d,%d,%d), want (0,1,0)", u, unch, f)
	}
	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("missing OK: %s", buf.String())
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
