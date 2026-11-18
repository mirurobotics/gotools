package funccomment

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "comments are not allowed inside function bodies"

// Check reports each non-directive comment inside a function body.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	bodies := analysis.FuncBodies(f)
	examples := exampleBodies(filename, f)
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if isOutput(cg) && analysis.InAnyBody(cg.Pos(), examples) {
			continue
		}
		for _, c := range cg.List {
			if analysis.IsDirective(c.Text) || !analysis.InAnyBody(c.Pos(), bodies) {
				continue
			}
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(c.Pos()).Line,
				Message: message,
			})
		}
	}
	return diags
}

// exampleBodies returns the bodies of Example functions in a test file.
func exampleBodies(filename string, f *ast.File) []*ast.BlockStmt {
	if !analysis.IsTestFile(filename) {
		return nil
	}
	var bodies []*ast.BlockStmt
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Body != nil && strings.HasPrefix(fn.Name.Name, "Example") {
			bodies = append(bodies, fn.Body)
		}
	}
	return bodies
}

// isOutput reports whether cg is an example output block, matched as go/doc does.
func isOutput(cg *ast.CommentGroup) bool {
	text := strings.ToLower(strings.TrimSpace(cg.Text()))
	return strings.HasPrefix(text, "output:") ||
		strings.HasPrefix(text, "unordered output:")
}
