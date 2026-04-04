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
			if !ok || is.Name == nil {
				continue
			}
			name := is.Name.Name
			if name == "_" || name == "." {
				continue
			}
			if !strings.Contains(name, "_") {
				continue
			}
			if strings.HasPrefix(name, "test_") ||
				strings.HasPrefix(name, "mock_") {
				continue
			}
			pos := fset.Position(is.Name.Pos())
			diags = append(diags, analysis.Diagnostic{
				File: filename,
				Line: pos.Line,
				Message: fmt.Sprintf(
					"import alias %q contains underscore; use camelCase instead",
					name,
				),
			})
		}
	}
	return diags
}
