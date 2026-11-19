package coverage

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func assertTestArgs(t *testing.T, subPkg, testPrefix string, want ...string) {
	t.Helper()
	got := buildTestArgs(subPkg, "pkg", testPrefix, "cov.out")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildTestArgs() = %v, want %v", got, want)
	}
}

func TestBuildTestArgs_EmptySubPkg(t *testing.T) {
	assertTestArgs(t, "", "tests",
		"test", "-coverprofile=cov.out", "-coverpkg=./pkg/...",
		"./pkg/...", "./tests/...")
}

func TestBuildTestArgs_SubPkgWithoutExtDir(t *testing.T) {
	assertTestArgs(t, "foo", t.TempDir(),
		"test", "-coverprofile=cov.out", "-coverpkg=./pkg/foo", "./pkg/foo")
}

func TestBuildTestArgs_SubPkgWithExtDir(t *testing.T) {
	dir := t.TempDir()
	//nolint:gosec // G301: test directory
	_ = os.MkdirAll(filepath.Join(dir, "bar"), 0o755)
	assertTestArgs(t, "bar", dir,
		"test", "-coverprofile=cov.out", "-coverpkg=./pkg/bar", "./pkg/bar",
		"./"+dir+"/bar")
}
