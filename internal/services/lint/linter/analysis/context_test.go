package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestContextPkgName_Imported(t *testing.T) {
	f := parseTestFile(t, `package foo; import "context"`)
	if got := ContextPkgName(f); got != "context" {
		t.Errorf("got %q, want %q", got, "context")
	}
}

func TestContextPkgName_Aliased(t *testing.T) {
	f := parseTestFile(t, `package foo; import ctx "context"`)
	if got := ContextPkgName(f); got != "ctx" {
		t.Errorf("got %q, want %q", got, "ctx")
	}
}

func TestContextPkgName_NotImported(t *testing.T) {
	f := parseTestFile(t, `package foo; import "fmt"`)
	if got := ContextPkgName(f); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestIsContextType_Match(t *testing.T) {
	src := `package foo
import "context"
func f(ctx context.Context) {}
`
	f := parseTestFile(t, src)
	param := firstParamType(t, f.Decls[1])
	pkg := ContextPkgName(f)
	if !IsContextType(param, pkg) {
		t.Error("expected IsContextType to return true")
	}
}

func TestIsContextType_EmptyPkg(t *testing.T) {
	src := `package foo
import "context"
func f(ctx context.Context) {}
`
	f := parseTestFile(t, src)
	param := firstParamType(t, f.Decls[1])
	if IsContextType(param, "") {
		t.Error("expected false with empty ctxPkg")
	}
}

func TestIsContextType_NonContext(t *testing.T) {
	src := `package foo
func f(x int) {}
`
	f := parseTestFile(t, src)
	param := firstParamType(t, f.Decls[0])
	if IsContextType(param, "context") {
		t.Error("expected false for non-context type")
	}
}

func TestIsContextType_NestedSelector(t *testing.T) {
	src := `package foo
import "go/ast"
func f(x ast.Expr) {}
`
	f := parseTestFile(t, src)
	param := firstParamType(t, f.Decls[1])
	inner := &ast.SelectorExpr{
		X:   &ast.SelectorExpr{X: ast.NewIdent("a"), Sel: ast.NewIdent("b")},
		Sel: ast.NewIdent("Context"),
	}
	if IsContextType(inner, "context") {
		t.Error("expected false for nested selector")
	}
	if IsContextType(param, "context") {
		t.Error("expected false for ast.Expr, not context.Context")
	}
}

func parseTestFile(t *testing.T, src string) *ast.File {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func firstParamType(t *testing.T, decl ast.Decl) ast.Expr {
	t.Helper()
	fn, ok := decl.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decl is %T, want *ast.FuncDecl", decl)
	}
	return fn.Type.Params.List[0].Type
}
