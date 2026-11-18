package todoissue

import "strings"

// isDirective reports whether a comment is a tool directive such as //nolint.
func isDirective(text string) bool {
	body, ok := strings.CutPrefix(text, "//")
	if !ok {
		return false
	}
	if isGoStyleDirective(body) {
		return true
	}
	for _, cgo := range []string{"export ", "extern ", "line "} {
		if strings.HasPrefix(body, cgo) {
			return true
		}
	}
	body = strings.TrimSpace(body)
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
