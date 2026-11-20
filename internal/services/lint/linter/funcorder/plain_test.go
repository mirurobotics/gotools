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
		`type "group" should be directly above its first user "checker.run"`,
	)
	assertMessages(t, `package p

func Run() { show(names) }

var names = []Name{}

type Name string

func show([]Name) {}
`,
		`type "Name" should be directly above its first user "names"`,
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
