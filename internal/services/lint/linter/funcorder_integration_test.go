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
	fset, file, err := reparse("foo.go", src)
	if err != nil {
		t.Fatal(err)
	}
	input := checkInput{fset, "foo.go", file, src}
	got, gotFset, gotF, err := adoptOrderFix(input, []byte("func {"))
	kept := string(got) == unorderedSrc && gotFset == fset && gotF == file
	if err == nil || !kept {
		t.Errorf("adoptOrderFix = %q, %v; want original and error", got, err)
	}
	var buf strings.Builder
	if count := warnFixSkipped(&buf, "foo.go", err); count != 1 {
		t.Errorf("warnFixSkipped counted %d, want 1", count)
	}
	want := "warning: foo.go: funcorder fix produced invalid Go: "
	if !strings.HasPrefix(buf.String(), want) {
		t.Errorf("warning = %q", buf.String())
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
	want := `foo.go:3: "helper" should be below its last caller "Run"`
	if diags != 1 || !strings.Contains(out.String(), want) {
		t.Errorf("got %d diagnostics, want 1 with %q:\n%s", diags, want, out.String())
	}
}
