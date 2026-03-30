package gocover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetThreshold(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		exists  bool
		defVal  float64
		want    float64
	}{
		{"valid", "95.5\n", true, 80.0, 95.5},
		{"zero", "0\n", true, 80.0, 0.0},
		{"missing file", "", false, 80.0, 80.0},
		{"empty file", "", true, 80.0, 80.0},
		{"malformed", "abc\n", true, 80.0, 80.0},
		{"hundred", "100.0\n", true, 80.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgDir := filepath.Join(dir, tt.name)
			_ = os.MkdirAll(pkgDir, 0o755)

			if tt.exists {
				covFile := filepath.Join(pkgDir, ".covgate")
				_ = os.WriteFile(covFile, []byte(tt.content), 0o644)
			}

			got := GetThreshold(pkgDir, tt.defVal)
			if got != tt.want {
				t.Errorf("GetThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelPkg(t *testing.T) {
	tests := []struct {
		name   string
		pkg    string
		module string
		want   string
	}{
		{"normal", "github.com/foo/bar/pkg/errs", "github.com/foo/bar", "pkg/errs"},
		{"root", "github.com/foo/bar", "github.com/foo/bar", "."},
		{"no match", "github.com/other/pkg", "github.com/foo/bar", "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RelPkg(tt.pkg, tt.module)
			if got != tt.want {
				t.Errorf(
					"RelPkg(%q, %q) = %q, want %q",
					tt.pkg, tt.module, got, tt.want,
				)
			}
		})
	}
}

func TestBuildTestPaths(t *testing.T) {
	dir := t.TempDir()
	testDir := filepath.Join(dir, "tests", "errs")
	_ = os.MkdirAll(testDir, 0o755)

	tests := []struct {
		name      string
		pkg       string
		relPkg    string
		srcPrefix string
		tDir      string
		wantLen   int
	}{
		{"no test dir", "mod/pkg/errs", "pkg/errs", "pkg", "", 1},
		{"empty prefix", "mod/pkg/errs", "pkg/errs", "", dir + "/tests", 1},
		{"with test dir", "mod/pkg/errs", "pkg/errs", "pkg", dir + "/tests", 2},
		{"test dir missing", "mod/pkg/foo", "pkg/foo", "pkg", dir + "/tests", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildTestPaths(tt.pkg, tt.relPkg, tt.srcPrefix, tt.tDir)
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d: %v", len(got), tt.wantLen, got)
			}
		})
	}
}

func TestParseCoverageOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{
			"normal output",
			"pkg/foo/bar.go:10:\tFoo\t\t100.0%\n" +
				"pkg/foo/baz.go:20:\tBaz\t\t75.0%\n" +
				"total:\t\t\t(statements)\t85.5%\n",
			85.5,
		},
		{"zero coverage", "total:\t(statements)\t0.0%\n", 0.0},
		{"full coverage", "total:\t(statements)\t100.0%\n", 100.0},
		{"no total line", "pkg/foo.go:1:\tFoo\t80.0%\n", 0.0},
		{"empty input", "", 0.0},
		{"total too few fields", "total:\n", 0.0},
		{"malformed percentage", "total:\t(statements)\tabc%\n", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCoverageOutput(tt.input)
			if got != tt.want {
				t.Errorf("ParseCoverageOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNonEmptyLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single", "hello", 1},
		{"with blanks", "a\n\nb\n\nc", 3},
		{"only blanks", "\n\n\n", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NonEmptyLines(tt.input)
			if len(got) != tt.want {
				t.Errorf("len = %d, want %d", len(got), tt.want)
			}
		})
	}
}
