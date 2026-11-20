package straycomment

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "comment is not attached to a declaration; " +
	"remove it or make it a doc comment"

// Check reports comments outside functions that are not attached to a declaration.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	attached := map[*ast.CommentGroup]bool{file.Doc: true}
	for _, group := range analysis.DocGroups(file) {
		attached[group] = true
	}
	for _, group := range analysis.LineComments(file) {
		attached[group] = true
	}
	bodies := analysis.FuncBodies(file)
	pkgLine := fset.Position(file.Package).Line

	var diags []analysis.Diagnostic
	for _, group := range file.Comments {
		if attached[group] || analysis.InAnyBody(group.Pos(), bodies) {
			continue
		}
		if fset.Position(group.Pos()).Line <= pkgLine {
			continue
		}
		if comment := firstProse(group); comment != nil {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(comment.Pos()).Line,
				Message: message,
			})
		}
	}
	return diags
}

func firstProse(group *ast.CommentGroup) *ast.Comment {
	for _, comment := range group.List {
		if !analysis.IsDirective(comment.Text) {
			return comment
		}
	}
	return nil
}
