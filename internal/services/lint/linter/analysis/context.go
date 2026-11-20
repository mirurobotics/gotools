package analysis

import "go/ast"

// ContextPkgName returns the local name of the context import, or "".
func ContextPkgName(f *ast.File) string { return PkgImportName(f, "context") }

// IsContextType checks if an expression refers to context.Context.
func IsContextType(expr ast.Expr, ctxPkg string) bool {
	if ctxPkg == "" {
		return false
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == ctxPkg && sel.Sel.Name == "Context"
}
