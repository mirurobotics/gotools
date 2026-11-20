package fieldassert

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// MinFieldAsserts is how many distinct fields asserted on one value trigger a report.
const MinFieldAsserts = 3

type checker struct {
	fset     *token.FileSet
	filename string
	f        *ast.File
	testify  map[string]bool
	imports  map[string]bool
	results  map[*ast.Field]bool
	diags    []analysis.Diagnostic
}

type fieldGroup struct {
	base   string
	root   *ast.Ident
	deps   map[string]bool
	calls  []*ast.CallExpr
	fields map[string]bool
}

// Check reports runs of testify asserts that compare one value field by field.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if !analysis.IsTestFile(filename) {
		return nil
	}
	testify := testifyNames(f)
	if len(testify) == 0 {
		return nil
	}
	c := &checker{
		fset:     fset,
		filename: filename,
		f:        f,
		testify:  testify,
		imports:  importNames(f),
		results:  resultFields(f),
		diags:    nil,
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BlockStmt:
			c.checkStmts(n.List)
		case *ast.CaseClause:
			c.checkStmts(n.Body)
		case *ast.CommClause:
			c.checkStmts(n.Body)
		}
		return true
	})
	return c.diags
}

func (c *checker) checkStmts(stmts []ast.Stmt) {
	var groups []*fieldGroup
	open := make(map[string]*fieldGroup)
	for _, s := range stmts {
		call, sel := c.fieldArg(s)
		if sel == nil {
			closeAssigned(open, s)
			continue
		}
		base := types.ExprString(ast.Unparen(sel.X))
		g := open[base]
		if g == nil {
			g = &fieldGroup{
				base: base, root: rootIdent(sel.X), deps: identSet(sel.X),
				calls: nil, fields: map[string]bool{},
			}
			open[base] = g
			groups = append(groups, g)
		}
		g.calls = append(g.calls, call)
		g.fields[sel.Sel.Name] = true
	}
	for _, g := range groups {
		if len(g.fields) >= MinFieldAsserts && !c.isParam(g.root) {
			c.report(g)
		}
	}
}

func (c *checker) report(g *fieldGroup) {
	first := c.fset.Position(g.calls[0].Pos()).Line
	last := c.fset.Position(g.calls[len(g.calls)-1].End()).Line
	if analysis.NolintOnLines(c.fset, c.f, "fieldassert", first, last) {
		return
	}
	msg := fmt.Sprintf(
		"%d asserts compare fields of %s one at a time; compare the whole value in one assert",
		len(g.calls), g.base,
	)
	if g.root != nil {
		msg += ", or move the checks into a helper that takes " + g.root.Name
	}
	c.diags = append(c.diags,
		analysis.Diagnostic{File: c.filename, Line: first, Message: msg})
}

func (c *checker) fieldArg(s ast.Stmt) (*ast.CallExpr, *ast.SelectorExpr) {
	es, ok := s.(*ast.ExprStmt)
	if !ok {
		return nil, nil
	}
	call, ok := es.X.(*ast.CallExpr)
	if !ok {
		return nil, nil
	}
	expected, actual, ok := c.comparedValues(call)
	if !ok {
		return nil, nil
	}
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

func (c *checker) comparedValues(
	call *ast.CallExpr,
) (expected, actual ast.Expr, ok bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !isEqualityFunc(sel.Sel.Name) {
		return nil, nil, false
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, nil, false
	}
	first := 0
	switch {
	case c.isAssertObject(recv):
	case recv.Obj == nil && c.testify[recv.Name]:
		first = 1
	default:
		return nil, nil, false
	}
	if len(call.Args) < first+2 {
		return nil, nil, false
	}
	return call.Args[first], call.Args[first+1], true
}

func (c *checker) asField(e ast.Expr) *ast.SelectorExpr {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if id, ok := sel.X.(*ast.Ident); ok && c.isPkg(id) {
		return nil
	}
	return sel
}

func (c *checker) isConstant(e ast.Expr) bool {
	switch x := ast.Unparen(e).(type) {
	case *ast.BasicLit:
		return true
	case *ast.CompositeLit:
		return c.allConstant(x.Elts)
	case *ast.KeyValueExpr:
		return c.isConstant(x.Value)
	case *ast.UnaryExpr:
		return c.isConstant(x.X)
	case *ast.Ident:
		return x.Name == "true" || x.Name == "false" || x.Name == "nil"
	case *ast.SelectorExpr:
		id, ok := x.X.(*ast.Ident)
		return ok && c.isPkg(id)
	}
	return false
}

func (c *checker) allConstant(elts []ast.Expr) bool {
	for _, e := range elts {
		if !c.isConstant(e) {
			return false
		}
	}
	return true
}

func isEqualityFunc(name string) bool {
	switch strings.TrimSuffix(name, "f") {
	case "Equal", "EqualValues", "Exactly", "EqualExportedValues":
		return true
	}
	return false
}

func testifyNames(f *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range f.Imports {
		if isTestifyPath(analysis.ImportPath(imp)) {
			names[analysis.ImportName(imp)] = true
		}
	}
	return names
}

func isTestifyPath(p string) bool {
	const prefix = "github.com/stretchr/testify/"
	return p == prefix+"assert" || p == prefix+"require"
}

func importNames(f *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range f.Imports {
		names[analysis.ImportName(imp)] = true
		names[path.Base(analysis.ImportPath(imp))] = true
	}
	return names
}
