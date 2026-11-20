package analysis

import "go/ast"

// BaseIdent returns the named type under pointers, parens, type args, and selectors.
func BaseIdent(expr ast.Expr) *ast.Ident {
	switch t := expr.(type) {
	case *ast.Ident:
		return t
	case *ast.StarExpr:
		return BaseIdent(t.X)
	case *ast.ParenExpr:
		return BaseIdent(t.X)
	case *ast.SelectorExpr:
		return t.Sel
	case *ast.IndexExpr:
		return BaseIdent(t.X)
	case *ast.IndexListExpr:
		return BaseIdent(t.X)
	}
	return nil
}
