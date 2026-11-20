package analysis

import "testing"

func TestImportName(t *testing.T) {
	src := `package foo

import (
	"context"
	ctx2 "context"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
	"github.com/google/go-cmp/cmp"
	"v2"
	` + "`raw/pkg`" + `
)
`
	f := parseTestFile(t, src)
	want := []string{"context", "ctx2", "chi", "yaml", "cmp", "v2", "pkg"}
	for i, imp := range f.Imports {
		if got := ImportName(imp); got != want[i] {
			t.Errorf("import %d: got %q, want %q", i, got, want[i])
		}
	}
}

func TestPkgImportName(t *testing.T) {
	f := parseTestFile(t, `package foo; import t2 "testing"`)
	if got := PkgImportName(f, "testing"); got != "t2" {
		t.Errorf("got %q, want t2", got)
	}
	if got := PkgImportName(f, "context"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
