package nestdepth

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports a diagnostic when nesting depth inside a function exceeds maxDepth.
func Check(
	fset *token.FileSet,
	filename string,
	file *ast.File,
	maxDepth int,
) []analysis.Diagnostic {
	if maxDepth <= 0 {
		return nil
	}
	chk := &checker{fset: fset, filename: filename, maxDepth: maxDepth, diags: nil}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}
		chk.checkBlock(funcDecl.Body.List, 0)
	}

	ast.Inspect(file, func(node ast.Node) bool {
		lit, ok := node.(*ast.FuncLit)
		if !ok || lit.Body == nil {
			return true
		}
		chk.checkBlock(lit.Body.List, 0)
		return true
	})

	return chk.diags
}

type checker struct {
	fset     *token.FileSet
	filename string
	maxDepth int
	diags    []analysis.Diagnostic
}

func (c *checker) checkBlock(stmts []ast.Stmt, depth int) {
	for _, stmt := range stmts {
		c.checkStmt(stmt, depth)
	}
}

func (c *checker) checkStmt(stmt ast.Stmt, depth int) {
	if body := nestingBody(stmt); body != nil {
		if nested, ok := c.deeper(stmt.Pos(), depth); ok {
			c.checkBlock(body.List, nested)
		}
	}
	switch stmt := stmt.(type) {
	case *ast.IfStmt:
		if stmt.Else != nil {
			c.checkStmt(stmt.Else, depth)
		}
	case *ast.CaseClause:
		c.checkBlock(stmt.Body, depth)
	case *ast.CommClause:
		c.checkBlock(stmt.Body, depth)
	case *ast.BlockStmt:
		c.checkBlock(stmt.List, depth)
	case *ast.LabeledStmt:
		c.checkStmt(stmt.Stmt, depth)
	}
}

func (c *checker) deeper(pos token.Pos, depth int) (int, bool) {
	nested := depth + 1
	if nested > c.maxDepth {
		c.report(pos, nested)
		return nested, false
	}
	return nested, true
}

func (c *checker) report(pos token.Pos, depth int) {
	position := c.fset.Position(pos)
	msg := fmt.Sprintf(
		"nesting depth is %d (limit %d)"+
			" — consider early returns or extracting a helper",
		depth, c.maxDepth,
	)
	c.diags = append(c.diags, analysis.Diagnostic{
		File:    c.filename,
		Line:    position.Line,
		Message: msg,
	})
}

func nestingBody(stmt ast.Stmt) *ast.BlockStmt {
	switch stmt := stmt.(type) {
	case *ast.IfStmt:
		return stmt.Body
	case *ast.ForStmt:
		return stmt.Body
	case *ast.RangeStmt:
		return stmt.Body
	case *ast.SwitchStmt:
		return stmt.Body
	case *ast.TypeSwitchStmt:
		return stmt.Body
	case *ast.SelectStmt:
		return stmt.Body
	}
	return nil
}
