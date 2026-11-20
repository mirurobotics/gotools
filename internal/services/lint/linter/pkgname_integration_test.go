package linter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_Check_PkgnameViolation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad_name.go", `package my_pkg

func f() {}
`)
	diags, fixed, err := Run(dir, false, defaultCfg())
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic, got %d", diags)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed, got %d", fixed)
	}
}

func TestRun_Fix_PkgnameViolationOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad_name.go", `package my_pkg

func f() {}
`)
	diags, fixed, err := Run(dir, true, defaultCfg())
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic (pkgname), got %d", diags)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed (no auto-fixer), got %d", fixed)
	}
}

func TestRun_Fix_PkgnameViolationWithImportFix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad_both.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() { _ = strings.ToUpper(os.Getenv("X")) }
`)
	diags, fixed, err := Run(dir, true, defaultCfg())
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic (pkgname), got %d", diags)
	}
	if fixed != 1 {
		t.Errorf("expected 1 fixed (imports), got %d", fixed)
	}
}

func TestRun_Check_PkgnameAndImportViolations(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() { _ = strings.ToUpper(os.Getenv("X")) }
`)
	diags, fixed, err := Run(dir, false, defaultCfg())
	if err != nil {
		t.Fatal(err)
	}
	if diags != 2 {
		t.Errorf("expected 2 diagnostics (pkgname + imports), got %d", diags)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed, got %d", fixed)
	}
}

func TestRun_Check_AllowedPkgnamePrefixes(t *testing.T) {
	tests := []struct {
		name    string
		pkgName string
	}{
		{name: "test_ prefix", pkgName: "test_helpers"},
		{name: "mock_ prefix", pkgName: "mock_service"},
		{name: "_test suffix", pkgName: "foo_test"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir+"/ok.go", "package "+tc.pkgName+"\n\nfunc f() {}\n")
			diags, _, err := Run(dir, false, defaultCfg())
			if err != nil {
				t.Fatal(err)
			}
			if diags != 0 {
				t.Errorf(
					"expected 0 diagnostics for package %q, got %d",
					tc.pkgName, diags,
				)
			}
		})
	}
}

func TestRun_Fix_PkgnameWithFuncinline(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package my_pkg

func Code() int {
	return 42
}

func Hello() string {
	return "hi"
}
`)
	diags, fixed, err := Run(dir, true, defaultCfg())
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic (pkgname), got %d", diags)
	}
	if fixed != 1 {
		t.Errorf("expected 1 fixed (funcinline), got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "bad.go"))
	want := `package my_pkg

func Code() int { return 42 }

func Hello() string { return "hi" }
`
	if string(content) != want {
		t.Errorf("fix produced unexpected result.\ngot:\n%s\nwant:\n%s", content, want)
	}
}

func TestRun_Fix_AllFixersWithPkgname(t *testing.T) {
	diags, fixed, got := runFix(t, defaultCfg(), collapsibleSrc("my_pkg"))
	assertFixed(t, diags, fixed, got,
		fixResult{diags: 1, fixed: 1, src: collapsedSrc("my_pkg")})
}
