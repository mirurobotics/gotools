package funcorder

import (
	"strings"
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

func TestFix_TypeLeavesOtherTypesBehind(t *testing.T) {
	typ := "type T struct{}\n"
	group := "type (\n\tA struct{}\n\tB struct{}\n)\n"
	an := "func (A) N() {}\n"
	run := "func Run() {}\n"
	m := "func (T) M() {}\n"
	assertFix(t, "package p\n\n"+typ+"\n"+group+"\n"+an+"\n"+run+"\n"+m,
		"package p\n\n"+group+"\n"+an+"\n"+run+"\n"+typ+"\n"+m)
}

func TestFix_KeepsVarInitOrder(t *testing.T) {
	typ := "type T struct{}\n"
	a := "var a = record(\"a\")\n"
	f := "func f() {}\n"
	b := "var b = record(\"b\")\n"
	m := "func (T) M() {}\n"
	assertFix(t, "package p\n\n"+typ+"\n"+a+"\n"+f+"\n"+b+"\n"+m,
		"package p\n\n"+a+"\n"+f+"\n"+b+"\n"+typ+"\n"+m)
}

func TestFix_KeepsVarInitOrderMovingUp(t *testing.T) {
	m := "func (T) M() {}\n"
	b := "var b = record(\"b\")\n"
	typ := "type T struct{}\n"
	a := "var a = record(\"a\")\n"
	assertFix(t, "package p\n\n"+m+"\n"+b+"\n"+typ+"\n"+a,
		"package p\n\n"+typ+"\n"+m+"\n"+b+"\n"+a)
}

func TestFix_KeepsVarInitOrderAcrossTypes(t *testing.T) {
	src := []string{
		"type T struct{}\n", "var a = record(\"a\")\n", "func f() {}\n",
		"func (U) M() {}\n", "func (T) N() {}\n", "type U struct{}\n",
		"var u = record(\"u\")\n",
	}
	want := []string{src[1], src[2], src[5], src[3], src[0], src[4], src[6]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_PlacedTypeKeepsItsVars(t *testing.T) {
	src := []string{
		"type Kind int\n", "var kindNames = record(\"k\")\n",
		"func (Kind) String() string { return \"\" }\n", "type Config struct{}\n",
		"var defaultConfig = record(\"c\")\n", "func Load() {}\n",
		"var ErrBad = record(\"e\")\n", "func (Config) Validate() {}\n",
	}
	want := []string{
		src[0], src[1], src[2], src[4], src[5], src[6], src[3], src[7],
	}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_TypeMovesUpWithItsConsts(t *testing.T) {
	m := "func (k Kind) String() string { return \"\" }\n"
	run := "func Run() {}\n"
	typ := "type Kind int\n\nconst A Kind = 0\n"
	assertFix(t, "package p\n\n"+m+"\n"+run+"\n"+typ,
		"package p\n\n"+typ+"\n"+m+"\n"+run)
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
