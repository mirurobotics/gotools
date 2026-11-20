package tempdir

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/analysis"
)

// Check reports t.TempDir references; use test_dirs.CreateTemp instead.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	pkg := analysis.PkgImportName(f, "testing")
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
		if analysis.NolintOnLines(fset, f, "tempdir", pos.Line, pos.Line) {
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

func baseName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}
