package analysis

import (
	"go/parser"
	"testing"
)

func TestBaseIdent(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"T", "T"},
		{"*T", "T"},
		{"**T", "T"},
		{"pkg.T", "T"},
		{"*pkg.T", "T"},
		{"T[K]", "T"},
		{"*T[K, V]", "T"},
		{"*(T)", "T"},
		{"[]T", ""},
	}
	for _, tc := range tests {
		parsed, err := parser.ParseExpr(tc.expr)
		if err != nil {
			t.Fatal(err)
		}
		got := ""
		if id := BaseIdent(parsed); id != nil {
			got = id.Name
		}
		if got != tc.want {
			t.Errorf("BaseIdent(%s) = %q, want %q", tc.expr, got, tc.want)
		}
	}
}

func TestQualifiedTypeName(t *testing.T) {
	tests := []struct {
		expr     string
		pkg      string
		wantName string
		wantPtr  bool
	}{
		{"*testing.PB", "testing", "PB", true},
		{"ctx.Context", "ctx", "Context", false},
		{"*gin.Context", "gin", "Context", true},
		{"other.T", "testing", "", false},
		{"testing.T", "", "", false},
		{"T", "testing", "", false},
		{"f().T", "testing", "", false},
	}
	for _, tc := range tests {
		parsed, err := parser.ParseExpr(tc.expr)
		if err != nil {
			t.Fatal(err)
		}
		name, isPtr := QualifiedTypeName(parsed, tc.pkg)
		if name != tc.wantName || isPtr != tc.wantPtr {
			t.Errorf("QualifiedTypeName(%s, %q) = (%q, %v), want (%q, %v)",
				tc.expr, tc.pkg, name, isPtr, tc.wantName, tc.wantPtr)
		}
	}
}
