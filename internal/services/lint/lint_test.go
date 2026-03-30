package lint

import "testing"

func TestFilterDeadcodeOutput(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		exclude string
		want    int
	}{
		{"empty", "", "", 0},
		{"filters mod path", "/go/pkg/mod/foo.go:1: unused", "", 0},
		{"keeps local", "internal/foo.go:1: unused", "", 1},
		{
			"exclude pattern",
			"internal/foo.go:1: Unused\n" +
				"internal/bar.go:1: Other", "Unused", 1,
		},
		{"mixed", "/go/pkg/mod/x.go:1: a\ninternal/y.go:1: b\n", "", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterDeadcodeOutput(tt.raw, tt.exclude)
			if len(got) != tt.want {
				t.Errorf("len = %d, want %d: %v", len(got), tt.want, got)
			}
		})
	}
}
