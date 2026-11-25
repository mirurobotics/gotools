package analysis

import "go/ast"

// TestingPkgName returns the local name of the testing import, or "".
func TestingPkgName(file *ast.File) string { return PkgImportName(file, "testing") }

// IsTestingType reports whether expr is a test handle or *testing.M.
func IsTestingType(expr ast.Expr, testingPkg string) bool {
	if IsTestingHandle(expr, testingPkg) {
		return true
	}
	name, isPtr := QualifiedTypeName(expr, testingPkg)
	return isPtr && name == "M"
}

// IsTestingHandle reports whether expr is *testing.T, B or F, or testing.TB.
func IsTestingHandle(expr ast.Expr, testingPkg string) bool {
	name, isPtr := QualifiedTypeName(expr, testingPkg)
	switch name {
	case "T", "B", "F":
		return isPtr
	case "TB":
		return !isPtr
	}
	return false
}

// IsTestingIface reports whether expr is TestingI or pkg.TestingI.
func IsTestingIface(expr ast.Expr) bool {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if _, isPkg := sel.X.(*ast.Ident); !isPkg {
			return false
		}
		expr = sel.Sel
	}
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "TestingI"
}
