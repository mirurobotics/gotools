package doclen

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Limits holds the doc comment line limits.
type Limits struct {
	Doc      int
	DocGoPkg int
}

// Check reports doc comments over Limits.Doc, or a doc.go package doc over DocGoPkg.
func Check(
	fset *token.FileSet,
	filename string,
	file *ast.File,
	lim Limits,
) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	pkgLimit := lim.Doc
	if filepath.Base(filename) == "doc.go" {
		pkgLimit = lim.DocGoPkg
	}
	if diag := check(fset, filename, file.Doc, pkgLimit); diag != nil {
		diags = append(diags, *diag)
	}
	preambles := analysis.CgoPreambles(file)
	for _, group := range analysis.DocGroups(file) {
		if group == nil || preambles[group] {
			continue
		}
		if diag := check(fset, filename, group, lim.Doc); diag != nil {
			diags = append(diags, *diag)
		}
	}
	return diags
}

func check(
	fset *token.FileSet,
	filename string,
	group *ast.CommentGroup,
	limit int,
) *analysis.Diagnostic {
	if group == nil || limit <= 0 {
		return nil
	}
	lines := analysis.ProseLineCount(group)
	if lines <= limit {
		return nil
	}
	return &analysis.Diagnostic{
		File:    filename,
		Line:    fset.Position(group.Pos()).Line,
		Message: fmt.Sprintf("doc comment is %d lines (limit %d)", lines, limit),
	}
}
