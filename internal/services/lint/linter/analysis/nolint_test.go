package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestNolintsRule(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"//nolint:privdoc // reason", true},
		{"//nolint:funclen, privdoc // reason", true},
		{"// nolint:privdoc", true},
		{"//nolint:privdocs // reason", false},
		{"//nolint:privdoc reason without slashes", true},
		{"//nolint:funclen,", false},
		{"//nolint:funclen // privdoc", false},
		{"// a privdoc note", false},
		{"//miru:nolint:privdoc", true},
		{"//nolint:gosec,privdoc // reason", true},
		{"//nolint:privdoc , gosec", true},
		{"//nolint:privdoc// reason", true},
		{"// remove nolint:privdoc once split", false},
	}
	for _, tc := range tests {
		if got := NolintsRule(tc.text, "privdoc"); got != tc.want {
			t.Errorf("NolintsRule(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestHasPackageNolint(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{"line above clause", "//nolint:filelen // big table\npackage p\n", true},
		{"on clause line", "package p //miru:nolint:filelen\n", true},
		{"in a nolint list", "package p //nolint:gosec,filelen // reason\n", true},
		{"higher in doc", "//nolint:filelen\n//go:generate x\npackage p\n", true},
		{"detached from clause", "//nolint:filelen\n\npackage p\n", false},
		{"below clause", "package p\n//nolint:filelen\n", false},
		{"other rule", "//nolint:funclen\npackage p\n", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", tc.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if got := HasPackageNolint(fset, file, "filelen"); got != tc.want {
				t.Errorf("HasPackageNolint = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHasNolint(t *testing.T) {
	group := &ast.CommentGroup{List: []*ast.Comment{
		{Slash: 0, Text: "// note"},
		{Slash: 0, Text: "//nolint:privdoc // reason"},
	}}
	if !HasNolint(group, "privdoc") {
		t.Error("HasNolint missed a directive in the group")
	}
	if HasNolint(group, "funclen") {
		t.Error("HasNolint matched an unnamed linter")
	}
}

func TestNolintOnLines(t *testing.T) {
	src := "package p\n\n//nolint:funclen // reason\nvar x = 1 //nolint:other\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if !NolintOnLines(fset, file, "funclen", 3, 4) {
		t.Error("directive on line 3 not found in [3, 4]")
	}
	if !NolintOnLines(fset, file, "funclen", 3, 3) {
		t.Error("directive on line 3 not found in [3, 3]")
	}
	if NolintOnLines(fset, file, "funclen", 4, 4) {
		t.Error("directive on line 3 found in [4, 4]")
	}
}
