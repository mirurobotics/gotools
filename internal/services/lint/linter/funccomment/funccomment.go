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
	exempt := exampleOutputs(filename, f)
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if exempt[cg] {
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

// exampleOutputs returns the Output comment groups of Example functions in tests.
func exampleOutputs(filename string, f *ast.File) map[*ast.CommentGroup]bool {
	exempt := map[*ast.CommentGroup]bool{}
	if !analysis.IsTestFile(filename) {
		return exempt
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Example") {
			continue
		}
		for _, cg := range f.Comments {
			if !analysis.InAnyBody(cg.Pos(), []*ast.BlockStmt{fn.Body}) {
				continue
			}
			text := strings.TrimSpace(cg.Text())
			if strings.HasPrefix(text, "Output:") ||
				strings.HasPrefix(text, "Unordered output:") {
				exempt[cg] = true
			}
		}
	}
	return exempt
}
