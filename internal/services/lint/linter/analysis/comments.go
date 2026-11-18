package analysis

import (
	"go/ast"
	"go/token"
	"strings"
)

// DocGroups returns the doc comment groups of declarations, specs, and fields.
func DocGroups(f *ast.File) []*ast.CommentGroup {
	var docs []*ast.CommentGroup
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			docs = append(docs, x.Doc)
		case *ast.GenDecl:
			if x.Tok != token.IMPORT {
				docs = append(docs, x.Doc)
			}
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

// ExampleOutputs returns the output comment groups of Example functions in a test file.
func ExampleOutputs(filename string, f *ast.File) map[*ast.CommentGroup]bool {
	if !IsTestFile(filename) {
		return nil
	}
	var bodies []*ast.BlockStmt
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Body != nil && strings.HasPrefix(fn.Name.Name, "Example") {
			bodies = append(bodies, fn.Body)
		}
	}
	outputs := map[*ast.CommentGroup]bool{}
	for _, cg := range f.Comments {
		if isOutput(cg) && InAnyBody(cg.Pos(), bodies) {
			outputs[cg] = true
		}
	}
	return outputs
}

// isOutput reports whether cg is an example output block, matched as go/doc does.
func isOutput(cg *ast.CommentGroup) bool {
	text := strings.ToLower(strings.TrimSpace(cg.Text()))
	return strings.HasPrefix(text, "output:") ||
		strings.HasPrefix(text, "unordered output:")
}
