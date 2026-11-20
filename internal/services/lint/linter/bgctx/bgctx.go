package bgctx

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for calls to context.Background().
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	ctxPkg := analysis.ContextPkgName(file)
	if ctxPkg == "" {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isBackgroundCall(call, ctxPkg) {
			return true
		}

		pos := fset.Position(call.Pos())
		if analysis.NolintOnLines(fset, file, "bgctx", pos.Line, pos.Line) {
			return true
		}

		diags = append(diags, analysis.Diagnostic{
			File: filename,
			Line: pos.Line,
			Message: "context.Background() is not allowed;" +
				" use mctx package to create contexts",
		})

		return true
	})

	return diags
}

func isBackgroundCall(call *ast.CallExpr, ctxPkg string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == ctxPkg && sel.Sel.Name == "Background"
}
