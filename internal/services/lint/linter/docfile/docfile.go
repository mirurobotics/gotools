package docfile

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports a doc.go file with more than maxLines non-blank lines.
func Check(filename string, src []byte, maxLines int) []analysis.Diagnostic {
	if maxLines <= 0 || filepath.Base(filename) != "doc.go" {
		return nil
	}
	n := 0
	for _, line := range bytes.Split(src, []byte("\n")) {
		if len(bytes.TrimSpace(line)) > 0 {
			n++
		}
	}
	if n <= maxLines {
		return nil
	}
	return []analysis.Diagnostic{{
		File:    filename,
		Line:    1,
		Message: fmt.Sprintf("doc.go is %d lines (limit %d)", n, maxLines),
	}}
}
