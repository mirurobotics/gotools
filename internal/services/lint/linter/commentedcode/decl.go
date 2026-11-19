package commentedcode

import "go/ast"

func loneBareDecl(body *ast.BlockStmt) bool {
	if len(body.List) != 1 {
		return false
	}
	ds, isDecl := body.List[0].(*ast.DeclStmt)
	return isDecl && !plausibleDecl(ds.Decl)
}

func plausibleDecl(d ast.Decl) bool {
	if fd, isFunc := d.(*ast.FuncDecl); isFunc {
		return fd.Body != nil
	}
	gd, isGen := d.(*ast.GenDecl)
	return isGen && (gd.Lparen.IsValid() || allPlausibleSpecs(gd.Specs))
}

func allPlausibleSpecs(specs []ast.Spec) bool {
	for _, s := range specs {
		if !plausibleSpec(s) {
			return false
		}
	}
	return true
}

func plausibleSpec(s ast.Spec) bool {
	switch x := s.(type) {
	case *ast.ValueSpec:
		return len(x.Values) > 0 || isComposite(x.Type)
	case *ast.TypeSpec:
		return x.Assign.IsValid() || isComposite(x.Type)
	}
	return true
}

func isComposite(e ast.Expr) bool {
	switch e.(type) {
	case nil, *ast.Ident, *ast.SelectorExpr:
		return false
	}
	return true
}
