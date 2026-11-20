package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestContextPkgName_Imported(t *testing.T) {
	file := parseTestFile(t, `package foo; import "context"`)
	if got := ContextPkgName(file); got != "context" {
		t.Errorf("got %q, want %q", got, "context")
	}
}

func TestContextPkgName_Aliased(t *testing.T) {
	file := parseTestFile(t, `package foo; import ctx "context"`)
	if got := ContextPkgName(file); got != "ctx" {
		t.Errorf("got %q, want %q", got, "ctx")
	}
}

func TestContextPkgName_NotImported(t *testing.T) {
	file := parseTestFile(t, `package foo; import "fmt"`)
	if got := ContextPkgName(file); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestIsContextType_Match(t *testing.T) {
	src := `package foo
import "context"
func f(ctx context.Context) {}
`
	file := parseTestFile(t, src)
	param := firstParamType(t, file.Decls[1])
	pkg := ContextPkgName(file)
	if !IsContextType(param, pkg) {
		t.Error("expected IsContextType to return true")
	}
}

func TestIsContextType_EmptyPkg(t *testing.T) {
	src := `package foo
import "context"
func f(ctx context.Context) {}
`
	file := parseTestFile(t, src)
	param := firstParamType(t, file.Decls[1])
	if IsContextType(param, "") {
		t.Error("expected false with empty ctxPkg")
	}
}

func TestIsContextType_NonContext(t *testing.T) {
	src := `package foo
func f(x int) {}
`
	file := parseTestFile(t, src)
	param := firstParamType(t, file.Decls[0])
	if IsContextType(param, "context") {
		t.Error("expected false for non-context type")
	}
}

func TestIsContextType_NestedSelector(t *testing.T) {
	src := `package foo
import "go/ast"
func f(x ast.Expr) {}
`
	file := parseTestFile(t, src)
	param := firstParamType(t, file.Decls[1])
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
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func firstParamType(t *testing.T, decl ast.Decl) ast.Expr {
	t.Helper()
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decl is %T, want *ast.FuncDecl", decl)
	}
	return funcDecl.Type.Params.List[0].Type
}
