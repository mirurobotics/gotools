package linter

import (
	"os"
	"path/filepath"
	"testing"
)

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

	cfg := defaultCfg()
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

func NewScope(dbScope Scope) Scope { return NewScope(dbScope) }
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

	cfg := defaultCfg()
	diags, fixed, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed in check mode, got %d", fixed)
	}
	if diags < 1 {
		t.Errorf("expected at least 1 diagnostic, got %d", diags)
	}
}

func TestRun_Fix_LineLengthViolationWithImportFix(t *testing.T) {
	body := `func f() { _ = strings.TrimSpace(os.Getenv("` +
		`SOME_REALLY_LONG_ENV_VAR_NAME_THAT_STAYS_OVER_LIMIT` +
		`")) }
`
	diags, fixed, got := runFix(t, defaultCfg(),
		"package foo\n\nimport (\n\t\"strings\"\n\t\"os\"\n)\n\n"+body)
	assertFixed(t, diags, fixed, got, fixResult{
		diags: 1,
		fixed: 1,
		src:   "package foo\n\nimport (\n\t\"os\"\n\t\"strings\"\n)\n\n" + body,
	})
}

func TestIntegration_ImportsAndCollapse(t *testing.T) {
	diags, fixed, got := runFix(t, defaultCfg(), `package foo

import (
	"strings"
	"os"
)

func f() {
	_ = strings.ToUpper(
		os.Getenv("hello"),
	)
}
`)
	assertFixed(t, diags, fixed, got, fixResult{diags: 0, fixed: 1, src: `package foo

import (
	"os"
	"strings"
)

func f() { _ = strings.ToUpper(os.Getenv("hello")) }
`})
}

func TestIntegration_AllThreeRules(t *testing.T) {
	diags, fixed, got := runFix(t, defaultCfg(), collapsibleSrc("foo"))
	assertFixed(t, diags, fixed, got,
		fixResult{diags: 0, fixed: 1, src: collapsedSrc("foo")})
}

func TestIntegration_FuncsigOnly(t *testing.T) {
	diags, fixed, got := runFix(t, defaultCfg(), `package foo

func Process(
	dbScope Scope,
	name string,
) error {
	validate(name)
	return dbScope.Save(name)
}
`)
	assertFixed(t, diags, fixed, got, fixResult{diags: 0, fixed: 1, src: `package foo

func Process(dbScope Scope, name string) error {
	validate(name)
	return dbScope.Save(name)
}
`})
}

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
	diags, fixed, err := Run(dir, false, defaultCfg())
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

func TestRun_Fix_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/bad.go", `package foo

import (
	"strings"
	"os"
)

func f() {
	_ = strings.ToUpper(
		os.Getenv("hello"),
	)
}
`)
	_, fixed, err := Run(dir, true, defaultCfg())
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 1 {
		t.Fatalf("expected 1 fixed on first pass, got %d", fixed)
	}

	_, fixed, err = Run(dir, true, defaultCfg())
	if err != nil {
		t.Fatal(err)
	}
	if fixed != 0 {
		t.Errorf("expected 0 fixed on second pass (idempotent), got %d", fixed)
	}
}

func collapsibleSrc(pkg string) string {
	return "package " + pkg + `

import (
	"strings"
	"os"
)

func Code() int {
	return 42
}

func f() {
	_ = strings.ToUpper(
		os.Getenv("hello"),
	)
}
`
}

func collapsedSrc(pkg string) string {
	return "package " + pkg + `

import (
	"os"
	"strings"
)

func Code() int { return 42 }

func f() { _ = strings.ToUpper(os.Getenv("hello")) }
`
}
