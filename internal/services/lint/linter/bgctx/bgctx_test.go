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

func TestCheck_NolintOnDifferentLineDoesNotExempt(t *testing.T) {
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

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	fset, file := linttest.Parse(t, tc.filename, tc.src)
	linttest.AssertCount(t, Check(fset, tc.filename, file), tc.wantN, tc.wantMsg)
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
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			parsed, err := parser.ParseExpr(tc.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			call, ok := parsed.(*ast.CallExpr)
			if !ok {
				t.Fatalf("%q is not a call", tc.expr)
			}
			if got := isBackgroundCall(call, "context"); got != tc.want {
				t.Errorf("isBackgroundCall(%q) = %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}
