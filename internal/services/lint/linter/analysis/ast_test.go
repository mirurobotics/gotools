package analysis

import (
	"go/parser"
	"go/token"
	"testing"
)

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
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	call := findFirstCallExpr(file)
	if call == nil {
		t.Fatal("no call expression found")
	}

	if !HasCommentsBetween(fset, file, call.Lparen, call.Rparen) {
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
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	call := findFirstCallExpr(file)
	if call == nil {
		t.Fatal("no call expression found")
	}

	if HasCommentsBetween(fset, file, call.Lparen, call.Rparen) {
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
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	call := findFirstCallExpr(file)
	if call == nil {
		t.Fatal("no call expression found")
	}

	if HasCommentsBetween(fset, file, call.Lparen, call.Rparen) {
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
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	call := findFirstCallExpr(file)
	if call == nil {
		t.Fatal("no call expression found")
	}

	if !HasCommentsBetween(fset, file, call.Lparen, call.Rparen) {
		t.Error("expected true for inline nolint comment inside args")
	}
}

func TestContainsFuncLit_DirectFuncLit(t *testing.T) {
	src := `package foo

func f() {
	x := func() {}
	_ = x
}
`
	file := parseFile(t, src)

	assign := findFirstAssign(file)
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
	file := parseFile(t, src)

	call := findFirstCallExpr(file)
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
	file := parseFile(t, src)

	call := findFirstCallExpr(file)
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
	file := parseFile(t, src)

	assign := findFirstAssign(file)
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
