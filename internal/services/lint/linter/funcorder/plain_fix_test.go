package funcorder

import (
	"strings"
	"testing"
)

func TestFix_PlainTypeMovesToFirstUser(t *testing.T) {
	src := []string{
		"type group struct{}\n", "func Check() { var c checker; c.run() }\n",
		"type checker struct{}\n", "func (c checker) run() { _ = group{} }\n",
		"func Run() { _ = Name(\"\") }\n", "type Name string\n",
	}
	want := []string{src[1], src[0], src[2], src[3], src[5], src[4]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeCarriesItsConsts(t *testing.T) {
	src := []string{
		"func Run() Kind { return A }\n", "type Kind int\n",
		"const (\n\tA Kind = iota\n\tB\n)\n", "var x = 1\n",
	}
	want := []string{src[1], src[2], src[0], src[3]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeCarriesConstUser(t *testing.T) {
	src := []string{
		"type Kind int\n", "func Run() { _ = x }\n", "const x = Kind(1)\n",
		"func g() {}\n", "func f(k Kind) {}\n",
	}
	want := []string{src[1], src[3], src[0], src[2], src[4]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeMovesAboveItsConsts(t *testing.T) {
	src := []string{
		"func Run() { _ = A }\n", "const A Method = \"a\"\n",
		"type call struct{ m Method }\n", "type Method string\n",
	}
	want := []string{src[0], src[3], src[1], src[2]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypesUsedByEachOthersConsts(t *testing.T) {
	src := []string{
		"func helper() {}\n", "const Y b = 0\n", "type a int\n",
		"const X a = 0\n", "type b int\n", "func Run() { helper() }\n",
	}
	want := []string{src[4], src[1], src[2], src[3], src[5], src[0]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypesLeaveSharedConsts(t *testing.T) {
	src := []string{
		"type a int\n", "type b int\n", "const (\n\tA a = 0\n\tB b = 0\n)\n",
		"func First(b) {}\n", "func Last(a) {}\n",
	}
	want := []string{src[2], src[1], src[3], src[0], src[4]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeAboveConstOfOtherType(t *testing.T) {
	src := []string{
		"func Run() { f(0) }\n", "type T int\n", "const C = T(U(0))\n",
		"type U int\n", "func f(T) {}\n",
	}
	want := []string{src[0], src[1], src[4], src[3], src[2]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeAboveUserInHead(t *testing.T) {
	src := []string{
		"func Start() {}\n", "const X = a(b(0))\n", "const Y b = 0\n",
		"type a int\n", "type b int\n",
	}
	want := []string{src[0], src[4], src[2], src[3], src[1]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeAboveFirstPlacedUser(t *testing.T) {
	src := []string{
		"func Run() { f() }\n", "const c2 T = 0\n", "type N int\n",
		"const c1 T = 0\n", "const u N = 0\n", "type O int\n",
		"func f(O) {}\n", "type T int\n",
	}
	want := []string{src[0], src[7], src[1], src[2], src[3], src[4], src[5], src[6]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeAboveFirstEmittedUser(t *testing.T) {
	src := []string{
		"func Run() { f(0) }\n", "const cA1 B = 0\n", "const cA2 T = 0\n",
		"type A int\n", "const cB T = 0\n", "type B int\n", "func f(A) {}\n",
		"type T int\n",
	}
	want := []string{src[0], src[7], src[4], src[5], src[1], src[2], src[3], src[6]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeAboveHeadOfFirstUser(t *testing.T) {
	src := []string{
		"func Run() { f(0) }\n", "const c = 1\n", "const X a = 0\n",
		"type P int\n", "type a int\n", "func f(P) {}\n",
	}
	want := []string{src[0], src[4], src[1], src[2], src[3], src[5]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeLeavesUnrelatedConsts(t *testing.T) {
	src := []string{
		"type Kind int\n", "const A Kind = 0\n", "const prefix = \"p\"\n",
		"func Run() { _ = prefix }\n", "func f(Kind) {}\n",
	}
	want := []string{src[2], src[3], src[0], src[1], src[4]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeCarriesConstAbove(t *testing.T) {
	src := []string{"const C P = 0\n", "var a = 1\n", "func f(P) {}\n", "type P int\n"}
	want := []string{src[1], src[3], src[0], src[2]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeCarriesConstAboveFirstFunction(t *testing.T) {
	src := []string{
		"type c int\n", "type a int\n", "func F(a, b) {}\n",
		"const X = c(a(0))\n", "type b int\n",
	}
	want := []string{src[0], src[1], src[3], src[4], src[2]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeLeavesHeaderConst(t *testing.T) {
	src := []string{
		"const defaultName = \"x\"\n", "type Kind int\n",
		"func Run() { _ = defaultName; helper() }\n", "func helper() {}\n",
		"func use(Kind) {}\n",
	}
	want := []string{src[0], src[2], src[3], src[1], src[4]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlainTypeCarriesLeadingConst(t *testing.T) {
	src := []string{
		"func Run() { g(); f() }\n", "const c = 1\n", "type K int\n",
		"func f(K) {}\n", "func g() {}\n",
	}
	want := []string{src[0], src[4], src[1], src[2], src[3]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}
