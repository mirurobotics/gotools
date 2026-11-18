package analysis

import (
	"strings"
)

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
