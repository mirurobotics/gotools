package tempdir

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports t.TempDir references; use test_dirs.CreateTemp instead.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	pkg := analysis.TestingPkgName(file)
	if pkg == "" {
		return nil
	}

	handles := handleNames(file, pkg)
	if len(handles) == 0 {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(file, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "TempDir" || !handles[baseName(sel.X)] {
			return true
		}

		pos := fset.Position(sel.Pos())
		if analysis.NolintOnLines(fset, file, "tempdir", pos.Line, pos.Line) {
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

func handleNames(file *ast.File, pkg string) map[string]bool {
	names := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok || !analysis.IsTestingHandle(field.Type, pkg) {
			return true
		}
		for _, name := range field.Names {
			names[name.Name] = true
		}
		return true
	})
	return names
}

func baseName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	}
	return ""
}
