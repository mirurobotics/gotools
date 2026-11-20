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
