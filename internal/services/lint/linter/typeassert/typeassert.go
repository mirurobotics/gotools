package typeassert

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for bare type assertions that don't use the comma-ok idiom.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if analysis.IsTestFile(filename) {
		return nil
	}

	safe := map[token.Pos]bool{}

	ast.Inspect(f, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			if len(s.Lhs) == 2 && len(s.Rhs) == 1 {
				if ta, ok := s.Rhs[0].(*ast.TypeAssertExpr); ok {
					safe[ta.Pos()] = true
				}
			}
		case *ast.TypeSwitchStmt:
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
