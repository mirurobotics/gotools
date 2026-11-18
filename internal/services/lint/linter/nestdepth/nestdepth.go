package nestdepth

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

type checker struct {
	fset     *token.FileSet
	filename string
	maxDepth int
	diags    []analysis.Diagnostic
}

// Check reports a diagnostic when nesting depth inside a function exceeds maxDepth.
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

func (c *checker) deeper(pos token.Pos, depth int) (int, bool) {
	d := depth + 1
	if d > c.maxDepth {
		c.report(pos, d)
		return d, false
	}
	return d, true
}

func (c *checker) checkStmt(stmt ast.Stmt, depth int) {
	if body := nestingBody(stmt); body != nil {
		if d, ok := c.deeper(stmt.Pos(), depth); ok {
			c.checkBlock(body.List, d)
		}
	}
	switch s := stmt.(type) {
	case *ast.IfStmt:
		if s.Else != nil {
			c.checkStmt(s.Else, depth)
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

func nestingBody(stmt ast.Stmt) *ast.BlockStmt {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		return s.Body
	case *ast.ForStmt:
		return s.Body
	case *ast.RangeStmt:
		return s.Body
	case *ast.SwitchStmt:
		return s.Body
	case *ast.TypeSwitchStmt:
		return s.Body
	case *ast.SelectStmt:
		return s.Body
	}
	return nil
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
