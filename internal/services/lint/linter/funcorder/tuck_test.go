package funcorder

import (
	"strings"
	"testing"
)

func TestCheck_HeldTypeBelowItsUser(t *testing.T) {
	assertMessages(t, `package p

func TestA() { _ = []obj{} }

func TestB() {}

type obj struct{ id string }

func (o obj) ID() string { return o.id }
`,
		`"obj.ID" should be directly below "TestA"`,
	)
}

func TestCheck_HeldTypeBelowUsersHelpers(t *testing.T) {
	assertMessages(t, `package p

func Run() { helper() }

func helper() { _ = obj{} }

func Other() {}

type obj struct{}

func (o obj) ID() string { return "" }
`,
		`"obj.ID" should be directly below "helper"`,
	)
}

func TestCheck_HeldTypePastDeclarations(t *testing.T) {
	assertMessages(t, `package p

func Run() { _ = obj{} }

type Meta struct{}

func Other() {}

type obj struct{}

func (o obj) ID() string { return "" }
`,
		`"obj.ID" should be directly below "Run"`,
	)
}

func TestCheck_HeldTypesPlaced(t *testing.T) {
	cases := map[string]string{
		"above its user": `package p

type obj struct{}

func (o obj) ID() string { return "" }

func Run() { _ = obj{} }
`,
		"directly below its user": `package p

func Run() { _ = obj{} }

type obj struct{}

func (o obj) ID() string { return "" }

func Other() {}
`,
		"used only below": `package p

type obj struct{}

func (o obj) ID() string { return "" }

func Other() {}

func New() obj { return obj{} }
`,
		"used only by another type's methods": `package p

type other struct{}

func (o other) use() { _ = obj{} }

func Run() {}

type obj struct{}

func (o obj) ID() string { return "" }
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) { assertMessages(t, src) })
	}
}

func TestFix_HeldTypeMovesBelowItsUser(t *testing.T) {
	src := []string{
		"func TestA() { _ = []obj{} }\n", "func TestB() {}\n",
		"type obj struct{ id string }\n",
		"func (o obj) ID() string { return o.id }\n",
	}
	want := []string{src[0], src[2], src[3], src[1]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_HeldTypesKeepTheirOrder(t *testing.T) {
	src := []string{
		"func Run() { _ = b{}; _ = a{} }\n", "func Other() {}\n",
		"type a struct{}\n", "func (x a) A() {}\n",
		"type b struct{}\n", "func (x b) B() {}\n",
	}
	want := []string{src[0], src[2], src[3], src[4], src[5], src[1]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}
