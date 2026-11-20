package funcorder

import (
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

func TestCheck_Ordered(t *testing.T) {
	cases := map[string]string{
		"stepdown": `package p

func Run() { a(); b() }

func a() { c() }

func b() { c() }

func c() {}
`,
		"recursion and cycles": `package p

func Run() { a() }

func b() { a(); z() }

func a() { a(); b() }

func z() {}
`,
		"exported callee below unexported caller": `package p

func get() int { return Get() }

func Get() int { return 1 }
`,
		"shared helper below last caller": `package p

func A() { a1(); h() }

func a1() {}

func B() { b1(); h() }

func b1() {}

func h() {}
`,
		"cycle member is not an entry point": `package p

func Run() { a() }

func a() { b() }

func b() { a() }

func Other() {}
`,
		"exported cycle member is not an entry point": `package p

func c() {}

func a() { B() }

func B() { a() }
`,
		"grouped type declaration": `package p

type (
	A struct{}
	B struct{}
)

func Run() {}

func (a A) Get() {}
`,
		"exported entry above unexported entry": `package p

func Run() { helper() }

func helper() {}

func unused() {}
`,
		"entry points and blank functions": `package main

func init() { setup() }

func init() {}

func main() { run() }

func _() {}

func setup() {}

func run() {}
`,
		"exempt function": `package p

//nolint:funcorder // kept near its data
func helper() {}

// Run runs.
func Run() { helper() }
`,
		"shadowing local": `package p

func Value() {}

func Run() { Value := 1; _ = Value }
`,
		"exempt caller": `package p

type T struct{}

func (t T) A() {}

func (t T) h() {}

//nolint:funcorder // kept last
func (t T) E() { t.h() }
`,
		"exempt method inside its group": `package p

type T struct{}

func (t T) A() {}

//nolint:funcorder // kept here
func (t T) e() {}

func (t T) B() {}
`,
		"shadowed receiver": `package p

type U struct{}

func (u U) Help() {}

type T struct{}

func (t T) Help() {}

func (t T) Run() {
	for _, t := range []U{{}} {
		t.Help()
	}
}
`,
		"composite literal key": `package p

type T struct{ helper int }

func helper() {}

func build() T { return T{helper: 1} }
`,
		"function without body": `package p

func Asm()

func Run() {}

func (s []int) bad() {}
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) { assertMessages(t, src) })
	}
}

func TestCheck_Stepdown(t *testing.T) {
	assertMessages(t, `package p

func helper() {}

func other() { helper() }

func Run() { other(); _ = helper }
`,
		`"helper" should be below its last caller "Run"`,
		`"other" should be below its last caller "Run"`,
	)
}

func TestCheck_SharedHelperBelowLastCaller(t *testing.T) {
	assertMessages(t, `package p

func A() { a1(); h() }

func h() {}

func a1() {}

func B() { b1(); h() }

func b1() {}
`,
		`"h" should be below its last caller "B"`,
	)
}

func TestCheck_ExportedMethodEntryFirst(t *testing.T) {
	assertMessages(t, `package p

type T struct{}

func (T) a() {}

func (T) B() {}
`,
		`"T.B" should be above unexported "T.a"`,
	)
}

func TestCheck_TypeAboveMethods(t *testing.T) {
	assertMessages(t, `package p

type runner struct{}

func Run() { r := runner{}; r.run() }

func (r runner) run() {}
`,
		`type "runner" should be directly above its methods`,
	)
}

func TestCheck_ExportedCalledInFile(t *testing.T) {
	assertMessages(t, `package p

func Load(path string) int { return loadFile(path) }

func Parse(b []byte) int { return validate(b) }

func loadFile(path string) int { return Parse(read(path)) }

func read(path string) []byte { return nil }

func validate(b []byte) int { return len(b) }
`,
		`"Parse" should be below its last caller "loadFile"`,
	)
}

func TestCheck_ExportedEntryFirst(t *testing.T) {
	assertMessages(t, `package p

func unused() {}

func Run() {}
`,
		`"Run" should be above unexported "unused"`,
	)
}

func TestCheck_ExportedCallees(t *testing.T) {
	assertMessages(t, `package p

func Get() int { return 1 }

func Run() int { return Get() }

type T struct{}

func (t T) Value() int { return Run() + help() }

func help() int { return 1 }
`,
		`"Get" should be below its last caller "Run"`,
		`"Run" should be below its last caller "T.Value"`,
	)
}

func TestCheck_Methods(t *testing.T) {
	assertMessages(t, `package p

func (t *T) helper() {}

type T struct{}

func (t *T) Run() { t.helper(); t.Run() }

func free() {}

func (t T) Other() {}
`,
		`"T.helper" should be below its last caller "T.Run"`,
		`method "T.helper" should be below type "T"`,
		`method "T.Run" should be grouped with the other methods of "T"`,
		`method "T.Other" should be grouped with the other methods of "T"`,
	)
}

func TestCheck_UnexportedTypeMethodsStayWithType(t *testing.T) {
	assertMessages(t, `package p

type runner struct{}

func (r runner) Run() {}

func (_ runner) Stop() {}

func (runner) Wait() {}

func (r (runner)) wrapped() {}

func New() runner { return runner{} }
`)
}

func TestCheck_ParenReceiver(t *testing.T) {
	assertMessages(t, "package p\n\nfunc (t (T)) m() {}\n\ntype T struct{}\n",
		`method "T.m" should be below type "T"`)
}

func assertMessages(t *testing.T, src string, want ...string) {
	t.Helper()
	fset, f := linttest.Parse(t, "a.go", src)
	diags := Check(fset, "a.go", f)
	if got, exp := messages(diags), strings.Join(want, "\n"); got != exp {
		t.Errorf("messages:\n%s\nwant:\n%s", got, exp)
	}
	fixed := Fix(fset, f, []byte(src))
	if (fixed == nil) != (len(diags) == 0) {
		t.Errorf("Fix changed = %v, Check found %d", fixed != nil, len(diags))
	}
	if fixed != nil {
		assertStable(t, string(fixed))
	}
}

func messages(diags []analysis.Diagnostic) string {
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	return strings.Join(msgs, "\n")
}

func assertStable(t *testing.T, fixed string) {
	t.Helper()
	fset, f := linttest.Parse(t, "a.go", fixed)
	if diags := Check(fset, "a.go", f); len(diags) != 0 {
		t.Errorf("fixed source still has diagnostics: %v", diags)
	}
	if again := Fix(fset, f, []byte(fixed)); again != nil {
		t.Errorf("Fix is not idempotent:\n%s", again)
	}
}
