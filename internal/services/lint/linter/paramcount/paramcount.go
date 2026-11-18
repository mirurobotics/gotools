package paramcount

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports a diagnostic for function declarations whose parameter count
// (excluding context.Context) exceeds maxParams. Function literals, main,
// init, Test*, and Benchmark* functions are exempt. Generated files are skipped.
func Check(
	fset *token.FileSet,
	filename string,
	f *ast.File,
	maxParams int,
) []analysis.Diagnostic {
	if maxParams <= 0 ||
		analysis.IsTestFile(filename) ||
		analysis.IsLintSource(filename) {
		return nil
	}

	ctxPkg := analysis.ContextPkgName(f)

	var diags []analysis.Diagnostic
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil || isExemptFunc(fn.Name.Name) {
			continue
		}

		count := countParams(fn.Type.Params, ctxPkg)
		if count > maxParams {
			pos := fset.Position(fn.Pos())
			diags = append(diags, analysis.Diagnostic{
				File: filename,
				Line: pos.Line,
				Message: fmt.Sprintf(
					"function %s has %d parameters"+
						" (limit %d);"+
						" consider an options struct",
					fn.Name.Name, count, maxParams),
			})
		}
	}

	return diags
}

// isExemptFunc reports whether a function name is exempt from the check:
// main, init, Test*, and Benchmark*.
func isExemptFunc(name string) bool {
	return name == "main" || name == "init" ||
		strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark")
}

// countParams returns the number of parameters, excluding context.Context.
// Grouped params (a, b int) are expanded.
func countParams(params *ast.FieldList, ctxPkg string) int {
	count := 0
	for _, field := range params.List {
		if analysis.IsContextType(field.Type, ctxPkg) {
			continue
		}
		names := len(field.Names)
		if names == 0 {
			names = 1
		}
		count += names
	}
	return count
}
