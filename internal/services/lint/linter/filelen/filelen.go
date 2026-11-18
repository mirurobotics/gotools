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

// Check reports a diagnostic when a file has more than maxLen non-blank,
// non-comment lines. Test files and test helpers are exempt.
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
	if analysis.HasNolintNear(fset, f, line, ruleName) {
		return nil
	}
	return []analysis.Diagnostic{{
		File:    filename,
		Line:    line,
		Message: fmt.Sprintf("file is %d lines (limit %d)", count, maxLen),
	}}
}

// countLines counts lines in src that hold something other than whitespace
// and comments. Comment bytes, including /* */ blocks, are blanked first.
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

// commentEnd returns the byte offset just past comment c in src. The scanner
// strips \r from comment text, so c.End() undershoots for CRLF block comments.
func commentEnd(src []byte, start int, c *ast.Comment) int {
	if !strings.HasPrefix(c.Text, "/*") {
		return start + len(c.Text)
	}
	// The file parsed, so the block comment is terminated.
	return start + 2 + bytes.Index(src[start+2:], []byte("*/")) + 2
}
