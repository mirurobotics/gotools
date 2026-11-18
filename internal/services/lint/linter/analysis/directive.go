package analysis

import (
	"go/ast"
	"strings"
)

// IsDirective reports whether a comment is a tool directive such as //nolint.
func IsDirective(text string) bool {
	if isGoStyleDirective(strings.TrimPrefix(text, "//")) {
		return true
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	for _, cgo := range []string{"//export ", "//extern ", "//line "} {
		if strings.HasPrefix(text, cgo) {
			return true
		}
	}
	return strings.HasPrefix(body, "nolint") ||
		strings.HasPrefix(body, "#nosec") ||
		strings.HasPrefix(body, "+build")
}

func isGoStyleDirective(s string) bool {
	name, rest, ok := strings.Cut(s, ":")
	if !ok || name == "" || rest == "" {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	r := rest[0]
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

// ProseLines counts the non-blank, non-directive lines in cg.
func ProseLines(cg *ast.CommentGroup) int {
	var lines []string
	for _, c := range cg.List {
		if IsDirective(c.Text) {
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
