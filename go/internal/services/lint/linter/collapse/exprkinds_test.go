package collapse

import (
	"testing"
)

func TestCollapse_CompositeLit(t *testing.T) {
	src := `package foo

func f() {
	x := []int{
		1,
		2,
		3,
	}
	_ = x
}
`
	want := `package foo

func f() {
	x := []int{1, 2, 3}
	_ = x
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_StructLit(t *testing.T) {
	src := `package foo

type S struct {
	A int
	B string
}

func f() {
	s := S{
		A: 1,
		B: "hello",
	}
	_ = s
}
`
	want := `package foo

type S struct {
	A int
	B string
}

func f() {
	s := S{A: 1, B: "hello"}
	_ = s
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_VariadicCall(t *testing.T) {
	src := `package foo

func f() {
	args := []string{"a", "b"}
	foo(
		arg1,
		args...,
	)
	_ = args
}
`
	want := `package foo

func f() {
	args := []string{"a", "b"}
	foo(arg1, args...)
	_ = args
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_MapLit(t *testing.T) {
	src := `package foo

func f() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	_ = m
}
`
	want := `package foo

func f() {
	m := map[string]int{"a": 1, "b": 2}
	_ = m
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_TypelessCompositeLit(t *testing.T) {
	src := `package foo

func f() {
	x := [][]int{
		{
			4,
			5,
		},
	}
	_ = x
}
`
	want := `package foo

func f() {
	x := [][]int{
		{4, 5},
	}
	_ = x
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_VariadicSingleArg(t *testing.T) {
	src := `package foo

func f() {
	args := []string{"a"}
	foo(
		args...,
	)
	_ = args
}
`
	want := `package foo

func f() {
	args := []string{"a"}
	foo(args...)
	_ = args
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_RawStringInCompositeLit(t *testing.T) {
	src := "package foo\n\nfunc f() {\n\tx := []string{\n\t\t`multi\nline`,\n\t}\n\t_ = x\n}\n"
	assertFixResult(t, src, src)
}

func TestCollapse_RegularStringInArgs(t *testing.T) {
	src := `package foo

func f() {
	foo(
		"hello",
		"world",
	)
}
`
	want := `package foo

func f() {
	foo("hello", "world")
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_BacktickStringWithoutNewlines(t *testing.T) {
	src := "package foo\n\nfunc f() {\n\tfoo(\n\t\t`hello`,\n\t\t`world`,\n\t)\n}\n"
	want := "package foo\n\nfunc f() {\n\tfoo(`hello`, `world`)\n}\n"
	assertFixResult(t, src, want)
}
