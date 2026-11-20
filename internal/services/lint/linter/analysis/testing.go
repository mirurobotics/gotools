package analysis

import "go/ast"

// TestingPkgName returns the local name of the testing import, or "".
func TestingPkgName(file *ast.File) string { return PkgImportName(file, "testing") }

// IsTestingType reports whether expr is a test handle or *testing.M.
func IsTestingType(expr ast.Expr, testingPkg string) bool {
	if IsTestingHandle(expr, testingPkg) {
		return true
	}
	name, isPtr := TestingTypeName(expr, testingPkg)
	return isPtr && name == "M"
}

// IsTestingHandle reports whether expr is *testing.T, B or F, or testing.TB.
func IsTestingHandle(expr ast.Expr, testingPkg string) bool {
	name, isPtr := TestingTypeName(expr, testingPkg)
	switch name {
	case "T", "B", "F":
		return isPtr
	case "TB":
		return !isPtr
	}
	return false
}

// TestingTypeName returns X for testing.X or *testing.X, or "" for other types.
func TestingTypeName(expr ast.Expr, testingPkg string) (name string, isPtr bool) {
	if testingPkg == "" {
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
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != testingPkg {
		return "", false
	}
	return sel.Sel.Name, isPtr
}
