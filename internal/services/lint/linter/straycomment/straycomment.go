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
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			attached[gd.Doc] = true
		}
	}
	for _, cg := range analysis.DocGroups(f) {
		attached[cg] = true
	}
	for _, cg := range analysis.LineComments(f) {
		attached[cg] = true
	}
	bodies := analysis.FuncBodies(f)

	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		skip := attached[cg] || cg.Pos() < f.Package ||
			analysis.InAnyBody(cg.Pos(), bodies)
		if skip {
			continue
		}
		for _, c := range cg.List {
			if analysis.IsDirective(c.Text) {
				continue
			}
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(c.Pos()).Line,
				Message: message,
			})
			break
		}
	}
	return diags
}
