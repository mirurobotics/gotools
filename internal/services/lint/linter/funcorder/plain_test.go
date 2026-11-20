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
	assertMessages(t, `package p

func Run() { show(names) }

var names = []Name{}

type Name string

func show([]Name) {}
`,
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

func TestCheck_PlainTypesPlaced(t *testing.T) {
	assertMessages(t, `package p

type Unused struct{}

func Run() { _ = A }

type Kind int

const A Kind = 0

type a struct{}

type b struct{ x a }

func use(a, b) {}
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
