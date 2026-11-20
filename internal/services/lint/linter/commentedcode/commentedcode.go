package commentedcode

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "commented-out code; delete it instead"

// Check reports comment groups that contain Go code which has been commented out.
func Check(
	fset *token.FileSet, filename string, f *ast.File, src []byte,
) []analysis.Diagnostic {
	outputs := analysis.ExampleOutputs(filename, f)
	preambles := analysis.CgoPreambles(f)
	s := source{tf: fset.File(f.Pos()), src: src}
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		skip := outputs[cg] || preambles[cg] || isTrailing(s, cg)
		if skip || analysis.HasNolint(cg, "commentedcode") {
			continue
		}
		if pos := findCode(s, cg); pos.IsValid() {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(pos).Line,
				Message: message,
			})
		}
	}
	return diags
}

func isTrailing(s source, cg *ast.CommentGroup) bool {
	return strings.TrimSpace(s.before(cg.Pos())) != ""
}

func findCode(s source, cg *ast.CommentGroup) token.Pos {
	for _, run := range runs(s, cg) {
		if isCode(joinText(run)) {
			return firstNonBlank(run)
		}
		if pos := perLine(run); pos.IsValid() {
			return pos
		}
	}
	return token.NoPos
}

func firstNonBlank(run []line) token.Pos {
	last := len(run) - 1
	for _, l := range run[:last] {
		if strings.TrimSpace(l.text) != "" {
			return l.pos
		}
	}
	return run[last].pos
}

func perLine(run []line) token.Pos {
	if len(run) == 1 {
		return token.NoPos
	}
	for _, l := range run {
		if !l.indented && isCode(l.text) {
			return l.pos
		}
	}
	return token.NoPos
}
