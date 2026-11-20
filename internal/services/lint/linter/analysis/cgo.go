package analysis

import (
	"go/ast"
	"go/token"
)

// CgoPreambles returns the comment groups that cmd/cgo reads as a cgo preamble.
func CgoPreambles(file *ast.File) map[*ast.CommentGroup]bool {
	preambles := map[*ast.CommentGroup]bool{}
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			addCgoPreambles(preambles, genDecl)
		}
	}
	return preambles
}

func addCgoPreambles(preambles map[*ast.CommentGroup]bool, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		imp, ok := spec.(*ast.ImportSpec)
		if !ok || imp.Path.Value != `"C"` {
			continue
		}
		if imp.Doc != nil {
			preambles[imp.Doc] = true
		} else if len(decl.Specs) == 1 {
			preambles[decl.Doc] = true
		}
	}
}
