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

// MinFieldAsserts is how many field asserts on one value trigger a report.
const MinFieldAsserts = 3

type checker struct {
	fset     *token.FileSet
	filename string
	f        *ast.File
	testify  map[string]bool
	objects  map[string]bool
	imports  map[string]bool
	diags    []analysis.Diagnostic
}

type fieldGroup struct {
	base  string
	root  string
	calls []*ast.CallExpr
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
		objects:  assertObjects(f, testify),
		imports:  importNames(f),
		diags:    nil,
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Body != nil {
			c.walkFunc(fn.Type, fn.Body)
		}
	}
	return c.diags
}

func (c *checker) walkFunc(ft *ast.FuncType, body *ast.BlockStmt) {
	params := paramNames(ft)
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			c.walkFunc(n.Type, n.Body)
			return false
		case *ast.BlockStmt:
			c.checkStmts(n.List, params)
		case *ast.CaseClause:
			c.checkStmts(n.Body, params)
		case *ast.CommClause:
			c.checkStmts(n.Body, params)
		}
		return true
	})
}

func (c *checker) checkStmts(stmts []ast.Stmt, params map[string]bool) {
	var groups []*fieldGroup
	byBase := make(map[string]*fieldGroup)
	for _, s := range stmts {
		call, sel := c.fieldArg(s)
		if sel == nil {
			continue
		}
		base := types.ExprString(ast.Unparen(sel.X))
		g := byBase[base]
		if g == nil {
			g = &fieldGroup{base: base, root: rootName(sel.X), calls: nil}
			byBase[base] = g
			groups = append(groups, g)
		}
		g.calls = append(g.calls, call)
	}
	for _, g := range groups {
		if len(g.calls) >= MinFieldAsserts && !params[g.root] {
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
	c.diags = append(c.diags, analysis.Diagnostic{
		File: c.filename,
		Line: first,
		Message: fmt.Sprintf(
			"%d asserts compare fields of %s one at a time; compare the whole value"+
				" in one assert, or move the checks into a helper that takes %s",
			len(g.calls), g.base, g.root,
		),
	})
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
	vals := c.comparedValues(call)
	if vals == nil {
		return nil, nil
	}
	if sel := c.asField(vals[1]); sel != nil {
		return call, sel
	}
	if !isLiteral(vals[1]) {
		return nil, nil
	}
	if sel := c.asField(vals[0]); sel != nil {
		return call, sel
	}
	return nil, nil
}

func (c *checker) comparedValues(call *ast.CallExpr) []ast.Expr {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !isEqualityFunc(sel.Sel.Name) {
		return nil
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}
	args := call.Args
	switch {
	case c.testify[recv.Name]:
		if len(args) < 3 {
			return nil
		}
		return args[1:3]
	case c.objects[recv.Name]:
		if len(args) < 2 {
			return nil
		}
		return args[0:2]
	}
	return nil
}

func (c *checker) asField(e ast.Expr) *ast.SelectorExpr {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if id, ok := sel.X.(*ast.Ident); ok && c.imports[id.Name] {
		return nil
	}
	return sel
}

func isLiteral(e ast.Expr) bool {
	switch x := ast.Unparen(e).(type) {
	case *ast.BasicLit:
		return true
	case *ast.UnaryExpr:
		return isLiteral(x.X)
	case *ast.Ident:
		return x.Name == "true" || x.Name == "false" || x.Name == "nil"
	}
	return false
}

func isEqualityFunc(name string) bool {
	switch strings.TrimSuffix(name, "f") {
	case "Equal", "EqualValues", "Exactly", "EqualExportedValues":
		return true
	}
	return false
}

func rootName(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.CallExpr:
			e = x.Fun
		default:
			return ""
		}
	}
}

func paramNames(ft *ast.FuncType) map[string]bool {
	names := make(map[string]bool)
	for _, field := range ft.Params.List {
		for _, n := range field.Names {
			names[n.Name] = true
		}
	}
	return names
}

func testifyNames(f *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range f.Imports {
		if p := importPath(imp); isTestifyPath(p) {
			names[localName(imp, p)] = true
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
		names[localName(imp, importPath(imp))] = true
	}
	return names
}

func importPath(imp *ast.ImportSpec) string {
	return strings.Trim(imp.Path.Value, "\"`")
}

func localName(imp *ast.ImportSpec, p string) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	return path.Base(p)
}

func assertObjects(f *ast.File, testify map[string]bool) map[string]bool {
	names := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		if ok && isNewCall(as.Rhs[0], testify) {
			names[id.Name] = true
		}
		return true
	})
	return names
}

func isNewCall(e ast.Expr, testify map[string]bool) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "New" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && testify[pkg.Name]
}
