package commentedcode

import "go/ast"

func anyPlausibleDecl(decls []ast.Decl) bool {
	for _, d := range decls {
		if plausibleDecl(d) {
			return true
		}
	}
	return false
}

func plausibleDecl(d ast.Decl) bool {
	if fd, isFunc := d.(*ast.FuncDecl); isFunc {
		return fd.Body != nil
	}
	gd, isGen := d.(*ast.GenDecl)
	return isGen && (len(gd.Specs) > 1 || allPlausibleSpecs(gd.Specs))
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
