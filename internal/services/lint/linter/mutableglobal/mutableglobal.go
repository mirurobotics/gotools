package mutableglobal

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for mutable package-level variables.
// Allowed exceptions:
//   - error sentinel values: var ErrFoo = errors.New(...)
//   - interface compliance checks: var _ SomeInterface = (*Impl)(nil)
//   - declarations with a //nolint:mutableglobal comment
//
// Test files are exempt.
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

		if hasNolintAtPos(fset, f, fset.Position(gd.TokPos)) {
			continue
		}

		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			if hasNolint(fset, f, vs) {
				continue
			}

			for i, name := range vs.Names {
				if isInterfaceCheck(name) {
					continue
				}

				if i < len(vs.Values) && isErrorSentinel(name, vs.Values[i]) {
					continue
				}

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

// isInterfaceCheck returns true for var _ SomeInterface = ... patterns.
func isInterfaceCheck(name *ast.Ident) bool { return name.Name == "_" }

// isErrVarName returns true when name follows the Go convention for error
// sentinels: it must start with "Err" (exported) or "err" (unexported) and
// be at least 3 characters long (so the bare string "er" is not matched).
func isErrVarName(name string) bool {
	if len(name) < 3 {
		return false
	}
	return strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "err")
}

// isErrorSentinel returns true when varName follows the error-sentinel naming
// convention (starts with "Err" or "err") AND expr is a call to a recognised
// error constructor: any pkg.NewXxx or any pkg.XxxErrorf.
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

// isErrorConstructor returns true for qualified calls that are conventional
// error constructors:
//   - pkg.New(...)       — exact name "New"
//   - pkg.NewFoo(...)    — "New" followed by uppercase letter
//   - pkg.Errorf(...)    — name ends with "Errorf"
//
// This covers errors.New, fmt.Errorf, grpc.Errorf, myerrors.NewNotFound, etc.
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

// hasNolint returns true if the node has a nolint:mutableglobal comment
// on the same line or the line immediately before it.
func hasNolint(fset *token.FileSet, f *ast.File, node ast.Node) bool {
	return hasNolintAtPos(fset, f, fset.Position(node.Pos()))
}

// hasNolintAtPos returns true if there is a nolint:mutableglobal comment
// on the same line or the line immediately before pos.
func hasNolintAtPos(fset *token.FileSet, f *ast.File, pos token.Position) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			cLine := fset.Position(c.Pos()).Line
			if cLine >= pos.Line-1 && cLine <= pos.Line {
				if strings.Contains(c.Text, "nolint:mutableglobal") {
					return true
				}
			}
		}
	}
	return false
}
