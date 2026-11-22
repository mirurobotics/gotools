package varname

import (
	"fmt"
	"go/ast"
	"go/token"
	"unicode/utf8"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// MinNameLen is the fewest characters a variable name may have.
const MinNameLen = 3

const ruleName = "varname"

// Check reports variables, parameters and named results with too-short names.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	chk := &checker{
		fset:     fset,
		filename: filename,
		file:     file,
		testing:  analysis.TestingPkgName(file),
		diags:    nil,
	}
	ast.Inspect(file, chk.visit)
	return chk.diags
}

type checker struct {
	fset     *token.FileSet
	filename string
	file     *ast.File
	testing  string
	diags    []analysis.Diagnostic
}

func (c *checker) visit(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.FuncDecl:
		c.checkFuncType(node.Type)
	case *ast.FuncLit:
		c.checkFuncType(node.Type)
	case *ast.GenDecl:
		return node.Tok != token.CONST
	case *ast.ValueSpec:
		c.checkIdents(node.Names)
	case *ast.AssignStmt:
		if node.Tok == token.DEFINE {
			c.checkDefined(node, node.Lhs...)
		}
	case *ast.RangeStmt:
		if node.Tok == token.DEFINE {
			c.checkRangeVars(node.Key, node.Value)
		}
	}
	return true
}

func (c *checker) checkFuncType(funcType *ast.FuncType) {
	c.checkFields(funcType.Params)
	c.checkFields(funcType.Results)
}

func (c *checker) checkFields(fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if !c.isTestingParam(field.Type) {
			c.checkIdents(field.Names)
		}
	}
}

func (c *checker) isTestingParam(expr ast.Expr) bool {
	if analysis.IsTestingType(expr, c.testing) || analysis.IsTestingIface(expr) {
		return true
	}
	name, isPtr := analysis.TestingTypeName(expr, c.testing)
	return isPtr && name == "PB"
}

func (c *checker) checkIdents(idents []*ast.Ident) {
	for _, ident := range idents {
		c.checkIdent(ident)
	}
}

func (c *checker) checkDefined(stmt ast.Node, exprs ...ast.Expr) {
	for _, expr := range exprs {
		ident, ok := expr.(*ast.Ident)
		if !ok || redeclared(ident, stmt) {
			continue
		}
		c.checkIdent(ident)
	}
}

func (c *checker) checkRangeVars(exprs ...ast.Expr) {
	for _, expr := range exprs {
		if ident, ok := expr.(*ast.Ident); ok {
			c.checkIdent(ident)
		}
	}
}

func (c *checker) checkIdent(ident *ast.Ident) {
	if c.allowed(ident.Name) || utf8.RuneCountInString(ident.Name) >= MinNameLen {
		return
	}
	line := c.fset.Position(ident.Pos()).Line
	if analysis.NolintOnLines(c.fset, c.file, ruleName, line, line) {
		return
	}
	c.diags = append(c.diags, analysis.Diagnostic{
		File: c.filename,
		Line: line,
		Message: fmt.Sprintf(
			"variable name %q is shorter than %d chars; use a descriptive name",
			ident.Name, MinNameLen,
		),
	})
}

func (c *checker) allowed(name string) bool {
	switch name {
	case "_", "i", "a", "b", "in", "ok", "id", "db", "wg", "mu":
		return true
	case "tc":
		return analysis.IsTestFile(c.filename)
	}
	return false
}

func redeclared(ident *ast.Ident, stmt ast.Node) bool {
	return ident.Obj != nil && ident.Obj.Decl != stmt
}
