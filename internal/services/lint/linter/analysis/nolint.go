package analysis

import (
	"go/ast"
	"go/token"
	"strings"
)

// HasNolint reports whether any comment in cg is a nolint directive naming rule.
func HasNolint(cg *ast.CommentGroup, rule string) bool {
	for _, c := range cg.List {
		if NolintsRule(c.Text, rule) {
			return true
		}
	}
	return false
}

// NolintOnLines reports whether a nolint naming rule sits on a line in [from, to].
func NolintOnLines(fset *token.FileSet, f *ast.File, rule string, from, to int) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			line := fset.Position(c.Pos()).Line
			if line >= from && line <= to && NolintsRule(c.Text, rule) {
				return true
			}
		}
	}
	return false
}

// NolintsRule reports whether text is a //nolint or //miru:nolint naming rule.
func NolintsRule(text, rule string) bool {
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	body = strings.TrimPrefix(body, "miru:")
	list, ok := strings.CutPrefix(body, "nolint:")
	if !ok {
		return false
	}
	list, _, _ = strings.Cut(list, "//")
	for _, entry := range strings.Split(list, ",") {
		if fields := strings.Fields(entry); len(fields) > 0 && fields[0] == rule {
			return true
		}
	}
	return false
}
