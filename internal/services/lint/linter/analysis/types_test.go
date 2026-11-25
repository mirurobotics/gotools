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
	parsed, err := parser.ParseExpr("*testing.PB")
	if err != nil {
		t.Fatal(err)
	}
	name, isPtr := QualifiedTypeName(parsed, "testing")
	if name != "PB" || !isPtr {
		t.Errorf("QualifiedTypeName = (%q, %v), want (\"PB\", true)", name, isPtr)
	}
}
