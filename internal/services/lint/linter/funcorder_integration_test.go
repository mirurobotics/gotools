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
	var errW strings.Builder
	in := checkInput{fset, "foo.go", f, src}
	got, _, gotF := adoptFix(in, []byte("func {"), &errW)
	if string(got) != unorderedSrc || gotF != f {
		t.Errorf("adoptFix replaced the source with unparseable output: %q", got)
	}
	if !strings.Contains(errW.String(), "warning: foo.go: fix skipped") {
		t.Errorf("missing warning, got %q", errW.String())
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
