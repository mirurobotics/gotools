package linter

import (
	"strings"
	"testing"
)

func runDocCheck(t *testing.T, dir string, docLines, docFileLines int) (int, string) {
	t.Helper()
	var out strings.Builder
	cfg := defaultCfg()
	cfg.MaxDocLines = docLines
	cfg.MaxDocFileLines = docFileLines
	cfg.Out = &out
	diags, _, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return diags, out.String()
}

func TestRun_Check_DocLimits(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/doc.go", `// Package foo does things.
// At some length.
package foo
`)
	writeFile(t, dir+"/foo.go", `package foo

// F does a thing.
// At some length.
func F() {}
`)
	diags, out := runDocCheck(t, dir, 1, 20)
	if diags != 1 || !strings.Contains(out, "foo.go:3: doc comment is 2 lines") {
		t.Errorf("got %d diagnostics, want only foo.go:3:\n%s", diags, out)
	}
}

func TestRun_Check_DocFileLimit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/doc.go", `// Package foo does things.
// At some length.
// And more.
package foo
`)
	diags, out := runDocCheck(t, dir, 1, 2)
	want := "doc.go:1: doc comment is 3 lines (limit 2)"
	if diags != 1 || !strings.Contains(out, want) {
		t.Errorf("got %d diagnostics, want doc.go:1:\n%s", diags, out)
	}
}

func assertCommentRule(t *testing.T, rule Rule, src, want string) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir+"/foo.go", src)
	for _, exclude := range []bool{false, true} {
		var out strings.Builder
		cfg := defaultCfg()
		cfg.Exclude = map[Rule]bool{rule: exclude}
		cfg.Out = &out
		diags, _, err := Run(dir, false, cfg)
		if err != nil {
			t.Fatal(err)
		}
		wantDiags := 1
		if exclude {
			wantDiags = 0
		}
		if diags != wantDiags || strings.Contains(out.String(), want) == exclude {
			t.Errorf("%s exclude=%v: %d diagnostics:\n%s", rule, exclude, diags, &out)
		}
	}
}

func TestRun_Check_FuncCommentRule(t *testing.T) {
	src := "package foo\n\nfunc F() {\n\t// note\n}\n"
	assertCommentRule(t, RuleFuncComment, src, "foo.go:4: comments")
}

func TestRun_Check_StrayCommentRule(t *testing.T) {
	src := "package foo\n\nfunc F() {}\n\n// note\n"
	assertCommentRule(t, RuleStrayComment, src, "foo.go:5: comment is not")
}

func TestRun_Check_PrivDocRule(t *testing.T) {
	src := "package foo\n\n// f does a thing.\nfunc f() {}\n"
	assertCommentRule(t, RulePrivDoc, src, "foo.go:3: comment on")
}

func TestRun_Check_DocRestateRule(t *testing.T) {
	src := "package foo\n\n// Run runs.\nfunc Run() {}\n"
	assertCommentRule(t, RuleDocRestate, src, "foo.go:3: doc comment for Run")
}
