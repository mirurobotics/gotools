package ctxpos

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports a diagnostic when context.Context is present in a function's
// parameter list but is not the first parameter.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	ctxPkg := analysis.ContextPkgName(f)
	if ctxPkg == "" {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(f, func(n ast.Node) bool {
		params, pos, ok := funcParams(n)
		if !ok {
			return true
		}
		if ctxParamIndex(params, ctxPkg) > 0 && !firstParamIsTesting(params) {
			p := fset.Position(pos)
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    p.Line,
				Message: "context.Context should be the first parameter",
			})
		}
		return true
	})

	return diags
}

// funcParams returns the parameter list and position of a function
// declaration or literal. ok is false for other nodes and for functions
// with a nil parameter list.
func funcParams(n ast.Node) (*ast.FieldList, token.Pos, bool) {
	var ft *ast.FuncType
	switch fn := n.(type) {
	case *ast.FuncDecl:
		ft = fn.Type
	case *ast.FuncLit:
		ft = fn.Type
	default:
		return nil, token.NoPos, false
	}
	if ft.Params == nil {
		return nil, token.NoPos, false
	}
	return ft.Params, n.Pos(), true
}

// ctxParamIndex returns the index of the first context.Context parameter,
// or -1 if there is none.
func ctxParamIndex(params *ast.FieldList, ctxPkg string) int {
	paramIdx := 0
	for _, field := range params.List {
		if analysis.IsContextType(field.Type, ctxPkg) {
			return paramIdx
		}
		names := len(field.Names)
		if names == 0 {
			names = 1
		}
		paramIdx += names
	}
	return -1
}

// firstParamIsTesting returns true if the first parameter's type is
// *testing.T, *testing.B, *testing.F, or *testing.M. These types must
// be the first parameter by Go convention, so context.Context is
// allowed in the second position.
func firstParamIsTesting(params *ast.FieldList) bool {
	typ := params.List[0].Type
	star, ok := typ.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if ident.Name != "testing" {
		return false
	}
	switch sel.Sel.Name {
	case "T", "B", "F", "M":
		return true
	}
	return false
}
