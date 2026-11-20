package varname

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

func TestCheck_ShortDefineFlagged(t *testing.T) {
	src := "package foo\n\nfunc run() {\n\tn := 1\n\t_ = n\n}\n"
	fset, file := linttest.Parse(t, "foo.go", src)
	linttest.AssertLines(t, Check(fset, "foo.go", file), []int{4}, wantMsg("n"))
}

func TestCheck_TwoCharNameFlagged(t *testing.T) {
	src := "package foo\n\nfunc run() {\n\tdx := 1\n\t_ = dx\n}\n"
	fset, file := linttest.Parse(t, "foo.go", src)
	linttest.AssertLines(t, Check(fset, "foo.go", file), []int{4}, wantMsg("dx"))
}

func TestCheck_ThreeCharNameAllowed(t *testing.T) {
	src := "package foo\n\nfunc run() {\n\tnum := 1\n\t_ = num\n}\n"
	assertFlagged(t, "foo.go", src)
}

func TestCheck_ExemptNames(t *testing.T) {
	src := `package foo

func run(a, b int) bool {
	for i := 0; i < a; i++ {
	}
	m := map[int]int{}
	_, ok := m[b]
	var _ = ok
	id, db := 1, 2
	var wg, mu int
	_, _, _, _ = id, db, wg, mu
	return ok
}
`
	assertFlagged(t, "foo.go", src, 6)
}

func TestCheck_TestCaseNameAllowedInTestsOnly(t *testing.T) {
	src := "package foo\n\nfunc run() {\n\ttc := 1\n\t_ = tc\n}\n"
	assertFlagged(t, "foo.go", src, 4)
	assertFlagged(t, "foo_test.go", src)
}

func TestCheck_MultiByteNameCountsRunes(t *testing.T) {
	src := "package foo\n\nfunc run() {\n\täö := 1\n\t_ = äö\n}\n"
	assertFlagged(t, "foo.go", src, 4)
}

func TestCheck_Params(t *testing.T) {
	src := `package foo

func run(s string, count int) (n int, err error) {
	return 0, nil
}
`
	assertFlagged(t, "foo.go", src, 3, 3)
}

func TestCheck_UnnamedParamsSkipped(t *testing.T) {
	assertFlagged(t, "foo.go", "package foo\n\nfunc run(int, string) {}\n")
}

func TestCheck_FuncLitParams(t *testing.T) {
	src := `package foo

var less = func(x, y int) bool { return x < y }
`
	assertFlagged(t, "foo.go", src, 3, 3)
}

func TestCheck_ReceiverSkipped(t *testing.T) {
	src := "package foo\n\ntype repo struct{}\n\nfunc (r repo) get() {}\n"
	assertFlagged(t, "foo.go", src)
}

func TestCheck_TypeParamsSkipped(t *testing.T) {
	src := "package foo\n\nfunc first[T any](items []T) T { return items[0] }\n"
	assertFlagged(t, "foo.go", src)
}

func TestCheck_TestingParamsSkipped(t *testing.T) {
	src := `package foo

import "testing"

func TestRun(t *testing.T) {}
func BenchmarkRun(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {})
}
func FuzzRun(f *testing.F) {}
func TestMain(m *testing.M) {}
func helper(tb testing.TB) {}
`
	assertFlagged(t, "foo_test.go", src)
}

func TestCheck_AliasedTestingImport(t *testing.T) {
	src := `package foo

import tst "testing"

func TestRun(t *tst.T) {}
`
	assertFlagged(t, "foo_test.go", src)
}

func TestCheck_TestingLookalikesFlagged(t *testing.T) {
	src := `package foo

import (
	"testing"
	"other"
)

func run(t testing.T, tb *testing.TB, x *other.T, y testing.Cover, z *[]int) {}
`
	assertFlagged(t, "foo.go", src, 8, 8, 8, 8, 8)
}

func TestCheck_TestingTypeWithoutImportFlagged(t *testing.T) {
	assertFlagged(t, "foo.go", "package foo\n\nfunc run(t *testing.T) {}\n", 3)
}

func TestCheck_Vars(t *testing.T) {
	src := `package foo

var xs = 1

const xy = 2

type tp int

func run() {
	var (
		n int
		count int
	)
	_, _ = n, count
}
`
	assertFlagged(t, "foo.go", src, 3, 11)
}

func TestCheck_RangeVars(t *testing.T) {
	src := `package foo

func run(items map[string]int) {
	for k, v := range items {
		_, _ = k, v
	}
	for key := range items {
		_ = key
	}
	var k2 string
	for k2 = range items {
	}
	_ = k2
}
`
	assertFlagged(t, "foo.go", src, 4, 4, 10)
}

func TestCheck_RedeclaredSkipped(t *testing.T) {
	src := `package foo

func run() error {
	x, err := 1, error(nil)
	x, err2 := 2, err
	_ = x
	return err2
}
`
	assertFlagged(t, "foo.go", src, 4)
}

func TestCheck_ShadowingShortNameFlaggedTwice(t *testing.T) {
	src := `package foo

func run(cond bool) {
	n := 1
	if cond {
		n := 2
		_ = n
	}
	_ = n
}
`
	assertFlagged(t, "foo.go", src, 4, 6)
}

func TestCheck_NewNameBesideReusedFlagged(t *testing.T) {
	src := `package foo

func run() error {
	num, err := 1, error(nil)
	num, e2 := 2, err
	_ = num
	return e2
}
`
	fset, file := linttest.Parse(t, "foo.go", src)
	linttest.AssertLines(t, Check(fset, "foo.go", file), []int{5}, wantMsg("e2"))
}

func wantMsg(name string) string {
	return `variable name "` + name + `" is shorter than 3 chars;` +
		" use a descriptive name"
}

func TestCheck_NonVariableNamesSkipped(t *testing.T) {
	src := `package foo

type point struct{ x, y int }

type handler func(id int) (n int)

type store interface {
	Get(id string) (v []byte, ok bool)
}
`
	assertFlagged(t, "foo.go", src)
}

func TestCheck_TypeSwitchVar(t *testing.T) {
	src := `package foo

func run(val any) {
	switch v := val.(type) {
	case int:
		_ = v
	}
}
`
	assertFlagged(t, "foo.go", src, 4)
}

func TestCheck_PlainAssignSkipped(t *testing.T) {
	src := "package foo\n\nvar val int\n\nfunc run() {\n\tval = 1\n}\n"
	assertFlagged(t, "foo.go", src)
}

func TestCheck_Nolint(t *testing.T) {
	src := "package foo\n\nfunc run() {\n\tn := 1 //nolint:varname // math convention\n\t_ = n\n}\n"
	assertFlagged(t, "foo.go", src)
}

func TestCheck_TestFilesChecked(t *testing.T) {
	src := "package foo\n\nimport \"testing\"\n\nfunc TestRun(t *testing.T) {\n\tgot := 1\n\tw := 2\n\t_, _ = got, w\n}\n"
	assertFlagged(t, "foo_test.go", src, 7)
}

func TestCheck_TestingLookalikePointersFlagged(t *testing.T) {
	src := `package foo

import "testing"

func run(pb *testing.Cover, pc *[]int, pd *other.PB) {}
`
	assertFlagged(t, "foo.go", src, 5, 5, 5)
}

func assertFlagged(t *testing.T, filename, src string, wantLines ...int) {
	t.Helper()
	fset, file := linttest.Parse(t, filename, src)
	linttest.AssertLines(t, Check(fset, filename, file), wantLines, "")
}
