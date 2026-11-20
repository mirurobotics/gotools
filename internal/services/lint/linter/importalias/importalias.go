package importalias

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports underscored import aliases, except test_ and mock_.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	for _, imp := range file.Imports {
		if imp.Name == nil || !isBadAlias(imp.Name.Name) {
			continue
		}
		pos := fset.Position(imp.Name.Pos())
		diags = append(diags, analysis.Diagnostic{
			File: filename,
			Line: pos.Line,
			Message: fmt.Sprintf(
				"import alias %q contains underscore; use camelCase instead",
				imp.Name.Name,
			),
		})
	}
	return diags
}

func isBadAlias(name string) bool {
	if name == "_" || name == "." || !strings.Contains(name, "_") {
		return false
	}
	return !strings.HasPrefix(name, "test_") && !strings.HasPrefix(name, "mock_")
}
