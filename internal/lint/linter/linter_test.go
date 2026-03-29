package linter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

var defaultCfg = Config{
	MaxLineWidth:  88,
	TabWidth:      4,
	MaxFuncLen:    0,
	MaxNestDepth:  0,
	MaxParamCount: 0,
	Exclude:       nil,
}

// --- Diagnostic tests ---

func TestDiagnostic_String(t *testing.T) {
	d := analysis.Diagnostic{File: "foo.go", Line: 42, Message: "bad imports"}
	want := "foo.go:42: bad imports"
	got := d.String()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- run / processFile tests ---

func TestRun_Check_Clean(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/clean.go", `package foo

import (
	"fmt"
	"strings"
)
`)
	diags, fixed, err := Run(dir, false, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 0 {
		t.Errorf("expected 0 diagnostics, got %d", diags)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed, got %d", fixed)
	}
}

func TestRun_Check_Violations(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package foo

import (
	"strings"
	"fmt"
)
`)
	diags, _, err := Run(dir, false, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic, got %d", diags)
	}
}

func TestRun_Fix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package foo

import (
	"strings"
	"fmt"
)
`)
	diags, fixed, err := Run(dir, true, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 0 {
		t.Errorf("expected 0 diagnostics, got %d", diags)
	}
	if fixed != 1 {
		t.Errorf("expected 1 fixed, got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "bad.go"))
	want := `package foo

import (
	"fmt"
	"strings"
)
`
	if string(content) != want {
		t.Errorf("file not fixed correctly.\ngot:\n%s\nwant:\n%s", content, want)
	}
}

func TestRun_Fix_AlreadyClean(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/clean.go", `package foo

import (
	"fmt"
	"strings"
)
`)
	_, fixed, err := Run(dir, true, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed, got %d", fixed)
	}
}

func TestRun_InvalidDir(t *testing.T) {
	_, _, err := Run("/nonexistent", false, defaultCfg)
	if err == nil {
		t.Error("expected error")
	}
}

func TestProcessFile_InvalidPath(t *testing.T) {
	_, _, err := ProcessFile("/nonexistent/file.go", false, defaultCfg)
	if err == nil {
		t.Error("expected error")
	}
}

// --- findGoFiles tests ---

func TestFindGoFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir+"/a.go", "package main")
	writeFile(t, dir+"/b.txt", "not go")
	writeFile(t, dir+"/sub/c.go", "package sub")
	writeFile(t, dir+"/.hidden/d.go", "package hidden")
	writeFile(t, dir+"/vendor/e.go", "package vendor")
	writeFile(t, dir+"/testdata/f.go", "package testdata")

	files, err := FindGoFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	goFiles := make(map[string]bool)
	for _, f := range files {
		goFiles[f] = true
	}

	if !goFiles[dir+"/a.go"] {
		t.Error("expected a.go")
	}
	if !goFiles[dir+"/sub/c.go"] {
		t.Error("expected sub/c.go")
	}
	if goFiles[dir+"/b.txt"] {
		t.Error("should not include b.txt")
	}
	if goFiles[dir+"/.hidden/d.go"] {
		t.Error("should not include .hidden/d.go")
	}
	if goFiles[dir+"/vendor/e.go"] {
		t.Error("should not include vendor/e.go")
	}
	if goFiles[dir+"/testdata/f.go"] {
		t.Error("should not include testdata/f.go")
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}
}

func TestFindGoFiles_NonExistentDir(t *testing.T) {
	_, err := FindGoFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

// --- checkOnly (pkgname) integration ---

func TestRun_Check_PkgnameViolation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad_name.go", `package my_pkg

func f() {}
`)
	diags, fixed, err := Run(dir, false, defaultCfg)
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
	diags, fixed, err := Run(dir, true, defaultCfg)
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
	diags, fixed, err := Run(dir, true, defaultCfg)
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

func TestRun_Fix_LineLengthViolationWithImportFix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad_both.go", `package foo

import (
	"strings"
	"os"
)

func f() { _ = strings.TrimSpace(os.Getenv("`+
		`SOME_REALLY_LONG_ENV_VAR_NAME_THAT_STAYS_OVER_LIMIT`+
		`")) }
`)
	diags, fixed, err := Run(dir, true, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic (linelen), got %d", diags)
	}
	if fixed != 1 {
		t.Errorf("expected 1 fixed (imports), got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "bad_both.go"))
	want := `package foo

import (
	"os"
	"strings"
)

func f() { _ = strings.TrimSpace(os.Getenv("` +
		`SOME_REALLY_LONG_ENV_VAR_NAME_THAT_STAYS_OVER_LIMIT` +
		`")) }
`
	if string(content) != want {
		t.Errorf("fix produced unexpected result.\ngot:\n%s\nwant:\n%s", content, want)
	}
}

// --- integration: imports + collapse ---

func TestIntegration_ImportsAndCollapse(t *testing.T) {
	dir := t.TempDir()
	src := `package foo

import (
	"strings"
	"fmt"
)

func f() {
	fmt.Println(
		strings.ToUpper("hello"),
	)
}
`
	writeFile(t, dir+"/test.go", src)

	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       nil,
	}
	_, fixed, err := Run(dir, true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 file fixed, got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "test.go"))
	// After collapse, f() has a single statement, so func-inline also fires.
	want := `package foo

import (
	"fmt"
	"strings"
)

func f() { fmt.Println(strings.ToUpper("hello")) }
`
	if string(content) != want {
		t.Errorf(
			"integration fix produced unexpected result.\ngot:\n%s\nwant:\n%s",
			content, want,
		)
	}
}

// --- integration: all three rules ---

func TestIntegration_AllThreeRules(t *testing.T) {
	dir := t.TempDir()
	src := `package foo

import (
	"strings"
	"fmt"
)

func f() {
	fmt.Println(
		strings.ToUpper("hello"),
	)
}

func Code() int {
	return 42
}
`
	writeFile(t, dir+"/test.go", src)

	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       nil,
	}
	_, fixed, err := Run(dir, true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 file fixed, got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "test.go"))
	want := `package foo

import (
	"fmt"
	"strings"
)

func f() { fmt.Println(strings.ToUpper("hello")) }

func Code() int { return 42 }
`
	if string(content) != want {
		t.Errorf(
			"integration fix produced unexpected result.\ngot:\n%s\nwant:\n%s",
			content, want,
		)
	}
}

// --- integration: check mode with multiple rule types ---

func TestRun_Check_PkgnameAndImportViolations(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() { _ = strings.ToUpper(os.Getenv("X")) }
`)
	diags, fixed, err := Run(dir, false, defaultCfg)
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

// --- integration: fix mode with all rules + pkgname ---

func TestRun_Fix_AllFixersWithPkgname(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() {
	_ = strings.ToUpper(
		os.Getenv("hello"),
	)
}

func Code() int {
	return 42
}
`)
	diags, fixed, err := Run(dir, true, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic (pkgname), got %d", diags)
	}
	if fixed != 1 {
		t.Errorf("expected 1 fixed, got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "bad.go"))
	want := `package my_pkg

import (
	"os"
	"strings"
)

func f() { _ = strings.ToUpper(os.Getenv("hello")) }

func Code() int { return 42 }
`
	if string(content) != want {
		t.Errorf("fix produced unexpected result.\ngot:\n%s\nwant:\n%s", content, want)
	}
}

// --- integration: allowed pkgname patterns ---

func TestRun_Check_AllowedPkgnamePrefixes(t *testing.T) {
	tests := []struct {
		name    string
		pkgName string
	}{
		{name: "test_ prefix", pkgName: "test_helpers"},
		{name: "mock_ prefix", pkgName: "mock_service"},
		{name: "_test suffix", pkgName: "foo_test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir+"/ok.go", "package "+tt.pkgName+"\n\nfunc f() {}\n")
			diags, _, err := Run(dir, false, defaultCfg)
			if err != nil {
				t.Fatal(err)
			}
			if diags != 0 {
				t.Errorf(
					"expected 0 diagnostics for package %q, got %d",
					tt.pkgName, diags,
				)
			}
		})
	}
}

// --- integration: fix mode with pkgname + funcinline only ---

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
	diags, fixed, err := Run(dir, true, defaultCfg)
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

// --- integration: multiple files with mixed violations ---

func TestRun_Check_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/clean.go", `package foo

func f() {}
`)
	writeFile(t, dir+"/bad_imports.go", `package foo

import (
	"strings"
	"os"
)

func g() { _ = strings.ToUpper(os.Getenv("hi")) }
`)
	diags, fixed, err := Run(dir, false, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 1 {
		t.Errorf("expected 1 diagnostic (imports only), got %d", diags)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed, got %d", fixed)
	}
}

// --- integration: fix is idempotent ---

func TestRun_Fix_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package foo

import (
	"strings"
	"fmt"
)

func f() {
	fmt.Println(
		strings.ToUpper("hello"),
	)
}
`)
	_, fixed, err := Run(dir, true, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 fixed on first pass, got %d", fixed)
	}

	// Run fix again — nothing should change.
	_, fixed, err = Run(dir, true, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed on second pass (idempotent), got %d", fixed)
	}
}

// --- error handling: Run skips unparseable files ---

func TestRun_SkipsUnparseableFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/good.go", `package foo

func f() {}
`)
	writeFile(t, dir+"/bad.go", `not valid go syntax at all!!!`)

	diags, fixed, err := Run(dir, false, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	// The good file has no violations; the bad file is skipped with a warning.
	if diags != 0 {
		t.Errorf("expected 0 diagnostics, got %d", diags)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed, got %d", fixed)
	}
}

func TestProcessFile_GeneratedFileSkipped(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/gen.go"
	writeFile(t, path, "// Code generated by foo; DO NOT EDIT.\n"+
		"package foo\n\nimport (\n\t\"strings\"\n\t\"os\"\n)\n\n"+
		"func f() { _ = strings.ToUpper(os.Getenv(\"X\")) }\n")
	diags, fixed, err := ProcessFile(path, false, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 0 {
		t.Errorf("expected 0 diagnostics for generated file, got %d", diags)
	}
	if fixed {
		t.Error("expected no fix for generated file")
	}
}

func TestProcessFile_UnparseableSyntax(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.go"
	writeFile(t, path, `not valid go`)
	_, _, err := ProcessFile(path, false, defaultCfg)
	if err == nil {
		t.Error("expected error for unparseable file")
	}
}

// --- integration: funcsig ---

func TestIntegration_FuncsigThenFuncinline(t *testing.T) {
	dir := t.TempDir()
	src := `package foo

func NewScope(
	dbScope Scope,
) Scope {
	return NewScope(dbScope)
}
`
	writeFile(t, dir+"/test.go", src)

	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       nil,
	}
	_, fixed, err := Run(dir, true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 file fixed, got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "test.go"))
	// funcsig collapses the signature, then funcinline collapses the body.
	want := `package foo

func NewScope(dbScope Scope) Scope { return NewScope(dbScope) }
`
	if string(content) != want {
		t.Errorf(
			"integration fix produced unexpected result.\ngot:\n%s\nwant:\n%s",
			content, want,
		)
	}
}

func TestIntegration_FuncsigOnly(t *testing.T) {
	dir := t.TempDir()
	src := `package foo

func Process(
	dbScope Scope,
	name string,
) error {
	validate(name)
	return dbScope.Save(name)
}
`
	writeFile(t, dir+"/test.go", src)

	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       nil,
	}
	_, fixed, err := Run(dir, true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 file fixed, got %d", fixed)
	}

	// #nosec G304 -- test reads a file created under t.TempDir.
	content, _ := os.ReadFile(filepath.Join(dir, "test.go"))
	// funcsig collapses the signature, but body has 2 statements so funcinline skips.
	want := `package foo

func Process(dbScope Scope, name string) error {
	validate(name)
	return dbScope.Save(name)
}
`
	if string(content) != want {
		t.Errorf(
			"integration fix produced unexpected result.\ngot:\n%s\nwant:\n%s",
			content, want,
		)
	}
}

func TestIntegration_FuncsigCheck(t *testing.T) {
	dir := t.TempDir()
	src := `package foo

func Get(
	key string,
) string {
	return key
}
`
	writeFile(t, dir+"/test.go", src)

	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       nil,
	}
	diags, fixed, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed in check mode, got %d", fixed)
	}
	// Should report at least 1 diagnostic for funcsig.
	if diags < 1 {
		t.Errorf("expected at least 1 diagnostic, got %d", diags)
	}
}

// --- ValidateExclusions tests ---

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

// --- exclude filtering tests ---

func TestRun_ExcludeFilter_SkipPkgNameAndImports(t *testing.T) {
	dir := t.TempDir()
	// This file has both a pkgname violation (my_pkg) and
	// a misordered import (would trigger imports rule).
	writeFile(t, dir+"/bad.go", `package my_pkg

import (
	"strings"
	"os"
)

func f() { _ = strings.ToUpper(os.Getenv("X")) }
`)
	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       map[Rule]bool{RulePkgName: true, RuleImports: true},
	}
	diags, _, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Both pkgname and imports are excluded, no other violations.
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
	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       map[Rule]bool{RuleImports: true},
	}
	diags, _, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Imports excluded, but pkgname should still fire.
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
	cfg := Config{
		MaxLineWidth:  88,
		TabWidth:      4,
		MaxFuncLen:    0,
		MaxNestDepth:  0,
		MaxParamCount: 0,
		Exclude:       map[Rule]bool{RulePkgName: true},
	}
	diags, fixed, err := Run(dir, true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Imports should be fixed, pkgname excluded so 0 diags.
	if diags != 0 {
		t.Errorf("expected 0 diagnostics (pkgname excluded), got %d", diags)
	}
	if fixed != 1 {
		t.Errorf("expected 1 fixed (imports), got %d", fixed)
	}

	//nolint:gosec // test helper reading test fixtures
	content, _ := os.ReadFile(dir + "/bad.go")
	// Package name should remain unchanged (pkgname excluded).
	if !strings.Contains(string(content), "package my_pkg") {
		t.Error("package name should not have changed")
	}
}

// --- self-lint regression ---

func TestRun_SelfLint(t *testing.T) {
	diags, fixed, err := Run("..", false, defaultCfg)
	if err != nil {
		t.Fatal(err)
	}
	if diags != 0 {
		t.Errorf("linter has %d self-violations", diags)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed, got %d", fixed)
	}
}

// --- helpers ---

func writeFile(t *testing.T, path, content string) {
	dir := path[:len(path)-len(path[strings.LastIndex(path, "/")+1:])]
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
