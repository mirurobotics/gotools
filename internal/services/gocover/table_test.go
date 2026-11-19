package gocover

import (
	"bytes"
	"fmt"
	"slices"
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
		var peak atomic.Int32
		row := func(v int) string { return fmt.Sprintf("PASS  pkg%d\nmore\n", v/10) }
		p := Progress{Enabled: enabled, Table: emptyTable}
		got := RunEach(p, &buf, 12, 2, peakTrackingRun(&peak), row)
		if len(got) != 12 {
			t.Fatalf("enabled=%v: got %d results, want 12", enabled, len(got))
		}
		want := make([]int, 12)
		for i := range want {
			want[i] = i * 10
		}
		if !slices.Equal(got, want) {
			t.Fatalf("enabled=%v: results = %v", enabled, got)
		}
		if p := peak.Load(); p != 2 {
			t.Errorf("enabled=%v: %d calls ran at once, want 2", enabled, p)
		}
		checkProgressOutput(t, enabled, buf.String())
	}
}

func peakTrackingRun(peak *atomic.Int32) func(int) int {
	var inFlight atomic.Int32
	return func(i int) int {
		cur := inFlight.Add(1)
		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}
		deadline := time.Now().Add(time.Second)
		for i < 2 && peak.Load() < 2 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		inFlight.Add(-1)
		return i * 10
	}
}

func checkProgressOutput(t *testing.T, enabled bool, out string) {
	t.Helper()
	if !enabled {
		if out != "" {
			t.Errorf("disabled run printed %q", out)
		}
		return
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	sort.Strings(lines)
	want := make([]string, 0, 12)
	for i := range 12 {
		want = append(want, fmt.Sprintf("[%2d/12]  PASS  pkg%d", i+1, i))
	}
	sort.Strings(want)
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("progress lines = %q, want %q", lines, want)
	}
}

func TestRunEach_DefaultsParallelism(t *testing.T) {
	p := Progress{Enabled: false, Table: emptyTable}
	id := func(i int) int { return i }
	got := RunEach(p, &bytes.Buffer{}, 2, 0, id, nil)
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

var emptyTable = Table{Format: "", Names: nil, Dashes: nil}
