package gocover

import (
	"context"
	"os"
	"os/exec"
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

func TestParseCoverageOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			"normal output",
			"pkg/foo/bar.go:10:\tFoo\t\t100.0%\n" +
				"pkg/foo/baz.go:20:\tBaz\t\t75.0%\n" +
				"total:\t\t\t(statements)\t85.5%\n",
			85.5,
			false,
		},
		{"zero coverage", "total:\t(statements)\t0.0%\n", 0.0, false},
		{"full coverage", "total:\t(statements)\t100.0%\n", 100.0, false},
		{"no total line", "pkg/foo.go:1:\tFoo\t80.0%\n", 0.0, true},
		{"empty input", "", 0.0, true},
		{"total too few fields", "total:\n", 0.0, true},
		{"malformed percentage", "total:\t(statements)\tabc%\n", 0.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCoverageOutput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCoverageOutput() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseCoverageOutput() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseCoverageOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeGoProject(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)

	goMod := "module testmod\n\ngo 1.23\n"
	//nolint:gosec // G306: test file
	_ = os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644)
	pkg := filepath.Join(tmp, "mypkg")
	//nolint:gosec // G301: test directory
	_ = os.MkdirAll(pkg, 0o755)
	lib := "package mypkg\n\n" +
		"func Add(a, b int) int { return a + b }\n"
	//nolint:gosec // G306: test file
	_ = os.WriteFile(filepath.Join(pkg, "lib.go"), []byte(lib), 0o644)
	testSrc := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestAdd(t *testing.T) {\n" +
		"\tif Add(1,2) != 3 { t.Fatal() }\n}\n"
	//nolint:gosec // G306: test file
	_ = os.WriteFile(filepath.Join(pkg, "lib_test.go"), []byte(testSrc), 0o644)
}

func TestGoModule(t *testing.T) {
	makeGoProject(t)

	mod, err := GoModule()
	if err != nil {
		t.Fatalf("GoModule: %v", err)
	}
	if mod != "testmod" {
		t.Errorf("got %q, want %q", mod, "testmod")
	}
}

func TestGoListPackages(t *testing.T) {
	makeGoProject(t)

	pkgs, err := GoListPackages("testmod/...")
	if err != nil {
		t.Fatalf("GoListPackages: %v", err)
	}
	found := false
	for _, p := range pkgs {
		if p == "testmod/mypkg" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected testmod/mypkg in %v", pkgs)
	}
}

func TestExtractCoverage(t *testing.T) {
	makeGoProject(t)

	// Generate a real coverage profile.
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

func TestMeasure_TestFailure(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	goMod := "module testmod\n\ngo 1.23\n"
	//nolint:gosec // G306: test file
	_ = os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0o644)
	pkg := filepath.Join(tmp, "mypkg")
	//nolint:gosec // G301: test directory
	_ = os.MkdirAll(pkg, 0o755)
	//nolint:gosec // G306: test file
	_ = os.WriteFile(filepath.Join(pkg, "lib.go"), []byte("package mypkg\n"), 0o644)
	testSrc := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestFail(t *testing.T) " +
		"{ t.Fatal(\"fail\") }\n"
	//nolint:gosec // G306: test file
	_ = os.WriteFile(filepath.Join(pkg, "lib_test.go"), []byte(testSrc), 0o644)

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
