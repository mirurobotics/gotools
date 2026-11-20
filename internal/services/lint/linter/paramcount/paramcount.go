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
	file *ast.File,
	maxParams int,
) []analysis.Diagnostic {
	if maxParams <= 0 || analysis.IsLintSource(filename) {
		return nil
	}

	ctxPkg := analysis.ContextPkgName(file)

	var diags []analysis.Diagnostic
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Type.Params == nil || isExemptFunc(funcDecl.Name.Name) {
			continue
		}

		count := countParams(funcDecl.Type.Params, ctxPkg)
		if count > maxParams {
			pos := fset.Position(funcDecl.Pos())
			diags = append(diags, analysis.Diagnostic{
				File: filename,
				Line: pos.Line,
				Message: fmt.Sprintf(
					"function %s has %d parameters"+
						" (limit %d);"+
						" consider an options struct",
					funcDecl.Name.Name, count, maxParams),
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
