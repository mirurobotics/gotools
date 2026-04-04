package coverage

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildTestArgs(t *testing.T) {
	dir := t.TempDir()
	extDir := filepath.Join(dir, "bar")
	//nolint:gosec // G301: test directory
	_ = os.MkdirAll(extDir, 0o755)

	tests := []struct {
		name       string
		subPkg     string
		srcPrefix  string
		testPrefix string
		coverFile  string
		want       []string
	}{
		{
			"empty subPkg",
			"", "pkg", "tests", "cov.out",
			[]string{
				"test",
				"-coverprofile=cov.out",
				"-coverpkg=./pkg/...",
				"./pkg/...",
				"./tests/...",
			},
		},
		{
			"subPkg without ext dir",
			"foo", "pkg", dir, "cov.out",
			[]string{
				"test", "-coverprofile=cov.out",
				"-coverpkg=./pkg/foo", "./pkg/foo",
			},
		},
		{
			"subPkg with ext dir",
			"bar", "pkg", dir, "cov.out",
			[]string{
				"test",
				"-coverprofile=cov.out",
				"-coverpkg=./pkg/bar",
				"./pkg/bar",
				"./" + dir + "/bar",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTestArgs(tt.subPkg, tt.srcPrefix, tt.testPrefix, tt.coverFile)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildTestArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
