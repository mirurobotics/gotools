package filelen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const ruleName = "filelen"

// Check flags non-test files over maxLen code lines unless nolinted at the package.
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
	count := countLines(fset, f, src)
	if count <= maxLen {
		return nil
	}
	line := fset.Position(f.Package).Line
	if hasNolint(fset, f, line) {
		return nil
	}
	return []analysis.Diagnostic{{
		File:    filename,
		Line:    line,
		Message: fmt.Sprintf("file is %d lines (limit %d)", count, maxLen),
	}}
}

func hasNolint(fset *token.FileSet, f *ast.File, line int) bool {
	var comments []*ast.Comment
	if f.Doc != nil {
		comments = append(comments, f.Doc.List...)
	}
	for _, cg := range f.Comments {
		if fset.Position(cg.Pos()).Line == line {
			comments = append(comments, cg.List...)
		}
	}
	for _, c := range comments {
		if analysis.NolintsRule(c.Text, ruleName) {
			return true
		}
	}
	return false
}

func countLines(fset *token.FileSet, f *ast.File, src []byte) int {
	code := bytes.Clone(src)
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			start := fset.Position(c.Pos()).Offset
			end := commentEnd(src, start, c)
			for i := start; i < end && i < len(code); i++ {
				if code[i] != '\n' {
					code[i] = ' '
				}
			}
		}
	}
	count := 0
	for _, line := range bytes.Split(code, []byte("\n")) {
		if len(bytes.TrimSpace(line)) > 0 {
			count++
		}
	}
	return count
}

func commentEnd(src []byte, start int, c *ast.Comment) int {
	if !strings.HasPrefix(c.Text, "/*") {
		return start + len(c.Text)
	}
	return start + 2 + bytes.Index(src[start+2:], []byte("*/")) + 2
}
