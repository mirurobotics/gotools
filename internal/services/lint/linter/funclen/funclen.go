package funclen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const nolintDirective = "nolint:funclen"

// Check reports a diagnostic for each function whose body exceeds maxLen
// non-blank, non-comment lines. Both FuncDecl and FuncLit are checked.
//
//nolint:funclen // two symmetric branches (FuncDecl/FuncLit) each need nolint check
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
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body == nil {
				return true
			}
			count := countLines(fset, fn.Body, allLines)
			if count > maxLen {
				pos := fset.Position(fn.Pos())
				if hasNolint(fset, f, pos.Line) {
					return true
				}
				diags = append(diags, analysis.Diagnostic{
					File: filename,
					Line: pos.Line,
					Message: fmt.Sprintf(
						"function %s is %d lines (limit %d)",
						fn.Name.Name, count, maxLen),
				})
			}
		case *ast.FuncLit:
			if fn.Body == nil {
				return true
			}
			count := countLines(fset, fn.Body, allLines)
			if count > maxLen {
				pos := fset.Position(fn.Pos())
				if hasNolint(fset, f, pos.Line) {
					return true
				}
				diags = append(diags, analysis.Diagnostic{
					File: filename,
					Line: pos.Line,
					Message: fmt.Sprintf(
						"anonymous function is %d lines (limit %d)",
						count, maxLen,
					),
				})
			}
		}
		return true
	})
	return diags
}

// hasNolint returns true if there is a //nolint:funclen comment on the
// given line or the line immediately before it.
func hasNolint(fset *token.FileSet, f *ast.File, line int) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			cLine := fset.Position(c.Pos()).Line
			if cLine >= line-1 && cLine <= line &&
				strings.Contains(c.Text, nolintDirective) {
				return true
			}
		}
	}
	return false
}

// countLines counts non-blank, non-comment-only lines inside a block
// statement (between opening and closing braces, exclusive).
func countLines(fset *token.FileSet, body *ast.BlockStmt, allLines [][]byte) int {
	startLine := fset.Position(body.Lbrace).Line // 1-based
	endLine := fset.Position(body.Rbrace).Line   // 1-based

	count := 0
	for lineNum := startLine + 1; lineNum < endLine; lineNum++ {
		idx := lineNum - 1 // 0-based index
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
