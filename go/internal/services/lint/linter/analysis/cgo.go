package analysis

import (
	"go/ast"
	"go/token"
)

// CgoPreambles returns the comment groups that cmd/cgo reads as a cgo preamble.
func CgoPreambles(f *ast.File) map[*ast.CommentGroup]bool {
	m := map[*ast.CommentGroup]bool{}
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			addCgoPreambles(m, gd)
		}
	}
	return m
}

func addCgoPreambles(m map[*ast.CommentGroup]bool, gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		is, ok := spec.(*ast.ImportSpec)
		if !ok || is.Path.Value != `"C"` {
			continue
		}
		if is.Doc != nil {
			m[is.Doc] = true
		} else if len(gd.Specs) == 1 {
			m[gd.Doc] = true
		}
	}
}
