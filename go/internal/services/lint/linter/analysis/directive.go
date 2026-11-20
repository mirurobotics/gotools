package analysis

import "strings"

// IsDirective reports whether a comment is a tool directive such as //nolint.
func IsDirective(text string) bool {
	if isGoStyleDirective(strings.TrimPrefix(text, "//")) {
		return true
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	if hasAnyPrefix(text, "//export ", "//extern ", "//line ") {
		return true
	}
	return hasAnyPrefix(body, "nolint", "#nosec", "+build")
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
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
