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

func TestFix_SingleCandidate(t *testing.T) {
	src := `package foo

func f() {
	foo(
		1,
		2,
	)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	call := findFirstCallExpr(f)
	if call == nil {
		t.Fatal("no call expression found")
	}

	candidates := []Candidate{{
		Pos:       call.Fun.Pos(),
		End:       call.Rparen + 1,
		Collapsed: "foo(1, 2)",
	}}

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
	if len(calls) < 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	candidates := []Candidate{
		{
			Pos:       calls[0].Fun.Pos(),
			End:       calls[0].Rparen + 1,
			Collapsed: "a(1)",
		},
		{
			Pos:       calls[1].Fun.Pos(),
			End:       calls[1].Rparen + 1,
			Collapsed: "b(2)",
		},
	}

	got := string(Fix(fset, []byte(src), candidates))
	if !strings.Contains(got, "a(1)") {
		t.Errorf("Fix did not apply first candidate:\n%s", got)
	}
	if !strings.Contains(got, "b(2)") {
		t.Errorf("Fix did not apply second candidate:\n%s", got)
	}
}

func TestFix_OverlappingCandidates(t *testing.T) {
	// Nested call: outer(inner(1, 2)) where both are
	// multi-line. The inner candidate (higher offset) should
	// be applied; the outer should be dropped to avoid
	// source corruption.
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
	if len(calls) < 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	// Identify outer and inner by position.
	outer, inner := calls[0], calls[1]
	if fset.Position(outer.Pos()).Offset > fset.Position(inner.Pos()).Offset {
		outer, inner = inner, outer
	}

	candidates := []Candidate{
		{
			Pos:       outer.Fun.Pos(),
			End:       outer.Rparen + 1,
			Collapsed: "outer(inner(1, 2))",
		},
		{
			Pos:       inner.Fun.Pos(),
			End:       inner.Rparen + 1,
			Collapsed: "inner(1, 2)",
		},
	}

	got := string(Fix(fset, []byte(src), candidates))

	// Inner should be applied (it's higher offset, processed
	// first). Outer should be dropped (it overlaps inner).
	if !strings.Contains(got, "inner(1, 2)") {
		t.Errorf("expected inner candidate to be applied:\n%s", got)
	}
	// The outer call should still be multi-line since its
	// candidate was dropped.
	if strings.Contains(got, "outer(inner(1, 2))") {
		t.Errorf("outer candidate should have been dropped:\n%s", got)
	}
}

func TestCheckCandidates_Empty(t *testing.T) {
	fset := token.NewFileSet()
	diags := CheckCandidates(fset, "f.go", nil, "msg")
	if len(diags) != 0 {
		t.Errorf("expected 0 diags, got %d", len(diags))
	}
}

func TestCheckCandidates_WithCandidates(t *testing.T) {
	src := `package foo

func f() {
	foo(
		1,
	)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	call := findFirstCallExpr(f)
	if call == nil {
		t.Fatal("no call found")
	}

	candidates := []Candidate{{
		Pos:       call.Fun.Pos(),
		End:       call.Rparen + 1,
		Collapsed: "foo(1)",
	}}

	msg := "test message"
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
