package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// MakePkgDir creates a temporary directory, changes into
// it, and creates a subdirectory at the given relative
// path. Returns the absolute path of the subdirectory.
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

// WriteCovgateFile writes a .covgate threshold file in
// the given directory.
func WriteCovgateFile(t *testing.T, dir, val string) {
	t.Helper()
	path := filepath.Join(dir, ".covgate")
	//nolint:gosec // G306: test file
	err := os.WriteFile(path, []byte(val), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}
