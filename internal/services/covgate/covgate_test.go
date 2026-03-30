package covgate

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintHeader(t *testing.T) {
	var buf bytes.Buffer
	printHeader(&buf)
	out := buf.String()

	for _, col := range []string{"STATUS", "COVERAGE", "REQUIRED", "PACKAGE"} {
		if !strings.Contains(out, col) {
			t.Errorf("output missing column %q", col)
		}
	}
	if !strings.Contains(out, "------") {
		t.Error("output missing separator line")
	}
}
