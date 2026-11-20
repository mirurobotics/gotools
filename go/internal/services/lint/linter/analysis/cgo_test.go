package analysis

import (
	"go/parser"
	"go/token"
	"testing"
)

const cgoSrc = `package foo

// single preamble
import "C"

// grouped doc, not a preamble
import (
	// spec preamble
	"C"
	"fmt"
)

// grouped doc, undocumented C spec
import (
	"C"
	"strings"
)

// plain import doc
import "os"

var _ = fmt.Sprint
var _ = os.Args
var _ = strings.ToUpper
`

func TestCgoPreambles(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", cgoSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := CgoPreambles(f)
	want := map[string]bool{
		"single preamble\n":                  true,
		"grouped doc, not a preamble\n":      false,
		"spec preamble\n":                    true,
		"grouped doc, undocumented C spec\n": false,
		"plain import doc\n":                 false,
	}
	for _, cg := range f.Comments {
		text := cg.Text()
		if got[cg] != want[text] {
			t.Errorf("CgoPreambles[%q] = %v, want %v", text, got[cg], want[text])
		}
	}
}
