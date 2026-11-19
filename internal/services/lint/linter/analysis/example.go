package analysis

import (
	"go/ast"
	"strings"
)

// ExampleOutputs returns the output comment groups of Example functions in a test file.
func ExampleOutputs(filename string, f *ast.File) map[*ast.CommentGroup]bool {
	var bodies []*ast.BlockStmt
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Body != nil && IsExampleFunc(filename, fn) {
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

func isOutput(cg *ast.CommentGroup) bool {
	text := strings.ToLower(strings.TrimSpace(cg.Text()))
	if strings.HasPrefix(text, "output:") {
		return true
	}
	return strings.HasPrefix(text, "unordered output:")
}
