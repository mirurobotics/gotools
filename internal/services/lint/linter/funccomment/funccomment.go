package funccomment

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "comments are not allowed inside function bodies"

// Check reports each non-directive comment inside a function body.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	bodies := analysis.FuncBodies(f)
	outputs := analysis.ExampleOutputs(filename, f)
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if outputs[cg] {
			continue
		}
		for _, c := range cg.List {
			if analysis.IsDirective(c.Text) || !analysis.InAnyBody(c.Pos(), bodies) {
				continue
			}
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(c.Pos()).Line,
				Message: message,
			})
		}
	}
	return diags
}
