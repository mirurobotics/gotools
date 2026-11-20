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

type T struct{}

type U struct{}

func (u U) Help() {}

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

func TestFix_MultiLineTrailingComment(t *testing.T) {
	helper := "func helper() {} /* a\nb */\n"
	run := "func Run() { helper() }\n"
	assertFix(t, "package p\n\n"+helper+"\n"+run, "package p\n\n"+run+"\n"+helper)
}

func TestFix_SharedHelperGroupsUnderLastCaller(t *testing.T) {
	h := "func h() {}\n"
	a := "func A() { a1(); h() }\n"
	a1 := "func a1() {}\n"
	b := "func B() { b1(); h() }\n"
	b1 := "func b1() {}\n"
	src := "package p\n\n" + a + "\n" + h + "\n" + a1 + "\n" + b + "\n" + b1
	want := "package p\n\n" + a + "\n" + a1 + "\n" + b + "\n" + h + "\n" + b1
	assertFix(t, src, want)
}

func TestFix_ExportedCalledInFileMovesDown(t *testing.T) {
	load := "func Load() int { return loadFile() }\n"
	parse := "func Parse() int { return 1 }\n"
	loadFile := "func loadFile() int { return Parse() }\n"
	src := "package p\n\n" + load + "\n" + parse + "\n" + loadFile
	assertFix(t, src, "package p\n\n"+load+"\n"+loadFile+"\n"+parse)
}

func TestFix_PackageLineComment(t *testing.T) {
	head := "package p /* a\nb */\n\n"
	assertFix(t, head+"func helper() {}\n\nfunc Run() { helper() }\n",
		head+"func Run() { helper() }\n\nfunc helper() {}\n")
}

func TestFix_SharedLines(t *testing.T) {
	assertFix(t, "package p; func helper() {}; func Run() {\n\thelper()\n}\n",
		"package p\nfunc Run() {\n\thelper()\n}\n\nfunc helper() {}\n")
}

func TestFix(t *testing.T) {
	src := `package p

import "fmt"

// helper prints.
func helper() { fmt.Println() }

// Section comment.

const x = 1

func (t T) b() { helper() }

// T is a type.
type T struct{}

func (t T) A() { t.b() }

func unrelated() {}

// Run runs.
func Run() { helper() } // trailing

// end of file
`
	want := `package p

import "fmt"

// Section comment.

const x = 1

// T is a type.
type T struct{}

func (t T) A() { t.b() }

func (t T) b() { helper() }

// Run runs.
func Run() { helper() } // trailing

// helper prints.
func helper() { fmt.Println() }

func unrelated() {}

// end of file
`
	assertFix(t, src, want)
}

func TestFix_NoTrailingNewline(t *testing.T) {
	assertFix(t, "package p\n\nfunc a() {}\n\nfunc B() { a() }",
		"package p\n\nfunc B() { a() }\n\nfunc a() {}\n")
}

func TestFix_AlreadyOrdered(t *testing.T) {
	src := "package p\n\nfunc A() { b() }\n\nfunc b() {}\n"
	fset, f := linttest.Parse(t, "a.go", src)
	if got := Fix(fset, f, nil); got != nil {
		t.Errorf("Fix = %q, want nil", got)
	}
}

func TestKahn_NothingReady(t *testing.T) {
	var got []int
	kahn([]int{1, 2}, func(int) bool { return false }, func(i int) {
		got = append(got, i)
	})
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("got %v, want [1 2]", got)
	}
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

func assertFix(t *testing.T, src, want string) {
	t.Helper()
	fset, f := linttest.Parse(t, "a.go", src)
	got := string(Fix(fset, f, []byte(src)))
	if got != want {
		t.Fatalf("Fix:\n%s\nwant:\n%s", got, want)
	}
	assertStable(t, got)
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
