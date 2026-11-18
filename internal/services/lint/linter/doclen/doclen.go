package doclen

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports doc comments longer than maxLines, except package docs in doc.go.
func Check(
	fset *token.FileSet,
	filename string,
	f *ast.File,
	maxLines int,
) []analysis.Diagnostic {
	if maxLines <= 0 {
		return nil
	}
	var docs []*ast.CommentGroup
	if filepath.Base(filename) != "doc.go" {
		docs = append(docs, f.Doc)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			docs = append(docs, x.Doc)
		case *ast.GenDecl:
			docs = append(docs, x.Doc)
		case *ast.TypeSpec:
			docs = append(docs, x.Doc)
		case *ast.ValueSpec:
			docs = append(docs, x.Doc)
		case *ast.Field:
			docs = append(docs, x.Doc)
		}
		return true
	})

	var diags []analysis.Diagnostic
	for _, cg := range docs {
		if cg == nil {
			continue
		}
		n := CountLines(cg)
		if n <= maxLines {
			continue
		}
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    fset.Position(cg.Pos()).Line,
			Message: fmt.Sprintf("doc comment is %d lines (limit %d)", n, maxLines),
		})
	}
	return diags
}

// CountLines counts the prose lines in cg, ignoring directives.
func CountLines(cg *ast.CommentGroup) int {
	var lines []string
	for _, c := range cg.List {
		if analysis.IsDirective(c.Text) {
			continue
		}
		text := strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
		lines = append(lines, strings.Split(text, "\n")...)
	}
	for len(lines) > 0 && isBlank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && isBlank(lines[0]) {
		lines = lines[1:]
	}
	return len(lines)
}

func isBlank(line string) bool {
	return strings.TrimSpace(strings.TrimPrefix(line, "//")) == ""
}
