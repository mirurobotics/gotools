package rcvrname

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for receiver names that are longer than 2
// characters or inconsistent across methods of the same type within a file.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic

	// Track first receiver name seen per type for consistency checks.
	firstRecv := map[string]string{} // typeName → first receiver name

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}

		recv := fn.Recv.List[0]
		if len(recv.Names) == 0 {
			continue
		}

		recvName := recv.Names[0].Name
		typeName := receiverTypeName(recv.Type)

		// Check length.
		if len(recvName) > 2 {
			pos := fset.Position(recv.Names[0].Pos())
			diags = append(diags, analysis.Diagnostic{
				File: filename,
				Line: pos.Line,
				Message: fmt.Sprintf(
					"receiver name %q is %d chars; use 1-2 chars (e.g., %q)",
					recvName, len(recvName), string(recvName[0]),
				),
			})
		}

		// Check consistency.
		if prev, exists := firstRecv[typeName]; exists {
			if prev != recvName {
				pos := fset.Position(recv.Names[0].Pos())
				diags = append(diags, analysis.Diagnostic{
					File: filename,
					Line: pos.Line,
					Message: fmt.Sprintf(
						"receiver name %q for type %s differs from %q used elsewhere in this file",
						recvName, typeName, prev,
					),
				})
			}
		} else {
			firstRecv[typeName] = recvName
		}
	}

	return diags
}

// receiverTypeName extracts the type name from a receiver's type expression,
// handling both value receivers (T) and pointer receivers (*T).
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.IndexListExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}
