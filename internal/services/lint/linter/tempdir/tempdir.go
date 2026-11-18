package tempdir

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports t.TempDir references; use test_dirs.CreateTemp instead.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	pkg := testingPkgName(f)
	if pkg == "" {
		return nil
	}

	handles := handleNames(f, pkg)
	if len(handles) == 0 {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "TempDir" || !handles[baseName(sel.X)] {
			return true
		}

		pos := fset.Position(sel.Pos())
		if hasNolint(fset, f, pos.Line) {
			return true
		}

		diags = append(diags, analysis.Diagnostic{
			File: filename,
			Line: pos.Line,
			Message: "t.TempDir() is not allowed; use test_dirs.CreateTemp(t)" +
				" to create temporary test directories",
		})

		return true
	})

	return diags
}

// testingPkgName returns the local name of the testing import, or "".
func testingPkgName(f *ast.File) string {
	for _, imp := range f.Imports {
		if imp.Path.Value != `"testing"` {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "testing"
	}
	return ""
}

// handleNames returns the identifier names declared with a testing handle type.
func handleNames(f *ast.File, pkg string) map[string]bool {
	names := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok || !isHandleType(field.Type, pkg) {
			return true
		}
		for _, name := range field.Names {
			names[name.Name] = true
		}
		return true
	})
	return names
}

// isHandleType reports whether expr is a testing handle type.
func isHandleType(expr ast.Expr, pkg string) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != pkg {
		return false
	}

	switch sel.Sel.Name {
	case "T", "B", "F", "TB":
		return true
	}
	return false
}

// baseName returns the identifier a selector's receiver resolves to, or "".
func baseName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}

// hasNolint returns true if there is a //nolint:tempdir comment on the given line.
func hasNolint(fset *token.FileSet, f *ast.File, line int) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if fset.Position(c.Pos()).Line == line &&
				strings.Contains(c.Text, "nolint:tempdir") {
				return true
			}
		}
	}
	return false
}
