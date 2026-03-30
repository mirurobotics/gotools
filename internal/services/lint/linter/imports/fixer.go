package imports

import (
	"fmt"
	"go/token"
	"sort"
	"strings"
)

// Fix applies canonical import ordering to the source. Blocks are processed
// in reverse order so earlier byte offsets remain valid.
func Fix(fset *token.FileSet, src []byte, blocks []ImportBlock) []byte {
	if len(blocks) == 0 {
		return src
	}

	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		if !block.IsGrouped || len(block.Imports) <= 1 {
			continue
		}
		rebuilt := rebuildBlock(block)
		startOff := fset.Position(block.Pos).Offset
		endOff := fset.Position(block.End).Offset

		out := make([]byte, 0, startOff+len(rebuilt)+(len(src)-endOff))
		out = append(out, src[:startOff]...)
		out = append(out, rebuilt...)
		out = append(out, src[endOff:]...)
		src = out
	}

	return src
}

// rebuildBlock produces the canonical form of an import block.
// Used by both the checker (to compare) and the fixer (to replace).
func rebuildBlock(block ImportBlock) string {
	groups := make(map[ImportGroup][]Import)
	for _, imp := range block.Imports {
		groups[imp.Group] = append(groups[imp.Group], imp)
	}
	for g := range groups {
		sort.Slice(groups[g], func(i, j int) bool {
			return groups[g][i].Path < groups[g][j].Path
		})
	}

	var b strings.Builder
	b.WriteString("import (\n")

	first := true
	for _, g := range []ImportGroup{GroupStd, GroupInternal, GroupExternal} {
		imps, ok := groups[g]
		if !ok {
			continue
		}
		if !first {
			b.WriteString("\n")
		}
		first = false
		for _, imp := range imps {
			if imp.Name != "" {
				fmt.Fprintf(&b, "\t%s %q\n", imp.Name, imp.Path)
			} else {
				fmt.Fprintf(&b, "\t%q\n", imp.Path)
			}
		}
	}

	b.WriteString(")")
	return b.String()
}
