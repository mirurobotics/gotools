package gocover

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testTable() Table {
	return Table{
		Format: "%-6s  %s\n",
		Names:  []string{"STATUS", "PACKAGE"},
		Dashes: []string{"------", "-------"},
	}
}

func TestProgress_WriteHeader(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    string
	}{
		{
			name:    "plain table",
			enabled: false,
			want:    "STATUS  PACKAGE\n------  -------\n",
		},
		{
			name:    "progress table",
			enabled: true,
			want: "Running 12 packages with parallelism=3; progress:\n" +
				"COUNT    STATUS  PACKAGE\n" +
				"-------  ------  -------\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Progress{Enabled: tt.enabled, Table: testTable()}.WriteHeader(&buf, 12, 3)
			if buf.String() != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", buf.String(), tt.want)
			}
		})
	}
}

func TestRunEach(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		var buf bytes.Buffer
		var inFlight, peak atomic.Int32
		run := func(i int) int {
			cur := inFlight.Add(1)
			for {
				old := peak.Load()
				if cur <= old || peak.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			inFlight.Add(-1)
			return i * 10
		}
		row := func(v int) string { return fmt.Sprintf("PASS  pkg%d\nmore\n", v/10) }
		got := RunEach(&buf, enabled, 12, 2, run, row)
		if got[0] != 0 || got[11] != 110 || len(got) != 12 {
			t.Errorf("enabled=%v: results = %v", enabled, got)
		}
		if p := peak.Load(); p > 2 {
			t.Errorf("enabled=%v: %d calls ran at once, limit 2", enabled, p)
		}
		if !enabled {
			if buf.Len() != 0 {
				t.Errorf("disabled run printed %q", buf.String())
			}
			continue
		}
		lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
		sort.Strings(lines)
		if len(lines) != 12 || lines[0] != "[ 1/12]  PASS  pkg0" ||
			lines[11] != "[12/12]  PASS  pkg11" {
			t.Errorf("progress lines = %q", lines)
		}
	}
}

func TestRunEach_DefaultsParallelism(t *testing.T) {
	got := RunEach(&bytes.Buffer{}, false, 2, 0, func(i int) int { return i }, nil)
	if len(got) != 2 || got[1] != 1 {
		t.Errorf("results = %v", got)
	}
}

func TestProgress_RestAndIndent(t *testing.T) {
	plain := Progress{Enabled: false, Table: testTable()}
	live := Progress{Enabled: true, Table: testTable()}
	if got := plain.Rest("a\nb\n"); got != "a\nb\n" {
		t.Errorf("plain Rest = %q", got)
	}
	if got := live.Rest("a\nb\n"); got != "b\n" {
		t.Errorf("progress Rest = %q", got)
	}
	if got := plain.Indent(12); got != "" {
		t.Errorf("plain Indent = %q", got)
	}
	if got := live.Indent(12); got != strings.Repeat(" ", 9) {
		t.Errorf("progress Indent = %q", got)
	}
}
