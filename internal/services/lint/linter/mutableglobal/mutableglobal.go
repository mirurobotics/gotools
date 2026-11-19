package mutableglobal

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for mutable package-level variables.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if analysis.IsTestFile(filename) {
		return nil
	}

	var diags []analysis.Diagnostic

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}

		if hasNolint(fset, f, gd.TokPos) {
			continue
		}

		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || hasNolint(fset, f, vs.Pos()) {
				continue
			}
			for _, name := range mutableNames(vs) {
				pos := fset.Position(name.Pos())
				diags = append(diags, analysis.Diagnostic{
					File:    filename,
					Line:    pos.Line,
					Message: "mutable global variable: use const or a function instead",
				})
			}
		}
	}

	return diags
}

func mutableNames(vs *ast.ValueSpec) []*ast.Ident {
	var names []*ast.Ident
	for i, name := range vs.Names {
		if isInterfaceCheck(name) {
			continue
		}
		if i < len(vs.Values) && isErrorSentinel(name, vs.Values[i]) {
			continue
		}
		names = append(names, name)
	}
	return names
}

func isInterfaceCheck(name *ast.Ident) bool { return name.Name == "_" }

func isErrVarName(name string) bool {
	if len(name) < 3 {
		return false
	}
	return strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "err")
}

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

func isErrorConstructor(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	_, ok = sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	fn := sel.Sel.Name
	if fn == "New" || (len(fn) > 3 && fn[:3] == "New" && fn[3] >= 'A' && fn[3] <= 'Z') {
		return true
	}
	if strings.HasSuffix(fn, "Errorf") {
		return true
	}
	return false
}

func hasNolint(fset *token.FileSet, f *ast.File, pos token.Pos) bool {
	line := fset.Position(pos).Line
	return analysis.NolintOnLines(fset, f, "mutableglobal", line-1, line)
}
