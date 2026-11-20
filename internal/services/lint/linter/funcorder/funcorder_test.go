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
		"exported callee of unexported caller": `package p

func Get() int { return 1 }

func get() int { return Get() }
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
		`"helper" should be below its first caller "other"`,
		`"other" should be below its first caller "Run"`,
		`"Run" should be above unexported "helper"`,
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
		`"Get" should be below its first caller "Run"`,
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
		`"T.helper" should be below its first caller "T.Run"`,
		`method "T.helper" should be below type "T"`,
		`"T.Run" should be above unexported "T.helper"`,
		`method "T.Run" should be grouped with the other methods of "T"`,
		`"T.Other" should be above unexported "T.helper"`,
		`method "T.Other" should be grouped with the other methods of "T"`,
	)
}

func TestCheck_UnexportedTypeMethods(t *testing.T) {
	assertMessages(t, `package p

type runner struct{}

func (r runner) Run() {}

func (_ runner) Stop() {}

func (runner) Wait() {}

func (r (runner)) wrapped() {}

func New() runner { return runner{} }
`,
		`"New" should be above unexported "runner.Run"`,
	)
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
		fset, f = linttest.Parse(t, "a.go", string(fixed))
		assertClean(t, Check(fset, "a.go", f))
	}
}

func assertFix(t *testing.T, src, want string) {
	t.Helper()
	fset, f := linttest.Parse(t, "a.go", src)
	got := string(Fix(fset, f, []byte(src)))
	if got != want {
		t.Fatalf("Fix:\n%s\nwant:\n%s", got, want)
	}
	fset, f = linttest.Parse(t, "a.go", got)
	assertClean(t, Check(fset, "a.go", f))
}

func messages(diags []analysis.Diagnostic) string {
	msgs := make([]string, len(diags))
	for i, d := range diags {
		msgs[i] = d.Message
	}
	return strings.Join(msgs, "\n")
}

func assertClean(t *testing.T, diags []analysis.Diagnostic) {
	t.Helper()
	if len(diags) != 0 {
		t.Errorf("fixed source still has diagnostics: %v", diags)
	}
}
