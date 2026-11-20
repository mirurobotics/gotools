package commentedcode

import "go/ast"

func anyPlausibleDecl(decls []ast.Decl) bool {
	for _, decl := range decls {
		if plausibleDecl(decl) {
			return true
		}
	}
	return false
}

func plausibleDecl(decl ast.Decl) bool {
	if funcDecl, isFunc := decl.(*ast.FuncDecl); isFunc {
		return funcDecl.Body != nil
	}
	genDecl, isGen := decl.(*ast.GenDecl)
	return isGen && (len(genDecl.Specs) > 1 || allPlausibleSpecs(genDecl.Specs))
}

func allPlausibleSpecs(specs []ast.Spec) bool {
	for _, spec := range specs {
		if !plausibleSpec(spec) {
			return false
		}
	}
	return true
}

func plausibleSpec(spec ast.Spec) bool {
	switch spec := spec.(type) {
	case *ast.ValueSpec:
		return len(spec.Values) > 0 || isComposite(spec.Type)
	case *ast.TypeSpec:
		return spec.Assign.IsValid() || isComposite(spec.Type)
	}
	return true
}

func isComposite(expr ast.Expr) bool {
	switch expr.(type) {
	case nil, *ast.Ident, *ast.SelectorExpr:
		return false
	}
	return true
}
