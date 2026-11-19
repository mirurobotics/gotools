package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestFix_Empty(t *testing.T) {
	fset := token.NewFileSet()
	src := []byte("hello world")
	got := Fix(fset, src, nil)
	if string(got) != string(src) {
		t.Errorf("expected unchanged src, got %q", got)
	}
}

func TestCheckCandidates_Empty(t *testing.T) {
	fset := token.NewFileSet()
	diags := CheckCandidates(fset, "f.go", nil, "msg")
	if len(diags) != 0 {
		t.Errorf("expected 0 diags, got %d", len(diags))
	}
}

func parseCalls(t *testing.T, src string) (*token.FileSet, []*ast.CallExpr) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var calls []*ast.CallExpr
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			calls = append(calls, c)
		}
		return true
	})
	return fset, calls
}

func callCandidate(call *ast.CallExpr, collapsed string) Candidate {
	return Candidate{Pos: call.Fun.Pos(), End: call.Rparen + 1, Collapsed: collapsed}
}

func TestFix_SingleCandidate(t *testing.T) {
	src := `package foo

func f() {
	foo(
		1,
		2,
	)
}
`
	fset, calls := parseCalls(t, src)
	candidates := []Candidate{callCandidate(calls[0], "foo(1, 2)")}
	got := string(Fix(fset, []byte(src), candidates))
	want := `package foo

func f() {
	foo(1, 2)
}
`
	if got != want {
		t.Errorf("Fix produced:\n%s\nwant:\n%s", got, want)
	}
}

func TestFix_MultipleCandidates(t *testing.T) {
	src := `package foo

func f() {
	a(
		1,
	)
	b(
		2,
	)
}
`
	fset, calls := parseCalls(t, src)
	candidates := []Candidate{
		callCandidate(calls[0], "a(1)"),
		callCandidate(calls[1], "b(2)"),
	}
	got := string(Fix(fset, []byte(src), candidates))
	if !strings.Contains(got, "a(1)") {
		t.Errorf("Fix did not apply first candidate:\n%s", got)
	}
	if !strings.Contains(got, "b(2)") {
		t.Errorf("Fix did not apply second candidate:\n%s", got)
	}
}

// TestFix_OverlappingCandidates keeps the inner of two nested candidates.
func TestFix_OverlappingCandidates(t *testing.T) {
	src := `package foo

func f() {
	outer(
		inner(
			1,
			2,
		),
	)
}
`
	fset, calls := parseCalls(t, src)
	outer, inner := calls[0], calls[1]
	candidates := []Candidate{
		callCandidate(outer, "outer(inner(1, 2))"),
		callCandidate(inner, "inner(1, 2)"),
	}
	got := string(Fix(fset, []byte(src), candidates))
	if !strings.Contains(got, "inner(1, 2)") {
		t.Errorf("expected inner candidate to be applied:\n%s", got)
	}
	if strings.Contains(got, "outer(inner(1, 2))") {
		t.Errorf("outer candidate should have been dropped:\n%s", got)
	}
}

func TestCheckCandidates_WithCandidates(t *testing.T) {
	fset, calls := parseCalls(t, "package foo\n\nfunc f() {\n\tfoo(\n\t\t1,\n\t)\n}\n")
	msg := "test message"
	candidates := []Candidate{callCandidate(calls[0], "foo(1)")}
	diags := CheckCandidates(fset, "test.go", candidates, msg)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diag, got %d", len(diags))
	}
	if diags[0].Message != msg {
		t.Errorf("message = %q, want %q", diags[0].Message, msg)
	}
	if diags[0].File != "test.go" {
		t.Errorf("file = %q, want %q", diags[0].File, "test.go")
	}
}
