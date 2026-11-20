package linter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateExclusions_Empty(t *testing.T) {
	err := ValidateExclusions(nil)
	if err != nil {
		t.Errorf("expected nil error for empty exclusions, got %v", err)
	}
}

func TestValidateExclusions_ValidRules(t *testing.T) {
	err := ValidateExclusions(map[Rule]bool{RuleLineLen: true, RulePkgName: true})
	if err != nil {
		t.Errorf("expected nil error for valid rules, got %v", err)
	}
}

func TestValidateExclusions_InvalidRule(t *testing.T) {
	err := ValidateExclusions(map[Rule]bool{RuleLineLen: true, Rule("typo"): true})
	if err == nil {
		t.Fatal("expected error for invalid rule")
	}
	if !strings.Contains(err.Error(), "typo") {
		t.Errorf("error should mention the bad rule name, got: %v", err)
	}
}

func TestRun_ExcludeFilter_SkipPkgNameAndImports(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() { _ = strings.ToUpper(os.Getenv("X")) }
`)
	cfg := cfgExcluding(RulePkgName, RuleImports)
	diags, _, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 0 {
		t.Errorf("expected 0 diagnostics with pkgname+imports excluded, got %d", diags)
	}
}

func TestRun_ExcludeFilter_SkipImports(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() { _ = strings.ToUpper(os.Getenv("X")) }
`)
	cfg := cfgExcluding(RuleImports)
	diags, _, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic (pkgname only), got %d", diags)
	}
}

func TestRun_ExcludeFilter_FixExcludePkgName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() { _ = strings.ToUpper(os.Getenv("X")) }
`)
	cfg := cfgExcluding(RulePkgName)
	diags, fixed, err := Run(dir, true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 0 {
		t.Errorf("expected 0 diagnostics (pkgname excluded), got %d", diags)
	}
	if fixed != 1 {
		t.Errorf("expected 1 fixed (imports), got %d", fixed)
	}

	//nolint:gosec // test helper reading test fixtures
	content, _ := os.ReadFile(dir + "/bad.go")
	if !strings.Contains(string(content), "package my_pkg") {
		t.Error("package name should not have changed")
	}
}

func TestRun_ExcludeFilter_SkipFileLen(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "big.go"),
		"package foo\n\nconst a = 1\nconst b = 2\nconst c = 3\n")

	cfg := defaultCfg()
	cfg.MaxFileLen = 3
	cfg.Exclude = map[Rule]bool{RuleFileLen: true}
	if err := ValidateExclusions(cfg.Exclude); err != nil {
		t.Fatal(err)
	}
	diags, _, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 0 {
		t.Fatalf("expected 0 diagnostics with filelen excluded, got %d", diags)
	}
}
