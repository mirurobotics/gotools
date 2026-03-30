package pkgname

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports a diagnostic if the file's package name contains an underscore,
// unless it has a "test_" or "mock_" prefix or a "_test" suffix (Go's external
// test package convention).
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	name := f.Name.Name
	if !strings.Contains(name, "_") {
		return nil
	}
	if strings.HasPrefix(name, "test_") ||
		strings.HasPrefix(name, "mock_") ||
		strings.HasSuffix(name, "_test") {
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
