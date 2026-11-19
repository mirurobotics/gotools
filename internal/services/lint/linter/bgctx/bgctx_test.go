package bgctx

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, tt.filename, tt.src)
	linttest.AssertCount(t, Check(fset, tt.filename, f), tt.wantN, tt.wantMsg)
}

func TestCheck_ContextBackgroundFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "context"

func f() context.Context {
	return context.Background()
}
`,
		wantN: 1,
		wantMsg: "context.Background() is not allowed;" +
			" use mctx package to create contexts",
	})
}

func TestCheck_ContextTODONotFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "context"

func f() context.Context {
	return context.TODO()
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NoContextImportNotFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func f() string {
	return "hello"
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

import "context"

func f() context.Context {
	return context.Background()
}
`,
		wantN: 1,
		wantMsg: "context.Background() is not allowed;" +
			" use mctx package to create contexts",
	})
}

func TestCheck_NolintCommentOnSameLine(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "context"

func f() context.Context {
	return context.Background() //nolint:bgctx
}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintCommentOnDifferentLineDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "context"

func f() context.Context {
	//nolint:bgctx
	return context.Background()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_MultipleCallsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "context"

func a() context.Context { return context.Background() }
func b() context.Context { return context.Background() }
`,
		wantN:   2,
		wantMsg: "",
	})
}

func TestCheck_AliasedContextImportFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import ctx "context"

func f() ctx.Context {
	return ctx.Background()
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestCheck_NolintOtherRuleDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

import "context"

func f() context.Context {
	return context.Background() //nolint:mutableglobal
}
`,
		wantN:   1,
		wantMsg: "",
	})
}

func TestIsBackgroundCall(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"context.Background()", true},
		{"context.TODO()", false},
		{"Background()", false},
		{"a.context.Background()", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			e, err := parser.ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			call, ok := e.(*ast.CallExpr)
			if !ok {
				t.Fatalf("%q is not a call", tt.expr)
			}
			if got := isBackgroundCall(call, "context"); got != tt.want {
				t.Errorf("isBackgroundCall(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}
