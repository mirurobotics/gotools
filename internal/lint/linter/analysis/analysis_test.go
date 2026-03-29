package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// --- Diagnostic ---

func TestDiagnostic_String(t *testing.T) {
	tests := []struct {
		name string
		diag Diagnostic
		want string
	}{
		{
			"basic",
			Diagnostic{File: "foo.go", Line: 42, Message: "bad"},
			"foo.go:42: bad",
		},
		{"line 1", Diagnostic{File: "a.go", Line: 1, Message: "m"}, "a.go:1: m"},
		{"empty message", Diagnostic{File: "x.go", Line: 0, Message: ""}, "x.go:0: "},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.diag.String()
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- LineStartOffset ---

func TestLineStartOffset(t *testing.T) {
	src := []byte("first\nsecond\nthird\n")
	//                    0123456789...
	// "first\n"  = bytes 0-5,  line start 0
	// "second\n" = bytes 6-12, line start 6
	// "third\n"  = bytes 13-18, line start 13

	tests := []struct {
		name string
		off  int
		want int
	}{
		{"start of file", 0, 0},
		{"mid first line", 3, 0},
		{"right before first newline", 5, 0},
		{"start of second line", 6, 6},
		{"mid second line", 9, 6},
		{"start of third line", 13, 13},
		{"mid third line", 16, 13},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := LineStartOffset(src, tc.off)
			if got != tc.want {
				t.Errorf("LineStartOffset(src, %d) = %d, want %d", tc.off, got, tc.want)
			}
		})
	}
}

func TestLineStartOffset_SingleLine(t *testing.T) {
	src := []byte("no newlines here")
	for _, off := range []int{0, 5, 15} {
		got := LineStartOffset(src, off)
		if got != 0 {
			t.Errorf("LineStartOffset(single-line, %d) = %d, want 0", off, got)
		}
	}
}

func TestLineStartOffset_EmptyLines(t *testing.T) {
	src := []byte("\n\nthird")
	// byte 0 = '\n', byte 1 = '\n', bytes 2-6 = "third"
	tests := []struct {
		off  int
		want int
	}{
		{0, 0}, // first empty line
		{1, 1}, // second empty line
		{2, 2}, // start of "third"
		{5, 2}, // mid "third"
	}
	for _, tc := range tests {
		got := LineStartOffset(src, tc.off)
		if got != tc.want {
			t.Errorf(
				"LineStartOffset(empty-lines, %d) = %d, want %d",
				tc.off, got, tc.want,
			)
		}
	}
}

// --- LineEndOffset ---

func TestLineEndOffset(t *testing.T) {
	src := []byte("first\nsecond\nthird\n")

	tests := []struct {
		name string
		off  int
		want int
	}{
		{"start of file", 0, 5},
		{"mid first line", 3, 5},
		{"start of second line", 6, 12},
		{"mid second line", 9, 12},
		{"start of third line", 13, 18},
		{"mid third line", 16, 18},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := LineEndOffset(src, tc.off)
			if got != tc.want {
				t.Errorf("LineEndOffset(src, %d) = %d, want %d", tc.off, got, tc.want)
			}
		})
	}
}

func TestLineEndOffset_NoTrailingNewline(t *testing.T) {
	src := []byte("first\nsecond")
	got := LineEndOffset(src, 6)
	if got != len(src) {
		t.Errorf("LineEndOffset = %d, want %d (EOF)", got, len(src))
	}
}

func TestLineEndOffset_AtNewline(t *testing.T) {
	src := []byte("abc\ndef\n")
	got := LineEndOffset(src, 3)
	if got != 3 {
		t.Errorf("LineEndOffset at newline = %d, want 3", got)
	}
}

// --- CollapsedLineWidth ---

func TestCollapsedLineWidth(t *testing.T) {
	// Source: "\tfoo([\n\t\t1,\n\t\t2,\n\t])\n"
	// startOff=5 at '[', ']' is at 18, endOff=19 (one past ']')
	// indent: "\tfoo(" → 4+4 = 8
	// collapsed: "[1, 2]" = 6
	// trailing: ")" = 1
	// total = 8 + 6 + 1 = 15
	src := []byte("\tfoo([\n\t\t1,\n\t\t2,\n\t])\n")
	got := CollapsedLineWidth(src, 5, 19, 4, "[1, 2]")
	if got != 15 {
		t.Errorf("CollapsedLineWidth = %d, want 15", got)
	}
}

func TestCollapsedLineWidth_NoTrailing(t *testing.T) {
	// Source: "  foo(\n    arg,\n  )\n"
	// startOff=2 at 'f', ')' is at 18, endOff=19 (one past ')')
	// indent: "  " = 2
	// collapsed: "foo(arg)" = 8
	// trailing after ')' to '\n' = "" = 0
	// total = 2 + 8 + 0 = 10
	src := []byte("  foo(\n    arg,\n  )\n")
	got := CollapsedLineWidth(src, 2, 19, 4, "foo(arg)")
	if got != 10 {
		t.Errorf("CollapsedLineWidth = %d, want 10", got)
	}
}

// --- VisualWidth ---

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		tabWidth int
		want     int
	}{
		{"empty", "", 4, 0},
		{"spaces only", "    ", 4, 4},
		{"single tab", "\t", 4, 4},
		{"tab width 8", "\t", 8, 8},
		{"tab after 1 char", "a\t", 4, 4},
		{"tab after 2 chars", "ab\t", 4, 4},
		{"tab after 3 chars", "abc\t", 4, 4},
		{"tab after 4 chars (aligned)", "abcd\t", 4, 8},
		{"two tabs", "\t\t", 4, 8},
		{"mixed tab space tab", "\t  \t", 4, 8},
		{"tab mid word", "a\tb", 4, 5},
		{"no tabs", "hello", 4, 5},
		{"unicode", "café", 4, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VisualWidth(tc.input, tc.tabWidth)
			if got != tc.want {
				t.Errorf(
					"VisualWidth(%q, %d) = %d, want %d",
					tc.input, tc.tabWidth, got, tc.want,
				)
			}
		})
	}
}

// --- HasCommentsBetween ---

func TestHasCommentsBetween_WithComment(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		// a comment
		arg2,
	)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	// Find the call expression's Lparen and Rparen.
	call := findFirstCallExpr(f)
	if call == nil {
		t.Fatal("no call expression found")
	}

	if !HasCommentsBetween(fset, f, call.Lparen, call.Rparen) {
		t.Error(
			"expected HasCommentsBetween to return true" +
				" for call with comment inside",
		)
	}
}

func TestHasCommentsBetween_WithoutComment(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		arg2,
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

	if HasCommentsBetween(fset, f, call.Lparen, call.Rparen) {
		t.Error("expected HasCommentsBetween to return false for call without comment")
	}
}

func TestHasCommentsBetween_CommentOutside(t *testing.T) {
	src := `package foo

// before the call
func f() {
	foo(
		arg1,
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

	if HasCommentsBetween(fset, f, call.Lparen, call.Rparen) {
		t.Error("expected false when comment is outside the range")
	}
}

func TestHasCommentsBetween_NolintDirective(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1, //nolint:errcheck
		arg2,
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

	if !HasCommentsBetween(fset, f, call.Lparen, call.Rparen) {
		t.Error("expected true for inline nolint comment inside args")
	}
}

// --- ContainsFuncLit ---

func TestContainsFuncLit_DirectFuncLit(t *testing.T) {
	src := `package foo

func f() {
	x := func() {}
	_ = x
}
`
	f := parseFile(t, src)

	// Find the assignment's RHS (the FuncLit).
	assign := findFirstAssign(f)
	if assign == nil {
		t.Fatal("no assignment found")
	}
	if !ContainsFuncLit(assign.Rhs[0]) {
		t.Error("expected ContainsFuncLit to return true for a FuncLit node")
	}
}

func TestContainsFuncLit_NestedFuncLit(t *testing.T) {
	src := `package foo

func f() {
	foo(func() {})
}
`
	f := parseFile(t, src)

	call := findFirstCallExpr(f)
	if call == nil {
		t.Fatal("no call expression found")
	}
	if !ContainsFuncLit(call) {
		t.Error(
			"expected ContainsFuncLit to return true" +
				" for call containing FuncLit arg",
		)
	}
}

func TestContainsFuncLit_NoFuncLit(t *testing.T) {
	src := `package foo

func f() {
	foo(1, 2, 3)
}
`
	f := parseFile(t, src)

	call := findFirstCallExpr(f)
	if call == nil {
		t.Fatal("no call expression found")
	}
	if ContainsFuncLit(call) {
		t.Error("expected ContainsFuncLit to return false when no FuncLit present")
	}
}

func TestContainsFuncLit_FuncLitInCompositeLit(t *testing.T) {
	src := `package foo

type S struct {
	Fn func()
}

func f() {
	s := S{Fn: func() {}}
	_ = s
}
`
	f := parseFile(t, src)

	assign := findFirstAssign(f)
	if assign == nil {
		t.Fatal("no assignment found")
	}
	if !ContainsFuncLit(assign.Rhs[0]) {
		t.Error(
			"expected ContainsFuncLit to return true" +
				" for composite lit containing FuncLit",
		)
	}
}

// --- helpers ---

func parseFile(t *testing.T, src string) *ast.File {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func findFirstCallExpr(f *ast.File) *ast.CallExpr {
	var call *ast.CallExpr
	ast.Inspect(f, func(n ast.Node) bool {
		if call != nil {
			return false
		}
		if c, ok := n.(*ast.CallExpr); ok {
			call = c
			return false
		}
		return true
	})
	return call
}

func findFirstAssign(f *ast.File) *ast.AssignStmt {
	var assign *ast.AssignStmt
	ast.Inspect(f, func(n ast.Node) bool {
		if assign != nil {
			return false
		}
		if a, ok := n.(*ast.AssignStmt); ok {
			assign = a
			return false
		}
		return true
	})
	return assign
}
