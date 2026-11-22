package ctxpos

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func TestCheck_ContextFirstParam(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; func f(ctx context.Context, x int) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_ContextSecondParam(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; func f(x int, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_NoContextParam(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     `package foo; func f(x int, y string) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FunctionLiteralWithContextNotFirst(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; var f = func(x int, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_FunctionLiteralWithContextFirst(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; var f = func(ctx context.Context, x int) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_AliasedContextImport(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import ctx "context"` +
			`; func f(x int, c ctx.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_ContextNotImported(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     `package foo; func f(x int) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_OnlyParamIsContext(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; func f(ctx context.Context) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_UnnamedContextParamFirst(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; func f(context.Context, int) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestingTFirstThenContextSecond(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "testing"` +
			`; func TestFoo(t *testing.T, ctx context.Context) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestingTBFirstThenContextSecond(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "testing"` +
			`; func helper(tb testing.TB, ctx context.Context) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_AliasedTestingFirstThenContextSecond(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import tst "testing"` +
			`; func TestFoo(t *tst.T, ctx context.Context) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestingBFirstThenContextSecond(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "testing"` +
			`; func BenchmarkFoo(b *testing.B, ctx context.Context) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestingFFirstThenContextSecond(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "testing"` +
			`; func FuzzFoo(f *testing.F, ctx context.Context) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestingMFirstThenContextSecond(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "testing"` +
			`; func TestMain(m *testing.M, ctx context.Context) {}`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NonTestingFirstThenContextStillFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; func f(x int, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_LocalTypePtrFirstThenContextFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; type myType struct{}` +
			`; func f(t *myType, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_TestingNonTBFMFirstThenContextFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "testing"` +
			`; func f(r *testing.PB, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_NonTestingPkgPtrFirstThenContextFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "os"` +
			`; func f(f *os.File, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_NonContextSelectorBeforeContext(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "io"` +
			`; func f(w io.Writer, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	fset, file := linttest.Parse(t, "test.go", tc.src)
	linttest.AssertCount(t, Check(fset, "test.go", file), tc.wantN, tc.wantMsg)
}

func TestFuncParams_NilParams(t *testing.T) {
	//nolint:exhaustruct // only Type matters here
	decl := &ast.FuncDecl{Type: &ast.FuncType{}}
	if _, _, ok := funcParams(decl); ok {
		t.Error("funcParams should report false for nil params")
	}
}

func TestCtxParamIndex(t *testing.T) {
	tests := []struct {
		name string
		sig  string
		want int
	}{
		{"unnamed params", "func(int, string, context.Context)", 2},
		{"grouped params", "func(a, b int, ctx context.Context)", 2},
		{"no context", "func(a int)", -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tc.sig)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			funcType, ok := expr.(*ast.FuncType)
			if !ok {
				t.Fatalf("%q is not a func type", tc.sig)
			}
			if got := ctxParamIndex(funcType.Params, "context"); got != tc.want {
				t.Errorf("ctxParamIndex(%q) = %d, want %d", tc.sig, got, tc.want)
			}
		})
	}
}
