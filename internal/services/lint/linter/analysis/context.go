package analysis

import "go/ast"

// ContextPkgName returns the local name of the context import, or "".
func ContextPkgName(file *ast.File) string { return PkgImportName(file, "context") }

// IsContextType checks if an expression refers to context.Context.
func IsContextType(expr ast.Expr, ctxPkg string) bool {
	name, isPtr := QualifiedTypeName(expr, ctxPkg)
	return !isPtr && name == "Context"
}
