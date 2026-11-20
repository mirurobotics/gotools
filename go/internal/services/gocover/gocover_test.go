package gocover

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mirurobotics/tools/go/internal/testutil"
)

func TestParallelism(t *testing.T) {
	if got := Parallelism(3); got != 3 {
		t.Errorf("Parallelism(3) = %d, want 3", got)
	}
	for _, n := range []int{0, -1} {
		if got := Parallelism(n); got != runtime.NumCPU() {
			t.Errorf("Parallelism(%d) = %d, want %d", n, got, runtime.NumCPU())
		}
	}
}

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
			//nolint:gosec // G301: test directory
			_ = os.MkdirAll(pkgDir, 0o755)

			if tt.exists {
				covFile := filepath.Join(pkgDir, ".covgate")
				//nolint:gosec // G306: test file
				_ = os.WriteFile(covFile, []byte(tt.content), 0o644)
			}

			got := GetThreshold(pkgDir, tt.defVal)
			if got != tt.want {
				t.Errorf("GetThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLookupThreshold_Present(t *testing.T) {
	dir := t.TempDir()
	covFile := filepath.Join(dir, ".covgate")
	//nolint:gosec // G306: test file
	if err := os.WriteFile(covFile, []byte("61.5\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	val, ok := LookupThreshold(dir)
	if !ok {
		t.Fatal("expected ok=true for present .covgate")
	}
	if val != 61.5 {
		t.Errorf("LookupThreshold() = %v, want 61.5", val)
	}
}

func TestLookupThreshold_Missing(t *testing.T) {
	dir := t.TempDir()

	val, ok := LookupThreshold(dir)
	if ok {
		t.Error("expected ok=false for missing .covgate")
	}
	if val != 0 {
		t.Errorf("LookupThreshold() = %v, want 0", val)
	}
}

func TestLookupThreshold_Malformed(t *testing.T) {
	dir := t.TempDir()
	covFile := filepath.Join(dir, ".covgate")
	//nolint:gosec // G306: test file
	if err := os.WriteFile(covFile, []byte("not-a-number\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	val, ok := LookupThreshold(dir)
	if ok {
		t.Error("expected ok=false for malformed .covgate")
	}
	if val != 0 {
		t.Errorf("LookupThreshold() = %v, want 0", val)
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
	//nolint:gosec // G301: test directory
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

func TestExtractCoverage(t *testing.T) {
	makeGoProject(t)

	tmp := t.TempDir()
	covFile := filepath.Join(tmp, "cover.out")
	//nolint:gosec // G204: test subprocess
	cmd := exec.CommandContext(
		context.Background(),
		"go", "test", "-coverprofile="+covFile,
		"-coverpkg=testmod/mypkg", "testmod/mypkg",
	)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test: %v\n%s", err, out)
	}

	cov, err := ExtractCoverage(covFile)
	if err != nil {
		t.Fatalf("ExtractCoverage: %v", err)
	}
	if cov < 100.0 {
		t.Errorf("expected 100%%, got %.1f%%", cov)
	}
}

func TestExtractCoverage_MissingFile(t *testing.T) {
	_, err := ExtractCoverage("/nonexistent/cover.out")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMeasure(t *testing.T) {
	makeGoProject(t)

	cov, _, err := Measure("testmod/mypkg", []string{"testmod/mypkg"})
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if cov < 100.0 {
		t.Errorf("expected 100%%, got %.1f%%", cov)
	}
}

func TestMeasureWithEnv_AppliesExtraEnv(t *testing.T) {
	makeGoProject(t)

	cov, _, err := MeasureWithEnv(
		"testmod/mypkg",
		[]string{"testmod/mypkg"},
		[]string{"GOMAXPROCS=1"},
	)
	if err != nil {
		t.Fatalf("MeasureWithEnv: %v", err)
	}
	if cov < 100.0 {
		t.Errorf("expected 100%%, got %.1f%%", cov)
	}
}

func TestMeasure_TestFailure(t *testing.T) {
	testSrc := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestFail(t *testing.T) " +
		"{ t.Fatal(\"fail\") }\n"
	testutil.WriteTestModule(t, "package mypkg\n", testSrc)
	_, output, err := Measure("testmod/mypkg", []string{"testmod/mypkg"})
	if err == nil {
		t.Fatal("expected error from failing test")
	}
	if len(output) == 0 {
		t.Error("expected non-empty output")
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

func TestParseCoverageOutput(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{
			"pkg/foo/bar.go:10:\tFoo\t\t100.0%\n" +
				"pkg/foo/baz.go:20:\tBaz\t\t75.0%\n" +
				"total:\t\t\t(statements)\t85.5%\n",
			85.5,
		},
		{"total:\t(statements)\t0.0%\n", 0.0},
		{"total:\t(statements)\t100.0%\n", 100.0},
	}
	for _, tt := range tests {
		got, err := ParseCoverageOutput(tt.input)
		if err != nil {
			t.Errorf("ParseCoverageOutput(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseCoverageOutput(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseCoverageOutput_Errors(t *testing.T) {
	inputs := map[string]string{
		"no total line":        "pkg/foo.go:1:\tFoo\t80.0%\n",
		"empty input":          "",
		"total too few fields": "total:\n",
		"malformed percentage": "total:\t(statements)\tabc%\n",
	}
	for name, input := range inputs {
		if _, err := ParseCoverageOutput(input); err == nil {
			t.Errorf("%s: ParseCoverageOutput() error = nil, want error", name)
		}
	}
}
