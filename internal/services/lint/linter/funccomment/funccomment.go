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
	var bodies []*ast.BlockStmt
	exempt := map[*ast.CommentGroup]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body == nil {
				return true
			}
			bodies = append(bodies, fn.Body)
			if isExample(filename, fn) {
				markOutput(f, fn.Body, exempt)
			}
		case *ast.FuncLit:
			bodies = append(bodies, fn.Body)
		}
		return true
	})

	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if exempt[cg] {
			continue
		}
		for _, c := range cg.List {
			if analysis.IsDirective(c.Text) || !inAny(c.Pos(), bodies) {
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

// isExample reports whether fn is an Example function in a test file.
func isExample(filename string, fn *ast.FuncDecl) bool {
	return analysis.IsTestFile(filename) && strings.HasPrefix(fn.Name.Name, "Example")
}

// markOutput exempts the example output comment groups in body.
func markOutput(f *ast.File, body *ast.BlockStmt, exempt map[*ast.CommentGroup]bool) {
	for _, cg := range f.Comments {
		if cg.Pos() <= body.Lbrace || cg.Pos() >= body.Rbrace {
			continue
		}
		text := strings.TrimSpace(cg.Text())
		if strings.HasPrefix(text, "Output:") ||
			strings.HasPrefix(text, "Unordered output:") {
			exempt[cg] = true
		}
	}
}

func inAny(pos token.Pos, bodies []*ast.BlockStmt) bool {
	for _, b := range bodies {
		if pos > b.Lbrace && pos < b.Rbrace {
			return true
		}
	}
	return false
}
