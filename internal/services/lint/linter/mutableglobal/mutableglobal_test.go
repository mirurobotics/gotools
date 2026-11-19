package mutableglobal

import (
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, tt.filename, tt.src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diags := Check(fset, tt.filename, f)
	if len(diags) != tt.wantN {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
	}
	if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
		t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
	}
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
