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
func VisualWidth(text string, tabWidth int) int {
	width := 0
	for _, char := range text {
		if char == '\t' {
			width += tabWidth - (width % tabWidth)
		} else {
			width++
		}
	}
	return width
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
	file *ast.File,
	open,
	closePos token.Pos,
) bool {
	openOff := fset.Position(open).Offset
	closeOff := fset.Position(closePos).Offset

	for _, comment := range Comments(file) {
		cOff := fset.Position(comment.Pos()).Offset
		if cOff > openOff && cOff < closeOff {
			return true
		}
	}
	return false
}

// ContainsFuncLit returns true if the AST node is or contains a *ast.FuncLit.
func ContainsFuncLit(root ast.Node) bool {
	found := false
	ast.Inspect(root, func(node ast.Node) bool {
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
