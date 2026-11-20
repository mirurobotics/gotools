package mutableglobal

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for mutable package-level variables.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		if hasNolint(fset, file, genDecl.TokPos) {
			continue
		}

		for _, name := range declMutableNames(fset, file, genDecl) {
			pos := fset.Position(name.Pos())
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    pos.Line,
				Message: "mutable global variable: use const or a function instead",
			})
		}
	}

	return diags
}

func declMutableNames(
	fset *token.FileSet, file *ast.File, decl *ast.GenDecl,
) []*ast.Ident {
	var names []*ast.Ident
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || hasNolint(fset, file, valueSpec.Pos()) {
			continue
		}
		names = append(names, mutableNames(valueSpec)...)
	}
	return names
}

func hasNolint(fset *token.FileSet, file *ast.File, pos token.Pos) bool {
	line := fset.Position(pos).Line
	return analysis.NolintOnLines(fset, file, "mutableglobal", line-1, line)
}

func mutableNames(spec *ast.ValueSpec) []*ast.Ident {
	var names []*ast.Ident
	for i, name := range spec.Names {
		if isInterfaceCheck(name) {
			continue
		}
		if i < len(spec.Values) && isErrorSentinel(name, spec.Values[i]) {
			continue
		}
		names = append(names, name)
	}
	return names
}

func isInterfaceCheck(name *ast.Ident) bool { return name.Name == "_" }

func isErrorSentinel(varName *ast.Ident, expr ast.Expr) bool {
	if !isErrVarName(varName.Name) {
		return false
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isErrorConstructor(call)
}

func isErrVarName(name string) bool {
	if len(name) < 3 {
		return false
	}
	return strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "err")
}

func isErrorConstructor(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	_, ok = sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	funcName := sel.Sel.Name
	rest, isNew := strings.CutPrefix(funcName, "New")
	if isNew && (rest == "" || rest[0] >= 'A' && rest[0] <= 'Z') {
		return true
	}
	if strings.HasSuffix(funcName, "Errorf") {
		return true
	}
	return false
}
