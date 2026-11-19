package straycomment

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "comment is not attached to a declaration; " +
	"remove it or make it a doc comment"

// Check reports comments outside functions that are not attached to a declaration.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	attached := map[*ast.CommentGroup]bool{f.Doc: true}
	for _, cg := range analysis.DocGroups(f) {
		attached[cg] = true
	}
	for _, cg := range analysis.LineComments(f) {
		attached[cg] = true
	}
	bodies := analysis.FuncBodies(f)
	pkgLine := fset.Position(f.Package).Line

	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if attached[cg] || analysis.InAnyBody(cg.Pos(), bodies) {
			continue
		}
		if fset.Position(cg.Pos()).Line <= pkgLine {
			continue
		}
		if c := firstProse(cg); c != nil {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(c.Pos()).Line,
				Message: message,
			})
		}
	}
	return diags
}

func firstProse(cg *ast.CommentGroup) *ast.Comment {
	for _, c := range cg.List {
		if !analysis.IsDirective(c.Text) {
			return c
		}
	}
	return nil
}
