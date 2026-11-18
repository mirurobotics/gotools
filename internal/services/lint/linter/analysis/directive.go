package analysis

import "strings"

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

// isGoStyleDirective reports whether s has the form name:value, as in go:build.
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
