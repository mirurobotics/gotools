package funcorder

import "testing"

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

func TestCheck_PlainTypeAboveTypeItHolds(t *testing.T) {
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

func TestCheck_PlainTypeBelowItsConst(t *testing.T) {
	assertMessages(t, `package p

const (
	A Kind = iota
)

type Kind int

func f(k Kind) {}
`,
		`type "Kind" should be directly above "A", which uses only it`,
	)
}

func TestCheck_PlainTypeAboveItsDistantConst(t *testing.T) {
	assertMessages(t, `package p

type Kind int

func f(Kind) {}

func g() {}

const A Kind = 0
`,
		`"A" should be grouped directly below type "Kind"`,
	)
}

func TestCheck_PlainTypeConstAfterUser(t *testing.T) {
	assertMessages(t, `package p

type K int

const A K = 0

func f(K) {}

const B K = 0
`,
		`"B" should be grouped directly below type "K"`,
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
