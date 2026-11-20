package linter

import (
	"strings"
	"testing"
)

const unorderedSrc = `package foo

func helper() {}

// Run starts the helper.
func Run() { helper() }
`

func TestRun_Fix_FuncOrder(t *testing.T) {
	diags, fixed, got := runFix(t, defaultCfg(), unorderedSrc)
	assertFixed(t, diags, fixed, got, fixResult{diags: 0, fixed: 1, src: `package foo

// Run starts the helper.
func Run() { helper() }

func helper() {}
`})
}

func TestRun_Fix_FuncOrderExcluded(t *testing.T) {
	diags, fixed, got := runFix(t, cfgExcluding(RuleFuncOrder), unorderedSrc)
	assertFixed(t, diags, fixed, got, fixResult{diags: 0, fixed: 0, src: unorderedSrc})
}

func TestAdoptFix_KeepsSourceWhenFixDoesNotParse(t *testing.T) {
	src := []byte(unorderedSrc)
	fset, f, err := reparse("foo.go", src)
	if err != nil {
		t.Fatal(err)
	}
	in := checkInput{fset, "foo.go", f, src}
	got, gotFset, gotF, err := adoptOrderFix(in, []byte("func {"))
	kept := string(got) == unorderedSrc && gotFset == fset && gotF == f
	if err == nil || !kept {
		t.Errorf("adoptOrderFix = %q, %v; want original and error", got, err)
	}
	var w strings.Builder
	if n := warnFixSkipped(&w, "foo.go", err); n != 1 {
		t.Errorf("warnFixSkipped counted %d, want 1", n)
	}
	want := "warning: foo.go: funcorder fix produced invalid Go: "
	if !strings.HasPrefix(w.String(), want) {
		t.Errorf("warning = %q", w.String())
	}
}

func TestRun_Check_FuncOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/foo.go", unorderedSrc)
	var out strings.Builder
	cfg := defaultCfg()
	cfg.Out = &out
	diags, _, err := Run(dir, false, cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := `foo.go:3: "helper" should be below its first caller "Run"`
	if diags != 2 || !strings.Contains(out.String(), want) {
		t.Errorf("got %d diagnostics, want 2 with %q:\n%s", diags, want, out.String())
	}
}
