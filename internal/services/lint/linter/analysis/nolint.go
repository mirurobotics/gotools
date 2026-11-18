package analysis

import (
	"go/ast"
	"go/token"
	"strings"
)

// HasNolintNear reports whether f has a nolint directive naming rule on the
// given line or the line immediately before it.
func HasNolintNear(fset *token.FileSet, f *ast.File, line int, rule string) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			cLine := fset.Position(c.Pos()).Line
			if cLine >= line-1 && cLine <= line && NolintsRule(c.Text, rule) {
				return true
			}
		}
	}
	return false
}

// NolintsRule reports whether comment text is a nolint directive whose
// comma-separated rule list names rule, e.g. //nolint:gosec,filelen. The
// directive must start the comment, optionally with a miru: prefix.
func NolintsRule(text, rule string) bool {
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	body = strings.TrimPrefix(body, "miru:")
	list, ok := strings.CutPrefix(body, "nolint:")
	if !ok {
		return false
	}
	list, _, _ = strings.Cut(list, "//")
	for _, name := range strings.Split(list, ",") {
		if strings.TrimSpace(name) == rule {
			return true
		}
	}
	return false
}
