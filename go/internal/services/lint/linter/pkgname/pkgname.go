package pkgname

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/analysis"
)

// Check reports package names with underscores, except test_, mock_, and _test.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	name := f.Name.Name
	if !strings.Contains(name, "_") {
		return nil
	}
	if strings.HasPrefix(name, "test_") || strings.HasPrefix(name, "mock_") {
		return nil
	}
	if strings.HasSuffix(name, "_test") {
		return nil
	}
	pos := fset.Position(f.Name.Pos())
	return []analysis.Diagnostic{{
		File: filename,
		Line: pos.Line,
		Message: fmt.Sprintf(
			"package name %q contains underscore; use camelCase instead",
			name,
		),
	}}
}
