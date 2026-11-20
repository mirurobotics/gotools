package funcinline

import (
	"testing"
)

func TestFuncInline_SkipMultipleStatements(t *testing.T) {
	src := `package foo

func f() {
	x := 1
	_ = x
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipEmptyBody(t *testing.T) {
	src := `package foo

func f() {
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipBodyWithComment(t *testing.T) {
	src := `package foo

func f() int {
	// important
	return 42
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipMultiLineStatement(t *testing.T) {
	src := `package foo

type S struct{ A, B int }

func f() S {
	return S{
		A: 1,
		B: 2,
	}
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipFuncLitInBody(t *testing.T) {
	src := `package foo

func f() func() {
	return func() {}
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipNilBody(t *testing.T) {
	src := `package foo

func Asm()
`
	assertFixResult(t, src, src)
}
