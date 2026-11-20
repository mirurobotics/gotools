package funcorder

import (
	"strings"
	"testing"
)

func TestCheck_TuckedTypeBelowItsUser(t *testing.T) {
	assertMessages(t, `package p

func TestA() { _ = []obj{} }

func TestB() {}

type obj struct{ id string }

func (o obj) ID() string { return o.id }
`,
		`"obj.ID" should be directly below "TestA"`,
	)
}

func TestCheck_TuckedTypeBelowUsersHelpers(t *testing.T) {
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

func TestCheck_TuckedTypePastDeclarations(t *testing.T) {
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

func TestCheck_TuckedTypeBelowVarBlockUser(t *testing.T) {
	assertMessages(t, `package p

func Other() {}

var cache map[string]obj

func Run() { _ = cache }

func Last() {}

type obj struct{}

func (o obj) ID() string { return "" }
`,
		`"obj.ID" should be directly below "Run"`,
	)
}

func TestCheck_TuckedTypesPlaced(t *testing.T) {
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
		"used only by a var block above it": `package p

import "io"

var _ io.Reader = (*reader)(nil)

type reader struct{}

func (r *reader) Read(p []byte) (int, error) { return 0, nil }
`,
		"head holds an initialized var": `package p

func Run() { _ = obj{} }

var a = f("a")

func Other() {}

type obj struct{}

var b = f("b")

func (obj) ID() {}

func f(s string) string { return s }
`,
		"helper called by code it would jump": `package p

func Run() { _ = obj{} }

func Other() { g() }

type obj struct{}

func (o obj) ID() { g() }

func g() {}
`,
		"used only by a var block in another type's head": `package p

type other struct{}

var x obj

func (o other) M() {}

func Run() {}

type obj struct{}

func (o obj) ID() string { return "" }
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

func TestFix_TuckedTypeMovesBelowItsUser(t *testing.T) {
	src := []string{
		"func TestA() { _ = []obj{} }\n", "func TestB() {}\n",
		"type obj struct{ id string }\n",
		"func (o obj) ID() string { return o.id }\n",
	}
	want := []string{src[0], src[2], src[3], src[1]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_TuckedTypesKeepTheirOrder(t *testing.T) {
	src := []string{
		"func Run() { _ = b{}; _ = a{} }\n", "func Other() {}\n",
		"type a struct{}\n", "func (x a) A() {}\n",
		"type b struct{}\n", "func (x b) B() {}\n",
	}
	want := []string{src[0], src[2], src[3], src[4], src[5], src[1]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_TuckedTypeBelowLastUser(t *testing.T) {
	src := []string{
		"func A() { _ = obj{} }\n", "func X() {}\n", "func B() { _ = obj{} }\n",
		"func C() {}\n",
		"type obj struct{}\n", "func (o obj) ID() string { return \"\" }\n",
	}
	want := []string{src[0], src[1], src[2], src[4], src[5], src[3]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_TuckedTypePastDeclarations(t *testing.T) {
	src := []string{
		"func Run() { _ = obj{} }\n", "type Meta struct{}\n", "func Other() {}\n",
		"type obj struct{}\n", "func (o obj) ID() string { return \"\" }\n",
	}
	want := []string{src[0], src[3], src[4], src[1], src[2]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_TuckedTypeUnblocksAnother(t *testing.T) {
	src := []string{
		"func Run() { _ = a{} }\n", "func Other() {}\n",
		"type b struct{}\n", "func (x b) B() {}\n",
		"type a struct{}\n", "func (x a) A() { h() }\n", "func h() { _ = b{} }\n",
	}
	want := []string{src[0], src[4], src[5], src[6], src[2], src[3], src[1]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_TuckedTypeTakesItsHelpers(t *testing.T) {
	src := []string{
		"func Run() { _ = obj{} }\n", "func Other() {}\n",
		"type obj struct{}\n", "func (o obj) ID() { g() }\n", "func g() {}\n",
	}
	want := []string{src[0], src[2], src[3], src[4], src[1]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_TuckedTypeWithUserBelow(t *testing.T) {
	src := []string{
		"func Run() { _ = obj{} }\n", "func Other() {}\n",
		"type obj struct{}\n", "func (o obj) ID() string { return \"\" }\n",
		"func New() obj { return obj{} }\n",
	}
	want := []string{src[0], src[2], src[3], src[1], src[4]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}
