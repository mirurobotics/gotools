package analysis

import (
	"go/parser"
	"testing"
)

func TestIsTestingType(t *testing.T) {
	tests := []struct {
		expr string
		pkg  string
		want bool
	}{
		{"*testing.T", "testing", true},
		{"*testing.B", "testing", true},
		{"*testing.F", "testing", true},
		{"*testing.M", "testing", true},
		{"*testing.PB", "testing", false},
		{"testing.TB", "testing", true},
		{"*tst.T", "tst", true},
		{"testing.T", "testing", false},
		{"*testing.TB", "testing", false},
		{"*testing.Cover", "testing", false},
		{"*other.T", "testing", false},
		{"*testing.T", "", false},
		{"*[]int", "testing", false},
		{"f().T", "testing", false},
	}
	for _, tc := range tests {
		parsed, err := parser.ParseExpr(tc.expr)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.expr, err)
		}
		if got := IsTestingType(parsed, tc.pkg); got != tc.want {
			t.Errorf("IsTestingType(%q, %q) = %v, want %v",
				tc.expr, tc.pkg, got, tc.want)
		}
	}
}

func TestIsTestingHandle(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"*testing.T", true},
		{"*testing.B", true},
		{"*testing.F", true},
		{"testing.TB", true},
		{"*testing.M", false},
		{"*testing.PB", false},
		{"testing.T", false},
	}
	for _, tc := range tests {
		parsed, err := parser.ParseExpr(tc.expr)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.expr, err)
		}
		if got := IsTestingHandle(parsed, "testing"); got != tc.want {
			t.Errorf("IsTestingHandle(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestIsTestingIface(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"TestingI", true},
		{"assert.TestingI", true},
		{"*TestingI", false},
		{"TestingI[int]", false},
		{"pkg.TestingIface", false},
		{"OtherI", false},
		{"f().TestingI", false},
	}
	for _, tc := range tests {
		parsed, err := parser.ParseExpr(tc.expr)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.expr, err)
		}
		if got := IsTestingIface(parsed); got != tc.want {
			t.Errorf("IsTestingIface(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestTestingPkgName_Imported(t *testing.T) {
	file := parseTestFile(t, `package foo; import "testing"`)
	if got := TestingPkgName(file); got != "testing" {
		t.Errorf("got %q, want %q", got, "testing")
	}
}

func TestTestingPkgName_Aliased(t *testing.T) {
	file := parseTestFile(t, `package foo; import tst "testing"`)
	if got := TestingPkgName(file); got != "tst" {
		t.Errorf("got %q, want %q", got, "tst")
	}
}

func TestTestingPkgName_NotImported(t *testing.T) {
	file := parseTestFile(t, `package foo; import "fmt"`)
	if got := TestingPkgName(file); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
