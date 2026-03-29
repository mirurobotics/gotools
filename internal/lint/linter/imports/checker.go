package imports

import (
	"bytes"
	"go/token"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// Check reports a single diagnostic per import block that doesn't match
// its canonical form. This uses the same rebuildBlock logic as the fixer,
// so checker and fixer can never disagree about what's correct.
func Check(
	fset *token.FileSet,
	filename string,
	src []byte,
	blocks []ImportBlock,
) []analysis.Diagnostic {
	var diags []analysis.Diagnostic

	for _, block := range blocks {
		if !block.IsGrouped || len(block.Imports) <= 1 {
			continue
		}

		canonical := rebuildBlock(block)
		startOff := fset.Position(block.Pos).Offset
		endOff := fset.Position(block.End).Offset
		actual := src[startOff:endOff]

		if !bytes.Equal(actual, []byte(canonical)) {
			pos := fset.Position(block.Pos)
			diags = append(diags, analysis.Diagnostic{
				File: filename,
				Line: pos.Line,
				Message: "import block is not in canonical order " +
					"(group: std, internal, external; sorted alphabetically within each group)",
			})
		}
	}

	return diags
}
