package typeassert

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for bare type assertions that don't use the comma-ok idiom.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	safe := safeAssertions(file)

	var diags []analysis.Diagnostic
	ast.Inspect(file, func(node ast.Node) bool {
		assert, ok := node.(*ast.TypeAssertExpr)
		if !ok {
			return true
		}
		if safe[assert.Pos()] || assert.Type == nil {
			return true
		}
		pos := fset.Position(assert.Pos())
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    pos.Line,
			Message: "bare type assertion may panic; use val, ok := x.(T)",
		})
		return true
	})

	return diags
}

func safeAssertions(file *ast.File) map[token.Pos]bool {
	safe := map[token.Pos]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.AssignStmt:
			if len(node.Lhs) == 2 {
				markSingleAssert(safe, node.Rhs)
			}
		case *ast.TypeSwitchStmt:
			switch a := node.Assign.(type) {
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
	if assert, ok := rhs[0].(*ast.TypeAssertExpr); ok {
		safe[assert.Pos()] = true
	}
}
