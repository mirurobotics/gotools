package funcorder

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

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

func TestFix_TypeMovesDownToMethods(t *testing.T) {
	typ := "type runner struct{}\n"
	run := "func Run() { r := runner{}; r.run() }\n"
	method := "func (r runner) run() {}\n"
	assertFix(t, "package p\n\n"+typ+"\n"+run+"\n"+method,
		"package p\n\n"+run+"\n"+typ+"\n"+method)
}

func TestFix_TypeCarriesItsConsts(t *testing.T) {
	typ := "type Kind int\n\nconst A Kind = 0\n"
	run := "func Run() Kind { return A }\n"
	method := "func (k Kind) String() string { return \"\" }\n"
	assertFix(t, "package p\n\n"+typ+"\n"+run+"\n"+method,
		"package p\n\n"+run+"\n"+typ+"\n"+method)
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

func assertFix(t *testing.T, src, want string) {
	t.Helper()
	fset, f := linttest.Parse(t, "a.go", src)
	got := string(Fix(fset, f, []byte(src)))
	if got != want {
		t.Fatalf("Fix:\n%s\nwant:\n%s", got, want)
	}
	assertStable(t, got)
}
