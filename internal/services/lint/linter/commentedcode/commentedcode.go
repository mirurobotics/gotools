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
	fset *token.FileSet, filename string, file *ast.File, src []byte,
) []analysis.Diagnostic {
	outputs := analysis.ExampleOutputs(filename, file)
	preambles := analysis.CgoPreambles(file)
	code := source{tf: fset.File(file.Pos()), src: src}
	var diags []analysis.Diagnostic
	for _, group := range file.Comments {
		skip := outputs[group] || preambles[group] || isTrailing(code, group)
		if skip || analysis.HasNolint(group, "commentedcode") {
			continue
		}
		if pos := findCode(code, group); pos.IsValid() {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(pos).Line,
				Message: message,
			})
		}
	}
	return diags
}

func isTrailing(code source, group *ast.CommentGroup) bool {
	return strings.TrimSpace(code.before(group.Pos())) != ""
}

func findCode(code source, group *ast.CommentGroup) token.Pos {
	for _, run := range runs(code, group) {
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
	for _, entry := range run[:last] {
		if strings.TrimSpace(entry.text) != "" {
			return entry.pos
		}
	}
	return run[last].pos
}

func perLine(run []line) token.Pos {
	if len(run) == 1 {
		return token.NoPos
	}
	for _, entry := range run {
		if !entry.indented && isCode(entry.text) {
			return entry.pos
		}
	}
	return token.NoPos
}
