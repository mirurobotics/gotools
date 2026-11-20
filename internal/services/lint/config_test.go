package lint

import (
	"bytes"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter"
)

func TestParseExclusions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single", "funclen", 1},
		{"multiple", "funclen,paramcount,linelen", 3},
		{"with spaces", "funclen , paramcount", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseExclusions(tc.input)
			if len(got) != tc.want {
				t.Errorf("len = %d, want %d", len(got), tc.want)
			}
		})
	}
}

func TestBuildSingleRuleExclusions(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		excl := BuildSingleRuleExclusions("funclen")
		if excl == nil {
			t.Fatal("expected non-nil exclusions")
		}
		if excl[linter.Rule("funclen")] {
			t.Error("funclen should not be excluded")
		}
		if !excl[linter.Rule("linelen")] {
			t.Error("linelen should be excluded")
		}
	})

	t.Run("unknown rule", func(t *testing.T) {
		excl := BuildSingleRuleExclusions("nonexistent")
		if excl != nil {
			t.Error("expected nil for unknown rule")
		}
	})
}

func TestSummarizeOutcome(t *testing.T) {
	tests := []struct {
		name     string
		doFix    bool
		diags    int
		fixed    int
		wantCode int
	}{
		{"clean", false, 0, 0, 0},
		{"violations", false, 3, 0, 1},
		{"fixed", true, 0, 2, 0},
		{"fixed with remaining", true, 1, 2, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			code := SummarizeOutcome(&buf, tc.doFix, tc.diags, tc.fixed)
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d", code, tc.wantCode)
			}
		})
	}
}
