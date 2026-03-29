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

// LineStartOffset returns the byte offset of the beginning of the line
// containing the given offset.
func LineStartOffset(src []byte, off int) int {
	for i := off - 1; i >= 0; i-- {
		if src[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// LineEndOffset returns the byte offset of the newline (or EOF) ending
// the line containing the given offset.
func LineEndOffset(src []byte, off int) int {
	for i := off; i < len(src); i++ {
		if src[i] == '\n' {
			return i
		}
	}
	return len(src)
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

// CollapsedLineWidth computes the total visual width of a line
// after collapsing an expression, including leading indent, the
// collapsed text, and any trailing content after endOff.
func CollapsedLineWidth(
	src []byte, startOff, endOff, tabWidth int, collapsed string,
) int {
	lineStart := LineStartOffset(src, startOff)
	indent := VisualWidth(string(src[lineStart:startOff]), tabWidth)
	lineEnd := LineEndOffset(src, endOff)
	trailing := VisualWidth(string(src[endOff:lineEnd]), tabWidth)
	return indent + VisualWidth(collapsed, tabWidth) + trailing
}

// HasCommentsBetween returns true if there are any comments
// between the open and close positions.
func HasCommentsBetween(
	fset *token.FileSet,
	f *ast.File,
	open,
	closePos token.Pos,
) bool {
	openOff := fset.Position(open).Offset
	closeOff := fset.Position(closePos).Offset

	for _, cg := range f.Comments {
		for _, c := range cg.List {
			cOff := fset.Position(c.Pos()).Offset
			if cOff > openOff && cOff < closeOff {
				return true
			}
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
