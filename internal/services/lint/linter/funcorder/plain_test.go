package funcorder

import (
	"strings"
	"testing"
)

func TestCheck_PlainTypeAboveFirstUser(t *testing.T) {
	assertMessages(t, `package p

type group struct{}

func Check() { var c checker; c.run() }

type checker struct{}

func (c checker) run() { _ = group{} }
`,
		`type "group" should be directly above "checker", which holds its first user "checker.run"`,
	)
}

func TestCheck_PlainTypeAboveVarUser(t *testing.T) {
	assertMessages(t, `package p

func Run() { show(names) }

var names = []Name{}

type Name string

func show([]Name) {}
`,
		`"show" should be directly below "Run"`,
		`type "Name" should be directly above its first user "names"`,
	)
	assertMessages(t, `package p

type mode int

var defaults = cfg{mode: 1}

func setMode(m mode) {}
`,
		`type "mode" should be directly above its first user "setMode"`,
	)
	assertMessages(t, `package p

var defaults = cfg{m: mode(1)}

type mode int

func setMode(m mode) {}
`,
		`type "mode" should be directly above its first user "defaults"`,
	)
}

func TestCheck_PlainTypeAboveConstUser(t *testing.T) {
	assertMessages(t, `package p

func Run() { _ = A }

const A Method = "a"

type Method string
`,
		`type "Method" should be directly above its first user "A"`,
	)
}

func TestCheck_PlainTypeAboveItsConsts(t *testing.T) {
	assertMessages(t, `package p

type Kind int

func Run() { _ = x }

const x = Kind(1)

func f(k Kind) {}
`,
		`type "Kind" should be directly above its first user "f"`,
	)
}

func TestCheck_PlainTypeAboveHeadOfFirstUser(t *testing.T) {
	assertMessages(t, `package p

func Run() { f(0) }

const D = 2

const C T = 0

type O int

func f(O) {}

type T int
`,
		`type "T" should be directly above "D", which sits over its first user "C"`,
	)
}

func TestCheck_PlainTypeBelowTypeItHolds(t *testing.T) {
	assertMessages(t, `package p

func Run() { f(0) }

const C T = 0

type O int

type T int

func f(O) {}
`,
		`type "O" should be directly above its first user "f"`,
	)
}

func TestCheck_MethodHeadKeepsPlainTypeConst(t *testing.T) {
	assertMessages(t, `package p

type S struct{}

const c K = 0

var v = 1

func (S) M() {}

type K int

func f(K) {}
`)
}

func TestCheck_PlainTypeAboveBlockOverFirstUser(t *testing.T) {
	assertMessages(t, `package p

type K int

func Run() { f() }

const (
	c1 = 1
	c2 = 2
)

func f(K) {}
`,
		`type "K" should be directly above "c1", which sits over its first user "f"`,
	)
}

func TestCheck_PlainTypeAboveForeignMethods(t *testing.T) {
	assertMessages(t, `package p

func (c *checker) a() { c.b() }

type group struct{}

func (c *checker) b() { _ = group{} }
`,
		`method "checker.b" should be grouped with the other methods of "checker"`,
		`type "group" should be directly above the methods of "checker", `+
			`which hold its first user "checker.b"`,
	)
}

func TestCheck_PlainTypesPlaced(t *testing.T) {
	assertMessages(t, `package p

type a int

type b int

const B b = 1

func use(a, b) {}
`)
	assertMessages(t, `package p

type Unused struct{}

func Run() { _ = A }

type Kind int

const A Kind = 0

type a struct{}

type b struct{ x a }

func use(a, b) {}
`)
	assertMessages(t, `package p

func Run() { _ = A }

type Method string

const A Method = "a"

type call struct{ m Method }
`)
}

func TestCheck_PlainTypesLeaveSharedConsts(t *testing.T) {
	assertMessages(t, `package p

func Run() { _ = LevelDebug }

type Level int

type Name string

const (
	LevelDebug Level = iota
	nameDebug  Name  = "d"
)

func log(Level) {}
`)
}

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
