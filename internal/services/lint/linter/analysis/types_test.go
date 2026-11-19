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
		{"[]T", ""},
	}
	for _, tt := range tests {
		e, err := parser.ParseExpr(tt.expr)
		if err != nil {
			t.Fatal(err)
		}
		got := ""
		if id := BaseIdent(e); id != nil {
			got = id.Name
		}
		if got != tt.want {
			t.Errorf("BaseIdent(%s) = %q, want %q", tt.expr, got, tt.want)
		}
	}
}
