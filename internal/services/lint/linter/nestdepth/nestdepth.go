package nestdepth

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// checker holds shared state for a single nest-depth analysis pass.
type checker struct {
	fset     *token.FileSet
	filename string
	maxDepth int
	diags    []analysis.Diagnostic
}

// Check reports a diagnostic when nesting depth inside a function
// exceeds maxDepth. Nesting constructs: if, for, range, switch,
// type switch, select. Case/default clause bodies do not add a
// nesting level.
func Check(
	fset *token.FileSet,
	filename string,
	f *ast.File,
	maxDepth int,
) []analysis.Diagnostic {
	if maxDepth <= 0 {
		return nil
	}
	c := &checker{fset: fset, filename: filename, maxDepth: maxDepth, diags: nil}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		c.checkBlock(fn.Body.List, 0)
	}

	// Also check function literals.
	ast.Inspect(f, func(n ast.Node) bool {
		fl, ok := n.(*ast.FuncLit)
		if !ok || fl.Body == nil {
			return true
		}
		c.checkBlock(fl.Body.List, 0)
		return true
	})

	return c.diags
}

func (c *checker) checkBlock(stmts []ast.Stmt, depth int) {
	for _, stmt := range stmts {
		c.checkStmt(stmt, depth)
	}
}

// deeper checks whether entering a nesting construct would exceed the
// limit. Returns true if it is safe to recurse at depth+1.
func (c *checker) deeper(pos token.Pos, depth int) (int, bool) {
	d := depth + 1
	if d > c.maxDepth {
		c.report(pos, d)
		return d, false
	}
	return d, true
}

func (c *checker) checkStmt(stmt ast.Stmt, depth int) {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		if d, ok := c.deeper(s.Pos(), depth); ok {
			c.checkBlock(s.Body.List, d)
		}
		if s.Else != nil {
			c.checkStmt(s.Else, depth)
		}
	case *ast.ForStmt:
		if d, ok := c.deeper(s.Pos(), depth); ok {
			c.checkBlock(s.Body.List, d)
		}
	case *ast.RangeStmt:
		if d, ok := c.deeper(s.Pos(), depth); ok {
			c.checkBlock(s.Body.List, d)
		}
	case *ast.SwitchStmt:
		if d, ok := c.deeper(s.Pos(), depth); ok {
			c.checkBlock(s.Body.List, d)
		}
	case *ast.TypeSwitchStmt:
		if d, ok := c.deeper(s.Pos(), depth); ok {
			c.checkBlock(s.Body.List, d)
		}
	case *ast.SelectStmt:
		if d, ok := c.deeper(s.Pos(), depth); ok {
			c.checkBlock(s.Body.List, d)
		}
	case *ast.CaseClause:
		c.checkBlock(s.Body, depth)
	case *ast.CommClause:
		c.checkBlock(s.Body, depth)
	case *ast.BlockStmt:
		c.checkBlock(s.List, depth)
	case *ast.LabeledStmt:
		c.checkStmt(s.Stmt, depth)
	}
}

func (c *checker) report(pos token.Pos, depth int) {
	p := c.fset.Position(pos)
	msg := fmt.Sprintf(
		"nesting depth is %d (limit %d)"+
			" — consider early returns or extracting a helper",
		depth, c.maxDepth,
	)
	c.diags = append(c.diags, analysis.Diagnostic{
		File:    c.filename,
		Line:    p.Line,
		Message: msg,
	})
}
