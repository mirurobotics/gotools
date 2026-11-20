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

// Check flags files over maxLen code lines unless nolinted at the package.
func Check(
	fset *token.FileSet,
	filename string,
	file *ast.File,
	src []byte,
	maxLen int,
) []analysis.Diagnostic {
	if maxLen <= 0 {
		return nil
	}
	count := countLines(fset, file, src)
	if count <= maxLen {
		return nil
	}
	line := fset.Position(file.Package).Line
	if hasNolint(fset, file, line) {
		return nil
	}
	return []analysis.Diagnostic{{
		File:    filename,
		Line:    line,
		Message: fmt.Sprintf("file is %d lines (limit %d)", count, maxLen),
	}}
}

func countLines(fset *token.FileSet, file *ast.File, src []byte) int {
	code := bytes.Clone(src)
	for _, comment := range analysis.Comments(file) {
		start := fset.Position(comment.Pos()).Offset
		blankRange(code, start, commentEnd(src, start, comment))
	}
	count := 0
	for _, line := range bytes.Split(code, []byte("\n")) {
		if len(bytes.TrimSpace(line)) > 0 {
			count++
		}
	}
	return count
}

func blankRange(code []byte, start, end int) {
	for i := start; i < end && i < len(code); i++ {
		if code[i] != '\n' {
			code[i] = ' '
		}
	}
}

func commentEnd(src []byte, start int, comment *ast.Comment) int {
	if !strings.HasPrefix(comment.Text, "/*") {
		return start + len(comment.Text)
	}
	return start + 2 + bytes.Index(src[start+2:], []byte("*/")) + 2
}

func hasNolint(fset *token.FileSet, file *ast.File, line int) bool {
	var comments []*ast.Comment
	if file.Doc != nil {
		comments = append(comments, file.Doc.List...)
	}
	for _, group := range file.Comments {
		if fset.Position(group.Pos()).Line == line {
			comments = append(comments, group.List...)
		}
	}
	for _, comment := range comments {
		if analysis.NolintsRule(comment.Text, ruleName) {
			return true
		}
	}
	return false
}
