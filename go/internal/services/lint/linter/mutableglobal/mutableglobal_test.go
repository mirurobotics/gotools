package mutableglobal

import (
	"testing"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
}

func TestCheck_ConstIsAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

const maxRetries = 3
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FunctionIsAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func GetConfig() int { return 42 }
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_MutableVarFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var counter int
`,
		wantN:   1,
		wantMsg: "mutable global variable: use const or a function instead",
	})
}

func TestCheck_MutableVarWithValueFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var name = "hello"
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_MutableMapFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var cache = map[string]string{}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_InterfaceComplianceCheckAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

type Foo struct{}
type Bar interface{ Do() }

var _ Bar = (*Foo)(nil)
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

var testData = "hello"
`,
		wantN:   1,
		wantMsg: "mutable global variable: use const or a function instead",
	})
}

func TestCheck_NolintCommentOnVarBlock(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

//nolint:mutableglobal
var debug = false
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintInlineComment(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var debug = false //nolint:mutableglobal
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_MultipleVarsInOneDecl(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "errors"

var (
	ErrA = errors.New("a")
	counter int
	name string
)
`,
		wantN:   2,
		wantMsg: "",
	})
}

func TestCheck_RegexpMustCompileFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "regexp"

var re = regexp.MustCompile("^foo$")
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_SyncMutexFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "sync"

var mu sync.Mutex
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_NolintOnGroupedVar(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "sync"

//nolint:mutableglobal
var (
	mu sync.Mutex
	counter int
)
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintOnSingleSpecInGroup(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var (
	clean = "ok"
	//nolint:mutableglobal
	dirty = "also ok"
)
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_LocalVarNotFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func f() {
	var x int
	_ = x
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, tt.filename, tt.src)
	linttest.AssertCount(t, Check(fset, tt.filename, f), tt.wantN, tt.wantMsg)
}
