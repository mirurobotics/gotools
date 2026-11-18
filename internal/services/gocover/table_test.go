package gocover

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"
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

func TestRun(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		var buf bytes.Buffer
		p := Progress{Enabled: enabled, Table: testTable()}
		got := Run(p, &buf, 3, 2, func(i int) (int, string) {
			return i * 10, fmt.Sprintf("PASS  pkg%d\nmore\n", i)
		})
		if fmt.Sprint(got) != "[0 10 20]" {
			t.Errorf("enabled=%v: results = %v", enabled, got)
		}
		lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
		if !enabled {
			if buf.Len() != 0 {
				t.Errorf("disabled run printed %q", buf.String())
			}
			continue
		}
		sort.Strings(lines)
		want := []string{"[1/3]  PASS  pkg0", "[2/3]  PASS  pkg1", "[3/3]  PASS  pkg2"}
		if strings.Join(lines, "|") != strings.Join(want, "|") {
			t.Errorf("progress lines = %q, want %q", lines, want)
		}
	}
}
