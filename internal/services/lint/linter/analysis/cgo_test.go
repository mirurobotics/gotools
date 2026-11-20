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
	file, err := parser.ParseFile(fset, "foo.go", cgoSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := CgoPreambles(file)
	want := map[string]bool{
		"single preamble\n":                  true,
		"grouped doc, not a preamble\n":      false,
		"spec preamble\n":                    true,
		"grouped doc, undocumented C spec\n": false,
		"plain import doc\n":                 false,
	}
	for _, group := range file.Comments {
		text := group.Text()
		if got[group] != want[text] {
			t.Errorf("CgoPreambles[%q] = %v, want %v", text, got[group], want[text])
		}
	}
}
