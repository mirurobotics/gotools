package analysis

import "go/ast"

// ContextPkgName returns the local name used for the "context" package,
// or "" if context is not imported.
func ContextPkgName(f *ast.File) string {
	for _, imp := range f.Imports {
		path := imp.Path.Value
		if path == `"context"` {
			if imp.Name != nil {
				return imp.Name.Name
			}
			return "context"
		}
	}
	return ""
}

// IsContextType checks if an expression refers to context.Context.
// Returns false if ctxPkg is empty (context not imported).
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
