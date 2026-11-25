package analysis

import "go/ast"

// BaseIdent returns the named type under pointers, parens, type args, and selectors.
func BaseIdent(expr ast.Expr) *ast.Ident {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr
	case *ast.StarExpr:
		return BaseIdent(expr.X)
	case *ast.ParenExpr:
		return BaseIdent(expr.X)
	case *ast.SelectorExpr:
		return expr.Sel
	case *ast.IndexExpr:
		return BaseIdent(expr.X)
	case *ast.IndexListExpr:
		return BaseIdent(expr.X)
	}
	return nil
}

// QualifiedTypeName returns X for pkg.X or *pkg.X, or "" for other types.
func QualifiedTypeName(expr ast.Expr, pkg string) (name string, isPtr bool) {
	if pkg == "" {
		return "", false
	}
	star, isPtr := expr.(*ast.StarExpr)
	if isPtr {
		expr = star.X
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok || pkgIdent.Name != pkg {
		return "", false
	}
	return sel.Sel.Name, isPtr
}
