package analysis

import (
	"go/ast"
	"go/token"
	"strings"
)

func HasPackageNolint(fset *token.FileSet, file *ast.File, rule string) bool {
	if file.Doc != nil && HasNolint(file.Doc, rule) {
		return true
	}
	line := fset.Position(file.Package).Line
	for _, group := range file.Comments {
		if fset.Position(group.Pos()).Line == line && HasNolint(group, rule) {
			return true
		}
	}
	return false
}

// HasNolint reports whether any comment in group is a nolint directive naming rule.
func HasNolint(group *ast.CommentGroup, rule string) bool {
	for _, comment := range group.List {
		if NolintsRule(comment.Text, rule) {
			return true
		}
	}
	return false
}

// NolintOnLines reports whether a nolint naming rule sits on a line in [from, through].
func NolintOnLines(
	fset *token.FileSet, file *ast.File, rule string, from, through int,
) bool {
	for _, comment := range Comments(file) {
		line := fset.Position(comment.Pos()).Line
		if line >= from && line <= through && NolintsRule(comment.Text, rule) {
			return true
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
