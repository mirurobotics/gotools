package typeassert

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// Check reports diagnostics for bare type assertions that don't use the
// comma-ok idiom. Type switch expressions and test files are exempt.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if analysis.IsTestFile(filename) {
		return nil
	}

	// Collect positions of type assertions that are safe (in comma-ok assigns
	// or type switch headers).
	safe := map[token.Pos]bool{}

	ast.Inspect(f, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			// val, ok := x.(T) — 2 LHS values
			if len(s.Lhs) == 2 && len(s.Rhs) == 1 {
				if ta, ok := s.Rhs[0].(*ast.TypeAssertExpr); ok {
					safe[ta.Pos()] = true
				}
			}
		case *ast.TypeSwitchStmt:
			// switch x.(type) — the assign inside is safe
			switch a := s.Assign.(type) {
			case *ast.ExprStmt:
				if ta, ok := a.X.(*ast.TypeAssertExpr); ok {
					safe[ta.Pos()] = true
				}
			case *ast.AssignStmt:
				if len(a.Rhs) == 1 {
					if ta, ok := a.Rhs[0].(*ast.TypeAssertExpr); ok {
						safe[ta.Pos()] = true
					}
				}
			}
		}
		return true
	})

	var diags []analysis.Diagnostic
	ast.Inspect(f, func(n ast.Node) bool {
		ta, ok := n.(*ast.TypeAssertExpr)
		if !ok {
			return true
		}
		if safe[ta.Pos()] {
			return true
		}
		// type switch uses nil Type
		if ta.Type == nil {
			return true
		}
		pos := fset.Position(ta.Pos())
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    pos.Line,
			Message: "bare type assertion may panic; use val, ok := x.(T)",
		})
		return true
	})

	return diags
}
