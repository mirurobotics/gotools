package typeassert

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for bare type assertions that don't use the comma-ok idiom.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	safe := safeAssertions(f)

	var diags []analysis.Diagnostic
	ast.Inspect(f, func(n ast.Node) bool {
		ta, ok := n.(*ast.TypeAssertExpr)
		if !ok {
			return true
		}
		if safe[ta.Pos()] || ta.Type == nil {
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

func safeAssertions(f *ast.File) map[token.Pos]bool {
	safe := map[token.Pos]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			if len(s.Lhs) == 2 {
				markSingleAssert(safe, s.Rhs)
			}
		case *ast.TypeSwitchStmt:
			switch a := s.Assign.(type) {
			case *ast.ExprStmt:
				markSingleAssert(safe, []ast.Expr{a.X})
			case *ast.AssignStmt:
				markSingleAssert(safe, a.Rhs)
			}
		}
		return true
	})
	return safe
}

func markSingleAssert(safe map[token.Pos]bool, rhs []ast.Expr) {
	if len(rhs) != 1 {
		return
	}
	if ta, ok := rhs[0].(*ast.TypeAssertExpr); ok {
		safe[ta.Pos()] = true
	}
}
