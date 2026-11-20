package funcinline

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestFuncInline_FuncLitInArgument(t *testing.T) {
	src := `package foo

func f() []string {
	return convert(
		items,
		func(x Item) string {
			return string(x)
		},
	)
}
`
	want := `package foo

func f() []string {
	return convert(
		items,
		func(x Item) string { return string(x) },
	)
}
`
	assertFixResult(t, src, want)
}

func TestFuncInline_FuncLitAlreadySingleLine(t *testing.T) {
	src := `package foo

func f() {
	apply(func(x int) int { return x + 1 })
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitMultipleStatements(t *testing.T) {
	src := `package foo

func f() {
	apply(func(x int) int {
		y := x + 1
		return y
	})
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitWithComment(t *testing.T) {
	src := `package foo

func f() {
	apply(func(x int) int {
		// important
		return x
	})
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitNestedFuncLit(t *testing.T) {
	src := `package foo

func f() {
	apply(func() func() {
		return func() {}
	})
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitAssignedToVar(t *testing.T) {
	src := `package foo

var fn = func(x int) int {
	return x * 2
}
`
	want := `package foo

var fn = func(x int) int { return x * 2 }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_BothDeclAndLit(t *testing.T) {
	src := `package foo

func simple() int {
	return 1
}

func wrapper() {
	apply(func(x int) int {
		return x
	})
}
`
	want := `package foo

func simple() int { return 1 }

func wrapper() {
	apply(func(x int) int { return x })
}
`
	assertFixResult(t, src, want)
}

func TestFuncInline_FuncLitTrailingContentExceedsLimit(t *testing.T) {
	src := `package foo

func f() {
	call(func(x int) int {
		return x
	}, other)
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, file, srcBytes, 44, 4)
	if len(candidates) != 0 {
		t.Fatalf(
			"expected 0 candidates (trailing content exceeds limit), got %d",
			len(candidates),
		)
	}
}

func TestFuncInline_FuncLitTrailingContentWithinLimit(t *testing.T) {
	src := `package foo

func f() {
	call(func(x int) int {
		return x
	}, other)
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, file, srcBytes, 45, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate at exact limit, got %d", len(candidates))
	}
}
