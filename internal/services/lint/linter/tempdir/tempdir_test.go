package tempdir

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

const wantMessage = "t.TempDir() is not allowed; use test_dirs.CreateTemp(t)" +
	" to create temporary test directories"

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
}

func TestCheck_TTempDirFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func TestF(t *testing.T) {
	dir := t.TempDir()
	_ = dir
}
`,
		wantN:   1,
		wantMsg: wantMessage,
	})
}

func TestCheck_BTempDirFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func BenchmarkF(b *testing.B) {
	_ = b.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_FTempDirFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func FuzzF(f *testing.F) {
	_ = f.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_TestingTBHandleFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "helper.go",
		src: `package foo

import "testing"

func helper(tb testing.TB) string {
	return tb.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_HandleHeldOnAFixtureStructFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

type fixture struct {
	t *testing.T
}

func (s *fixture) dir() string {
	return s.t.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_HandlePassedToAPlainHelperFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func run(t *testing.T) {
	_ = t.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_NoTestingImportNotFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

type fake struct{}

func (f fake) TempDir() string { return "" }

func g() string {
	t := fake{}
	return t.TempDir()
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TempDirOnNonHandleReceiverNotFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

type fake struct{}

func (f fake) TempDir() string { return "" }

func TestF(t *testing.T) {
	other := fake{}
	_ = other.TempDir()
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_OtherHandleMethodsNotFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func TestF(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {})
	t.Setenv("K", "V")
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintCommentOnSameLine(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "os.go",
		src: `package foo

import "testing"

func CreateTemp(t *testing.T) string {
	return t.TempDir() //nolint:tempdir
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintOnDifferentLineDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func TestF(t *testing.T) {
	//nolint:tempdir
	_ = t.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_NolintOtherRuleDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func TestF(t *testing.T) {
	_ = t.TempDir() //nolint:bgctx
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_AliasedTestingImportFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import tst "testing"

func TestF(t *tst.T) {
	_ = t.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_MultipleCallsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func TestA(t *testing.T) { _ = t.TempDir() }
func TestB(t *testing.T) { _ = t.TempDir() }
`,
		wantN:   2,
		wantMsg: "",
	})
}

func TestCheck_NonTestingFieldTypesIgnored(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import (
	"os"
	"testing"
)

type fixture struct {
	f *os.File
	m *testing.M
	t *testing.T
}

func (s *fixture) dir() string {
	return s.t.TempDir()
}

func (s *fixture) other() string {
	return s.m.TempDir()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_TestingImportedButNoHandleDeclared(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

var _ = testing.Verbose

type fake struct{}

func (f fake) TempDir() string { return "" }

func g() string { return fake{}.TempDir() }
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_MethodValueFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

func TestF(t *testing.T) {
	fns := []func() string{t.TempDir}
	_ = fns[0]()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_CallExpressionReceiverNotFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "testing"

type fake struct{}

func (f fake) TempDir() string { return "" }

func newFake() fake { return fake{} }

func TestF(t *testing.T) {
	_ = newFake().TempDir()
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	fset, file := linttest.Parse(t, tc.filename, tc.src)
	linttest.AssertCount(t, Check(fset, tc.filename, file), tc.wantN, tc.wantMsg)
}
