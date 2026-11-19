package ctxpos

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diags := Check(fset, "test.go", f)
	if len(diags) != tt.wantN {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
	}
	if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
		t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
	}
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

func TestCheck_PointerToLocalTypeFirstThenContextFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; type myType struct{}` +
			`; func f(t *myType, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_TestingNonTBFMTypeFirstThenContextFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "context"` +
			`; import "testing"` +
			`; func f(r *testing.PB, ctx context.Context) {}`,
		wantN: 1,
		wantMsg: "context.Context should be" +
			" the first parameter",
	})
}

func TestCheck_PointerToNonTestingPackageFirstThenContextFlagged(t *testing.T) {
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

func TestFuncParams_NilParams(t *testing.T) {
	//nolint:exhaustruct // only Type matters here
	fn := &ast.FuncDecl{Type: &ast.FuncType{}}
	if _, _, ok := funcParams(fn); ok {
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := parser.ParseExpr(tt.sig)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			ft, ok := e.(*ast.FuncType)
			if !ok {
				t.Fatalf("%q is not a func type", tt.sig)
			}
			if got := ctxParamIndex(ft.Params, "context"); got != tt.want {
				t.Errorf("ctxParamIndex(%q) = %d, want %d", tt.sig, got, tt.want)
			}
		})
	}
}
