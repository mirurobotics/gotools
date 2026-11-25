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
	analysis.WalkDeclaredNames(file, chk.check)
	return chk.diags
}

type checker struct {
	fset     *token.FileSet
	filename string
	file     *ast.File
	testing  string
	diags    []analysis.Diagnostic
}

func (c *checker) check(name analysis.DeclaredName) {
	switch name.Kind {
	case analysis.KindParam, analysis.KindResult:
		if !c.isTestingParam(name.Type) {
			c.checkIdent(name.Ident)
		}
	case analysis.KindLocalVar, analysis.KindPackageVar:
		c.checkIdent(name.Ident)
	case analysis.KindField, analysis.KindFunc, analysis.KindTestFunc,
		analysis.KindMethod, analysis.KindIfaceMethod, analysis.KindConst,
		analysis.KindType:
	}
}

func (c *checker) isTestingParam(expr ast.Expr) bool {
	if analysis.IsTestingType(expr, c.testing) || analysis.IsTestingIface(expr) {
		return true
	}
	name, isPtr := analysis.QualifiedTypeName(expr, c.testing)
	return isPtr && name == "PB"
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
