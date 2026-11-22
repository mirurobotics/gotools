package namelen

import (
	"fmt"
	"go/ast"
	"go/token"
	"unicode/utf8"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// MaxNameLen is the most characters a declared name may have.
const MaxNameLen = 30

// MaxLocalNameLen is the most characters a local variable or parameter name may have.
const MaxLocalNameLen = 20

// MaxTestNameLen is the most characters a test function name may have.
const MaxTestNameLen = 50

const ruleName = "namelen"

// Check reports declared names longer than their limit.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	chk := &checker{fset: fset, filename: filename, file: file, diags: nil}
	analysis.WalkDeclaredNames(file, chk.check)
	return chk.diags
}

type checker struct {
	fset     *token.FileSet
	filename string
	file     *ast.File
	diags    []analysis.Diagnostic
}

func (c *checker) check(name analysis.DeclaredName) {
	limit := nameLimit(name.Kind)
	length := utf8.RuneCountInString(name.Ident.Name)
	if length <= limit {
		return
	}
	line := c.fset.Position(name.Ident.Pos()).Line
	if analysis.NolintOnLines(c.fset, c.file, ruleName, line, line) {
		return
	}
	c.diags = append(c.diags, analysis.Diagnostic{
		File: c.filename,
		Line: line,
		Message: fmt.Sprintf(
			"name %q is %d chars (limit %d)", name.Ident.Name, length, limit,
		),
	})
}

func nameLimit(kind analysis.NameKind) int {
	switch kind {
	case analysis.KindLocalVar, analysis.KindParam, analysis.KindResult:
		return MaxLocalNameLen
	case analysis.KindTestFunc:
		return MaxTestNameLen
	}
	return MaxNameLen
}
