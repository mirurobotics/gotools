package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
)

// Diagnostic represents a single lint violation.
type Diagnostic struct {
	File    string
	Line    int
	Message string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d: %s", d.File, d.Line, d.Message)
}

// CollapsedLineWidth returns the visual width of the line after collapsing.
func CollapsedLineWidth(
	src []byte, startOff, endOff, tabWidth int, collapsed string,
) int {
	lineStart := LineStartOffset(src, startOff)
	indent := VisualWidth(string(src[lineStart:startOff]), tabWidth)
	lineEnd := LineEndOffset(src, endOff)
	trailing := VisualWidth(string(src[endOff:lineEnd]), tabWidth)
	return indent + VisualWidth(collapsed, tabWidth) + trailing
}

// LineStartOffset returns the offset of the start of off's line.
func LineStartOffset(src []byte, off int) int {
	for i := off - 1; i >= 0; i-- {
		if src[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// VisualWidth returns the visual width of a string, expanding tabs.
func VisualWidth(s string, tabWidth int) int {
	w := 0
	for _, ch := range s {
		if ch == '\t' {
			w += tabWidth - (w % tabWidth)
		} else {
			w++
		}
	}
	return w
}

// LineEndOffset returns the offset of the newline or EOF ending off's line.
func LineEndOffset(src []byte, off int) int {
	for i := off; i < len(src); i++ {
		if src[i] == '\n' {
			return i
		}
	}
	return len(src)
}

// HasCommentsBetween reports whether any comment lies between open and closePos.
func HasCommentsBetween(
	fset *token.FileSet,
	f *ast.File,
	open,
	closePos token.Pos,
) bool {
	openOff := fset.Position(open).Offset
	closeOff := fset.Position(closePos).Offset

	for _, c := range Comments(f) {
		cOff := fset.Position(c.Pos()).Offset
		if cOff > openOff && cOff < closeOff {
			return true
		}
	}
	return false
}

// ContainsFuncLit returns true if the AST node is or contains a *ast.FuncLit.
func ContainsFuncLit(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(node ast.Node) bool {
		if found {
			return false
		}
		if _, ok := node.(*ast.FuncLit); ok {
			found = true
			return false
		}
		return true
	})
	return found
}
