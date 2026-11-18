package analysis

import (
	"go/ast"
	"go/token"
)

// DocGroups returns the doc comment groups of declarations, specs, and fields.
func DocGroups(f *ast.File) []*ast.CommentGroup {
	var docs []*ast.CommentGroup
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			docs = append(docs, x.Doc)
		case *ast.GenDecl:
			docs = append(docs, x.Doc)
		case *ast.TypeSpec:
			docs = append(docs, x.Doc)
		case *ast.ValueSpec:
			docs = append(docs, x.Doc)
		case *ast.ImportSpec:
			docs = append(docs, x.Doc)
		case *ast.Field:
			docs = append(docs, x.Doc)
		}
		return true
	})
	return docs
}

// LineComments returns the trailing line comment groups of specs and fields.
func LineComments(f *ast.File) []*ast.CommentGroup {
	var groups []*ast.CommentGroup
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			groups = append(groups, x.Comment)
		case *ast.ValueSpec:
			groups = append(groups, x.Comment)
		case *ast.ImportSpec:
			groups = append(groups, x.Comment)
		case *ast.Field:
			groups = append(groups, x.Comment)
		}
		return true
	})
	return groups
}

// FuncBodies returns the bodies of all function declarations and literals.
func FuncBodies(f *ast.File) []*ast.BlockStmt {
	var bodies []*ast.BlockStmt
	ast.Inspect(f, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body != nil {
				bodies = append(bodies, fn.Body)
			}
		case *ast.FuncLit:
			bodies = append(bodies, fn.Body)
		}
		return true
	})
	return bodies
}

// InAnyBody reports whether pos lies strictly inside one of bodies.
func InAnyBody(pos token.Pos, bodies []*ast.BlockStmt) bool {
	for _, b := range bodies {
		if pos > b.Lbrace && pos < b.Rbrace {
			return true
		}
	}
	return false
}
