package funclen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports functions over maxLen code lines; a multi-line literal counts as one.
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
	endLine := min(fset.Position(body.Rbrace).Line, len(allLines)+1)
	skip := literalTailLines(fset, body)

	count := 0
	for lineNum := startLine + 1; lineNum < endLine; lineNum++ {
		if !skip[lineNum] && isCodeLine(allLines[lineNum-1]) {
			count++
		}
	}
	return count
}

func literalTailLines(fset *token.FileSet, body *ast.BlockStmt) map[int]bool {
	skip := make(map[int]bool)
	codeStarts := make(map[int]bool)
	var closingLines []int
	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case nil, *ast.CommentGroup, *ast.Comment:
			return false
		}
		codeStarts[fset.Position(n.Pos()).Line] = true
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		first := fset.Position(lit.Pos()).Line
		last := fset.Position(lit.End()).Line
		for line := first + 1; line <= last; line++ {
			skip[line] = true
		}
		closingLines = append(closingLines, last)
		return false
	})
	for _, line := range closingLines {
		if codeStarts[line] {
			delete(skip, line)
		}
	}
	return skip
}

func isCodeLine(line []byte) bool {
	trimmed := strings.TrimSpace(string(line))
	return trimmed != "" && !strings.HasPrefix(trimmed, "//")
}
