package analysis

import (
	"go/ast"
	"strings"
)

// ExampleOutputs returns the output comment groups of Example functions in a test file.
func ExampleOutputs(filename string, file *ast.File) map[*ast.CommentGroup]bool {
	var bodies []*ast.BlockStmt
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && funcDecl.Body != nil && IsExampleFunc(filename, funcDecl) {
			bodies = append(bodies, funcDecl.Body)
		}
	}
	outputs := map[*ast.CommentGroup]bool{}
	for _, group := range file.Comments {
		if isOutput(group) && InAnyBody(group.Pos(), bodies) {
			outputs[group] = true
		}
	}
	return outputs
}

func isOutput(group *ast.CommentGroup) bool {
	text := strings.ToLower(strings.TrimSpace(group.Text()))
	if strings.HasPrefix(text, "output:") {
		return true
	}
	return strings.HasPrefix(text, "unordered output:")
}
