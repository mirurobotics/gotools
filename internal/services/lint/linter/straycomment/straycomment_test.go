package straycomment

import (
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	src       string
	wantLines []int
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", tt.src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	diags := Check(fset, "foo.go", f)
	if len(diags) != len(tt.wantLines) {
		t.Fatalf(
			"got %d diagnostics, want %d: %v",
			len(diags), len(tt.wantLines), diags,
		)
	}
	for i, d := range diags {
		if d.Line != tt.wantLines[i] || d.Message != message {
			t.Errorf("diag %d = %v, want line %d", i, d, tt.wantLines[i])
		}
	}
}

func TestCheck_SectionBannerFlaggedOncePerGroup(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// ----------------
// Helpers
// ----------------

func f() {}
`,
		wantLines: []int{3},
	})
}

func TestCheck_LooseNoteInStructAndConstBlockFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type T struct {
	A int

	// group of fields below

	B int
}

const (
	A = 1

	// more constants

	B = 2
)
`,
		wantLines: []int{6, 14},
	})
}

func TestCheck_AttachedCommentsAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `// Package foo is a package.
package foo

import (
	// fmt is used for printing.
	"fmt" // trailing

	// #include <stdlib.h>
	// #include <string.h>
	"C"
)

// T is a type.
type T struct {
	// A is a field.
	A int // trailing
}

// Group doc.
const (
	// A is a constant.
	A = 1 // trailing
)

// F prints.
func F() { fmt.Println() }
`,
		wantLines: nil,
	})
}

func TestCheck_CgoPreambleOnASingleImportAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// #include <stdio.h>
// #include <stdlib.h>
import "C"
`,
		wantLines: nil,
	})
}

func TestCheck_HeaderDirectivesAndFunctionCommentsSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `// Copyright 2026 Example.

//go:build linux

package foo

//go:generate echo hi

func f() {
	// funccomment handles this
}
`,
		wantLines: nil,
	})
}

func TestCheck_CommentAfterLastDeclarationFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {}

// TODO: more
`,
		wantLines: []int{5},
	})
}

func TestCheck_ImportCommentOnPackageClauseAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo // import "example.com/foo"

func f() {}
`,
		wantLines: nil,
	})
}

func TestCheck_CommentInFunctionSignatureFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(
	// a is documented.
	a int,
) {
}
`,
		wantLines: []int{4},
	})
}
