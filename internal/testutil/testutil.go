package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MakePkgDir chdirs into a temp dir and creates rel, returning its path.
func MakePkgDir(t *testing.T, rel string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	dir := filepath.Join(tmp, rel)
	//nolint:gosec // G301: test directory
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// WriteCovgateFile writes a .covgate threshold file in the given directory.
func WriteCovgateFile(t *testing.T, dir, val string) {
	t.Helper()
	path := filepath.Join(dir, ".covgate")
	//nolint:gosec // G306: test file
	err := os.WriteFile(path, []byte(val), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

// MakePkgDirs chdirs into a temp dir and creates each relative package dir.
func MakePkgDirs(t *testing.T, rels ...string) {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	for _, rel := range rels {
		//nolint:gosec // G301: test directory
		if err := os.MkdirAll(filepath.Join(tmp, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// AssertInOrder checks that each string appears in out, in the given order.
func AssertInOrder(t *testing.T, out string, ordered ...string) {
	t.Helper()
	prev, prevIdx := "", -1
	for _, s := range ordered {
		idx := strings.Index(out, s)
		if idx < 0 {
			t.Errorf("output missing %q:\n%s", s, out)
			return
		}
		if idx <= prevIdx {
			t.Errorf("expected %q (%d) after %q (%d):\n%s", s, idx, prev, prevIdx, out)
		}
		prev, prevIdx = s, idx
	}
}

// AssertRowsOnce checks that each package ends exactly one line of out.
func AssertRowsOnce(t *testing.T, out string, pkgs ...string) {
	t.Helper()
	for _, pkg := range pkgs {
		if got := strings.Count(out, pkg+"\n"); got != 1 {
			t.Errorf("expected %s row once, got %d:\n%s", pkg, got, out)
		}
	}
}

// WriteTestModule chdirs into a temp module whose mypkg holds lib and test.
func WriteTestModule(t *testing.T, lib, test string) {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	pkg := filepath.Join(tmp, "mypkg")
	//nolint:gosec // G301: test directory
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(tmp, "go.mod"):      "module testmod\n\ngo 1.23\n",
		filepath.Join(pkg, "lib.go"):      lib,
		filepath.Join(pkg, "lib_test.go"): test,
	}
	for path, content := range files {
		//nolint:gosec // G306: test file
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
