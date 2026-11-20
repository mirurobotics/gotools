package funccomment

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "comments are not allowed inside function bodies"

// Check reports each non-directive comment inside a function body.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	bodies := analysis.FuncBodies(file)
	outputs := analysis.ExampleOutputs(filename, file)
	var diags []analysis.Diagnostic
	for _, group := range file.Comments {
		if outputs[group] {
			continue
		}
		for _, comment := range bodyComments(group, bodies) {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(comment.Pos()).Line,
				Message: message,
			})
		}
	}
	return diags
}

func bodyComments(group *ast.CommentGroup, bodies []*ast.BlockStmt) []*ast.Comment {
	var out []*ast.Comment
	for _, comment := range group.List {
		if analysis.IsDirective(comment.Text) {
			continue
		}
		if analysis.InAnyBody(comment.Pos(), bodies) {
			out = append(out, comment)
		}
	}
	return out
}
