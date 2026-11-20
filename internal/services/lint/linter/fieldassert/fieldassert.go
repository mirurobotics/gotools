package fieldassert

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"path"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// MinFieldAsserts is how many distinct fields asserted on one value trigger a report.
const MinFieldAsserts = 3

// Check reports runs of testify asserts that compare one value field by field.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	if !analysis.IsTestFile(filename) {
		return nil
	}
	testify := testifyNames(file)
	if len(testify) == 0 {
		return nil
	}
	chk := &checker{
		fset:     fset,
		filename: filename,
		f:        file,
		testify:  testify,
		imports:  importNames(file),
		results:  resultFields(file),
		diags:    nil,
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BlockStmt:
			chk.checkStmts(node.List)
		case *ast.CaseClause:
			chk.checkStmts(node.Body)
		case *ast.CommClause:
			chk.checkStmts(node.Body)
		}
		return true
	})
	return chk.diags
}

func testifyNames(file *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range file.Imports {
		if isTestifyPath(analysis.ImportPath(imp)) {
			names[analysis.ImportName(imp)] = true
		}
	}
	return names
}

func isTestifyPath(importPath string) bool {
	const prefix = "github.com/stretchr/testify/"
	return importPath == prefix+"assert" || importPath == prefix+"require"
}

func importNames(file *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range file.Imports {
		names[analysis.ImportName(imp)] = true
		if imp.Name == nil {
			names[path.Base(analysis.ImportPath(imp))] = true
		}
	}
	return names
}

type fieldGroup struct {
	base   string
	root   *ast.Ident
	deps   map[any]bool
	calls  []*ast.CallExpr
	fields map[string]bool
}

type checker struct {
	fset     *token.FileSet
	filename string
	f        *ast.File
	testify  map[string]bool
	imports  map[string]bool
	results  map[*ast.Field]bool
	diags    []analysis.Diagnostic
}

func (c *checker) checkStmts(stmts []ast.Stmt) {
	var groups []*fieldGroup
	open := make(map[string]*fieldGroup)
	for _, stmt := range stmts {
		call, sel := c.fieldArg(stmt)
		if sel == nil {
			c.closeUsed(open, stmt)
			continue
		}
		base := types.ExprString(ast.Unparen(sel.X))
		deps := identSet(sel.X)
		group := open[base]
		if group == nil || !maps.Equal(group.deps, deps) {
			group = &fieldGroup{
				base: base, root: rootIdent(sel.X), deps: deps,
				calls: nil, fields: map[string]bool{},
			}
			open[base] = group
			groups = append(groups, group)
		}
		group.calls = append(group.calls, call)
		group.fields[sel.Sel.Name] = true
	}
	for _, group := range groups {
		if len(group.fields) >= MinFieldAsserts && !c.isParam(group.root) {
			c.report(group)
		}
	}
}

func (c *checker) fieldArg(stmt ast.Stmt) (*ast.CallExpr, *ast.SelectorExpr) {
	call, name, args := c.testifyCall(stmt)
	if call == nil || !isEqualityFunc(name) || len(args) < 2 {
		return nil, nil
	}
	expected, actual := args[0], args[1]
	if sel := c.asField(actual); sel != nil {
		return call, sel
	}
	if !c.isConstant(actual) {
		return nil, nil
	}
	if sel := c.asField(expected); sel != nil {
		return call, sel
	}
	return nil, nil
}

func (c *checker) asField(expr ast.Expr) *ast.SelectorExpr {
	sel, ok := ast.Unparen(expr).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if id, ok := sel.X.(*ast.Ident); ok && c.isPkg(id) {
		return nil
	}
	return sel
}

func (c *checker) isConstant(expr ast.Expr) bool {
	switch inner := ast.Unparen(expr).(type) {
	case *ast.BasicLit:
		return true
	case *ast.CompositeLit:
		return c.allConstant(inner.Elts)
	case *ast.KeyValueExpr:
		return c.isConstant(inner.Value)
	case *ast.UnaryExpr:
		return c.isConstant(inner.X)
	case *ast.Ident:
		return inner.Name == "true" || inner.Name == "false" || inner.Name == "nil"
	case *ast.SelectorExpr:
		id, ok := inner.X.(*ast.Ident)
		return ok && c.isPkg(id)
	}
	return false
}

func (c *checker) allConstant(elts []ast.Expr) bool {
	for _, expr := range elts {
		if !c.isConstant(expr) {
			return false
		}
	}
	return true
}

func (c *checker) report(group *fieldGroup) {
	first := c.fset.Position(group.calls[0].Pos()).Line
	last := c.fset.Position(group.calls[len(group.calls)-1].End()).Line
	if analysis.NolintOnLines(c.fset, c.f, "fieldassert", first, last) {
		return
	}
	msg := fmt.Sprintf(
		"%d asserts compare fields of %s one at a time; compare the whole value in one assert",
		len(group.calls), group.base,
	)
	if group.root != nil {
		msg += ", or move the checks into a helper that takes " + group.root.Name
	}
	c.diags = append(c.diags,
		analysis.Diagnostic{File: c.filename, Line: first, Message: msg})
}

func isEqualityFunc(name string) bool {
	switch strings.TrimSuffix(name, "f") {
	case "Equal", "EqualValues", "Exactly", "EqualExportedValues":
		return true
	}
	return false
}
