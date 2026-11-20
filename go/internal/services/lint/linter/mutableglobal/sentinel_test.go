package mutableglobal

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestCheck_ErrorSentinelAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "errors"

var ErrNotFound = errors.New("not found")
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FmtErrorfSentinelAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "fmt"

var ErrBadInput = fmt.Errorf("bad input: %s", "x")
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NonErrorCallExpressionFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "os"

var stdout = os.Stdout
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_CustomNewFooSentinelAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var ErrNotFound = myerrors.NewNotFound("x")
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_ErrorfSuffixSentinelAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var ErrUnavailable = grpc.Errorf("unavailable")
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_LowercaseErrPrefixWithPkgNewAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var errInternal = pkg.New("x")
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NonErrVarWithPkgNewStillFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var counter = pkg.New("x")
`,
		wantN:   1,
		wantMsg: "mutable global variable: use const or a function instead",
	})
}

func TestCheck_NewLowercaseNotTreatedAsConstructor(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var ErrBad = pkg.Newsletter("x")
`,
		wantN:   1,
		wantMsg: "mutable global variable: use const or a function instead",
	})
}

func TestCheck_UppercaseErrNameAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var Err = errors.New("x")
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_LowercaseErrNameAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var err = errors.New("x")
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_LessThan3CharErNameFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var Er = errors.New("x")
`,
		wantN:   1,
		wantMsg: "mutable global variable: use const or a function instead",
	})
}

func TestCheck_UnqualifiedNewNotTreatedAsConstructor(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func New(s string) error { return nil }

var ErrBad = New("x")
`,
		wantN:   1,
		wantMsg: "mutable global variable: use const or a function instead",
	})
}

func TestCheck_AllCapsERRNotTreatedAsErrPrefix(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var ERR = errors.New("x")
`,
		wantN:   1,
		wantMsg: "mutable global variable: use const or a function instead",
	})
}

func TestIsErrorConstructor_NotPkgSelector(t *testing.T) {
	for _, src := range []string{"newErr()", "a.b.New()"} {
		expr, err := parser.ParseExpr(src)
		if err != nil {
			t.Fatal(err)
		}
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			t.Fatalf("%s is %T, want *ast.CallExpr", src, expr)
		}
		if isErrorConstructor(call) {
			t.Errorf("isErrorConstructor(%s) = true, want false", src)
		}
	}
}
