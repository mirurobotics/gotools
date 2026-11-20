package doclen

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename  string
	src       string
	lim       Limits
	wantLines []int
	wantMsg   string
}

func TestCheck_MultiLineFuncDocFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// F does a thing.
// And explains it at length.
func F() {}
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: []int{3},
		wantMsg:   "doc comment is 2 lines (limit 1)",
	})
}

func TestCheck_OneLineDocsWithDirectivesAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// F does a thing.
//
//nolint:funclen // reason
func F() {}

// T is a type.
//
//go:generate echo hi
type T struct {
	// A is a field.
	A int // trailing
}
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: nil,
		wantMsg:   "",
	})
}

func TestCheck_TypeValueAndFieldDocsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// T is a type
// with two lines.
type T struct {
	// A is a field
	// with two lines.
	A int
}

const (
	// C is a const
	// with two lines.
	C = 1
)
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: []int{3, 6, 12},
		wantMsg:   "",
	})
}

func TestCheck_BlockCommentCountedByLines(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

/* F does a thing
and more. */
func F() {}
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: []int{3},
		wantMsg:   "",
	})
}

func TestCheck_PackageDocFlaggedOutsideDocGo(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `// Package foo does things.
// At length.
package foo
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: []int{1},
		wantMsg:   "",
	})
}

func TestCheck_DocGoPackageDocWithinItsOwnLimitAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "doc.go",
		src: `// Package foo does things.
// At length.
package foo
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: nil,
		wantMsg:   "",
	})
}

func TestCheck_HigherLimitAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// F does a thing.
// And explains it.
func F() {}
`,
		lim:       Limits{Doc: 2, DocGoPkg: 20},
		wantLines: nil,
		wantMsg:   "",
	})
}

func TestCheck_DocGoPackageDocOverItsOwnLimitFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/doc.go",
		src: `// Copyright header.
// Not counted.

// Package foo does things.
// At length.
// And more.
package foo
`,
		lim:       Limits{Doc: 1, DocGoPkg: 2},
		wantLines: []int{4},
		wantMsg:   "doc comment is 3 lines (limit 2)",
	})
}

func TestCheck_CgoPreambleAndExportDirectiveSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// #include <stdio.h>
// #cgo LDFLAGS: -lm
// static int one() { return 1; }
import "C"

import (
	// #include <stdlib.h>
	// #include <string.h>
	"C"
)

// G is exported to C.
//
//export G
func G() {}

// F is exported to C.
//
//export F
func F() {}
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: nil,
		wantMsg:   "",
	})
}

func TestCheck_BlankBlockCommentLinesTrimmed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// Plain is an import doc
// over two lines.
import "fmt"

/*
F does a thing.
*/
func F() { fmt.Println() }
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: []int{3},
		wantMsg:   "doc comment is 2 lines (limit 1)",
	})
}

func TestCheck_DocOnAMultiSpecImportBlockWithCChecked(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// Imports for foo,
// over two lines.
import (
	"fmt"
	"C"
)

var _ = fmt.Sprint
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: []int{3},
		wantMsg:   "",
	})
}

func TestCheck_BlockDocAboveASingleCSpecWithItsOwnPreambleChecked(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// Imports for foo,
// over two lines.
import (
	// #include <stdlib.h>
	"C"
)
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: []int{3},
		wantMsg:   "",
	})
}

func TestCheck_DisabledWhenLimitIsZero(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// F does a thing.
// And explains it.
func F() {}
`,
		lim:       Limits{Doc: 0, DocGoPkg: 20},
		wantLines: nil,
		wantMsg:   "",
	})
}

func TestCheck_LocalDeclarationsLeftToFunccomment(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func F() {
	// x is a local type
	// with two lines.
	type x struct {
		// A is a field
		// with two lines.
		A int
	}
	_ = x{}
}
`,
		lim:       Limits{Doc: 1, DocGoPkg: 20},
		wantLines: nil,
		wantMsg:   "",
	})
}

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	fset, file := linttest.Parse(t, tc.filename, tc.src)
	diags := Check(fset, tc.filename, file, tc.lim)
	linttest.AssertLines(t, diags, tc.wantLines, tc.wantMsg)
}
