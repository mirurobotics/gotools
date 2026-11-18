package funclen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports functions whose body exceeds maxLen non-blank, non-comment lines.
func Check(
	fset *token.FileSet,
	filename string,
	f *ast.File,
	src []byte,
	maxLen int,
) []analysis.Diagnostic {
	if maxLen <= 0 {
		return nil
	}
	if analysis.IsTestFile(filename) || analysis.IsTestHelper(filename) {
		return nil
	}
	allLines := bytes.Split(src, []byte("\n"))

	var diags []analysis.Diagnostic
	ast.Inspect(f, func(n ast.Node) bool {
		body, label, ok := funcBody(n)
		if !ok {
			return true
		}
		count := countLines(fset, body, allLines)
		if count <= maxLen {
			return true
		}
		pos := fset.Position(n.Pos())
		if analysis.NolintOnLines(fset, f, "funclen", pos.Line-1, pos.Line) {
			return true
		}
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    pos.Line,
			Message: fmt.Sprintf("%s is %d lines (limit %d)", label, count, maxLen),
		})
		return true
	})
	return diags
}

func funcBody(n ast.Node) (*ast.BlockStmt, string, bool) {
	switch fn := n.(type) {
	case *ast.FuncDecl:
		if fn.Body == nil {
			return nil, "", false
		}
		return fn.Body, "function " + fn.Name.Name, true
	case *ast.FuncLit:
		if fn.Body == nil {
			return nil, "", false
		}
		return fn.Body, "anonymous function", true
	}
	return nil, "", false
}

func countLines(fset *token.FileSet, body *ast.BlockStmt, allLines [][]byte) int {
	startLine := fset.Position(body.Lbrace).Line
	endLine := fset.Position(body.Rbrace).Line

	count := 0
	for lineNum := startLine + 1; lineNum < endLine; lineNum++ {
		idx := lineNum - 1
		if idx >= len(allLines) {
			break
		}
		trimmed := strings.TrimSpace(string(allLines[idx]))
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		count++
	}
	return count
}
