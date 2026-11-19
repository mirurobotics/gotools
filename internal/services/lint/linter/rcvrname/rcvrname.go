package rcvrname

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports receiver names that are over 2 chars or inconsistent per type.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic

	firstRecv := map[string]string{}

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
		prev, exists := firstRecv[typeName]
		if !exists {
			firstRecv[typeName] = recvName
		}

		line := fset.Position(recv.Names[0].Pos()).Line
		for _, msg := range checkRecvName(recvName, typeName, prev) {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    line,
				Message: msg,
			})
		}
	}

	return diags
}

func checkRecvName(recvName, typeName, prev string) []string {
	var msgs []string
	if len(recvName) > 2 {
		msgs = append(msgs, fmt.Sprintf(
			"receiver name %q is %d chars; use 1-2 chars (e.g., %q)",
			recvName, len(recvName), string(recvName[0]),
		))
	}
	if prev != "" && prev != recvName {
		msgs = append(msgs, fmt.Sprintf(
			"receiver name %q for type %s differs from %q used elsewhere in this file",
			recvName, typeName, prev,
		))
	}
	return msgs
}

func receiverTypeName(expr ast.Expr) string {
	if id := analysis.BaseIdent(expr); id != nil {
		return id.Name
	}
	return ""
}
