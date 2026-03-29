package ctxpos

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
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
		var params *ast.FieldList
		var pos token.Pos

		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Type.Params == nil {
				return true
			}
			params = fn.Type.Params
			pos = fn.Pos()
		case *ast.FuncLit:
			if fn.Type.Params == nil {
				return true
			}
			params = fn.Type.Params
			pos = fn.Pos()
		default:
			return true
		}

		ctxIdx := -1
		paramIdx := 0
		for _, field := range params.List {
			names := len(field.Names)
			if names == 0 {
				names = 1
			}
			if analysis.IsContextType(field.Type, ctxPkg) {
				ctxIdx = paramIdx
				break
			}
			paramIdx += names
		}

		if ctxIdx > 0 && !firstParamIsTesting(params) {
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
