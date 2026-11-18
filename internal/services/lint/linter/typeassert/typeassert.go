package typeassert

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for bare type assertions that don't use the
// comma-ok idiom. Type switch expressions and test files are exempt.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if analysis.IsTestFile(filename) {
		return nil
	}

	safe := safeAssertions(f)

	var diags []analysis.Diagnostic
	ast.Inspect(f, func(n ast.Node) bool {
		ta, ok := n.(*ast.TypeAssertExpr)
		if !ok {
			return true
		}
		// comma-ok/type-switch assign, or x.(type) (nil Type)
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

// safeAssertions collects positions of type assertions that are safe (in
// comma-ok assigns or type switch headers).
func safeAssertions(f *ast.File) map[token.Pos]bool {
	safe := map[token.Pos]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			// val, ok := x.(T) — 2 LHS values
			if len(s.Lhs) == 2 {
				markSingleAssert(safe, s.Rhs)
			}
		case *ast.TypeSwitchStmt:
			// switch x.(type) — the assign inside is safe
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

// markSingleAssert marks rhs[0] as safe when rhs is a single type assertion.
func markSingleAssert(safe map[token.Pos]bool, rhs []ast.Expr) {
	if len(rhs) != 1 {
		return
	}
	if ta, ok := rhs[0].(*ast.TypeAssertExpr); ok {
		safe[ta.Pos()] = true
	}
}
