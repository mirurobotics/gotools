package collapse

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCollapse_SkipCommentInsideArgs(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		// important comment
		arg2,
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_SkipNolintDirective(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1, //nolint:errcheck
		arg2,
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_SkipFuncLitArg(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		func() { return },
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_SkipNestedMultiLineArg(t *testing.T) {
	src := `package foo

func f() {
	foo(
		bar(
			baz,
		),
		arg2,
	)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, file, []byte(src), 88, 4)

	for _, cand := range candidates {
		pos := fset.Position(cand.Pos)
		if pos.Line == 4 {
			t.Error(
				"foo() should not be collapsed because" +
					" bar() arg spans multiple lines",
			)
		}
	}
}

func TestCollapse_SkipRawStringWithNewlines(t *testing.T) {
	src := "package foo\n\nfunc f() {\n\tfoo(\n\t\t`multi\nline`,\n\t\targ2,\n\t)\n}\n"
	assertFixResult(t, src, src)
}

func TestCollapse_EmptyArgs_NoCollapse(t *testing.T) {
	src := `package foo

func f() {
	foo()
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_EmptyArgs_MultiLine_NoCollapse(t *testing.T) {
	src := `package foo

func f() {
	foo(
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_EmptyCompLit_MultiLine_NoCollapse(t *testing.T) {
	src := `package foo

func f() {
	x := []int{
	}
	_ = x
}
`
	assertFixResult(t, src, src)
}
