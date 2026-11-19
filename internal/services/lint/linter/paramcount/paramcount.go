package paramcount

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports functions with more than maxParams non-context parameters.
func Check(
	fset *token.FileSet,
	filename string,
	f *ast.File,
	maxParams int,
) []analysis.Diagnostic {
	if maxParams <= 0 {
		return nil
	}
	if analysis.IsTestFile(filename) || analysis.IsLintSource(filename) {
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

func isExemptFunc(name string) bool {
	if name == "main" || name == "init" {
		return true
	}
	return strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark")
}

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
