package importalias

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports underscored import aliases, except test_ and mock_.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	for _, is := range f.Imports {
		if is.Name == nil || !isBadAlias(is.Name.Name) {
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
	return diags
}

func isBadAlias(name string) bool {
	if name == "_" || name == "." || !strings.Contains(name, "_") {
		return false
	}
	return !strings.HasPrefix(name, "test_") && !strings.HasPrefix(name, "mock_")
}
