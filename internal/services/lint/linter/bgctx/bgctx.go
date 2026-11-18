package bgctx

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for calls to context.Background().
// Use mctx package functions to create contexts instead.
//
// Exemptions:
//   - Test files are exempt.
//   - A call with a //nolint:bgctx comment on the same line is exempt.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if analysis.IsTestFile(filename) {
		return nil
	}

	ctxPkg := analysis.ContextPkgName(f)
	if ctxPkg == "" {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isBackgroundCall(call, ctxPkg) {
			return true
		}

		pos := fset.Position(call.Pos())
		if hasNolint(fset, f, pos.Line) {
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

// isBackgroundCall reports whether call is ctxPkg.Background().
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

// hasNolint returns true if there is a //nolint:bgctx comment on the given line.
func hasNolint(fset *token.FileSet, f *ast.File, line int) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if fset.Position(c.Pos()).Line == line &&
				strings.Contains(c.Text, "nolint:bgctx") {
				return true
			}
		}
	}
	return false
}
