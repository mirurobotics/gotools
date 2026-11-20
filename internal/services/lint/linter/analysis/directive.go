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

func isGoStyleDirective(text string) bool {
	name, rest, ok := strings.Cut(text, ":")
	if !ok || name == "" || rest == "" {
		return false
	}
	for _, char := range name {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	char := rest[0]
	isLetter := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
	return isLetter || char >= '0' && char <= '9'
}

func hasAnyPrefix(text string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}
