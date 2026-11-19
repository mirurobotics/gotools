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
	f *ast.File,
	lim Limits,
) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	pkgLimit := lim.Doc
	if filepath.Base(filename) == "doc.go" {
		pkgLimit = lim.DocGoPkg
	}
	if d := check(fset, filename, f.Doc, pkgLimit); d != nil {
		diags = append(diags, *d)
	}
	preambles := analysis.CgoPreambles(f)
	for _, cg := range analysis.DocGroups(f) {
		if cg == nil || preambles[cg] {
			continue
		}
		if d := check(fset, filename, cg, lim.Doc); d != nil {
			diags = append(diags, *d)
		}
	}
	return diags
}

func check(
	fset *token.FileSet,
	filename string,
	cg *ast.CommentGroup,
	limit int,
) *analysis.Diagnostic {
	if cg == nil || limit <= 0 {
		return nil
	}
	n := analysis.ProseLineCount(cg)
	if n <= limit {
		return nil
	}
	return &analysis.Diagnostic{
		File:    filename,
		Line:    fset.Position(cg.Pos()).Line,
		Message: fmt.Sprintf("doc comment is %d lines (limit %d)", n, limit),
	}
}
