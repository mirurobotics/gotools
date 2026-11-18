package analysis

import (
	"regexp"
	"strings"
	"sync"
)

// directiveRe matches Go-style directives such as //go:build.
func directiveRe() *regexp.Regexp {
	return sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^//[a-z0-9]+:[a-zA-Z0-9]`)
	})()
}

// IsDirective reports whether a comment is a tool directive such as //nolint.
func IsDirective(text string) bool {
	if directiveRe().MatchString(text) {
		return true
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	return strings.HasPrefix(body, "nolint") ||
		strings.HasPrefix(body, "#nosec") ||
		strings.HasPrefix(body, "+build")
}
