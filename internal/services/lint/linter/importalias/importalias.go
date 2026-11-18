package importalias

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports a diagnostic for each import alias that contains an
// underscore. Aliases with a "test_" or "mock_" prefix are allowed.
// Test files are exempt.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if analysis.IsTestFile(filename) {
		return nil
	}
	var diags []analysis.Diagnostic
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gd.Specs {
			is, ok := spec.(*ast.ImportSpec)
			if !ok || is.Name == nil || !isBadAlias(is.Name.Name) {
				continue
			}
			pos := fset.Position(is.Name.Pos())
			diags = append(diags, analysis.Diagnostic{
				File: filename,
				Line: pos.Line,
				Message: fmt.Sprintf(
					"import alias %q contains underscore; use camelCase instead",
					is.Name.Name,
				),
			})
		}
	}
	return diags
}

// isBadAlias reports whether an import alias contains an underscore and is
// not a blank, dot, or "test_"/"mock_" prefixed alias.
func isBadAlias(name string) bool {
	if name == "_" || name == "." || !strings.Contains(name, "_") {
		return false
	}
	return !strings.HasPrefix(name, "test_") && !strings.HasPrefix(name, "mock_")
}
