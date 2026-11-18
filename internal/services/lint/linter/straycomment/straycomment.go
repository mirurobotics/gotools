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
	attached := attachedGroups(f)
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if attached[cg] || cg.Pos() < f.Package || inFunc(f, cg.Pos()) {
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

func attachedGroups(f *ast.File) map[*ast.CommentGroup]bool {
	m := map[*ast.CommentGroup]bool{f.Doc: true}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			m[x.Doc] = true
		case *ast.GenDecl:
			m[x.Doc] = true
		case *ast.TypeSpec:
			m[x.Doc], m[x.Comment] = true, true
		case *ast.ValueSpec:
			m[x.Doc], m[x.Comment] = true, true
		case *ast.ImportSpec:
			m[x.Doc], m[x.Comment] = true, true
		case *ast.Field:
			m[x.Doc], m[x.Comment] = true, true
		}
		return true
	})
	return m
}

func inFunc(f *ast.File, pos token.Pos) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		if found || n == nil || pos < n.Pos() || pos >= n.End() {
			return false
		}
		switch fn := n.(type) {
		case *ast.FuncDecl:
			found = fn.Body != nil && pos > fn.Body.Lbrace
		case *ast.FuncLit:
			found = pos > fn.Body.Lbrace
		}
		return !found
	})
	return found
}
