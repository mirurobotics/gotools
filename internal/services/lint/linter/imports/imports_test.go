package imports

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck_Correct(t *testing.T) {
	src := []byte(`package foo

import (
	"fmt"
	"strings"

	"github.com/mirurobotics/core/pkg/errs"

	"github.com/go-git/go-git/v5"
)
`)
	assertNoDiags(t, src)
}

func TestCheck_WrongGroupOrder(t *testing.T) {
	src := []byte(`package foo

import (
	"github.com/go-git/go-git/v5"
	"fmt"
	"github.com/mirurobotics/core/pkg/errs"
)
`)
	assertHasDiags(t, src)
}

func TestCheck_UnsortedWithinGroup(t *testing.T) {
	src := []byte(`package foo

import (
	"strings"
	"fmt"
)
`)
	assertHasDiags(t, src)
}

func TestCheck_MissingGroupSeparator(t *testing.T) {
	src := []byte(`package foo

import (
	"fmt"
	"github.com/mirurobotics/core/pkg/errs"
)
`)
	assertHasDiags(t, src)
}

func TestCheck_MissingGroupSeparator_InternalExternal(t *testing.T) {
	src := []byte(`package foo

import (
	"fmt"

	"github.com/mirurobotics/core/pkg/errs"
	"github.com/go-git/go-git/v5"
)
`)
	assertHasDiags(t, src)
}

func TestCheck_ExtraBlankLineWithinGroup(t *testing.T) {
	src := []byte(`package foo

import (
	"github.com/mirurobotics/core/pkg/errs"

	"github.com/mirurobotics/core/pkg/files"
)
`)
	assertHasDiags(t, src)
}

func assertHasDiags(t *testing.T, src []byte) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	blocks := ExtractBlocks(f)
	diags := Check(fset, "test.go", src, blocks)
	if len(diags) == 0 {
		t.Error("expected diagnostics, got none")
	}
}

func TestCheck_SingleImport_Ignored(t *testing.T) {
	src := []byte(`package foo

import "fmt"
`)
	assertNoDiags(t, src)
}

func TestCheck_SingleGroupedImport_Ignored(t *testing.T) {
	src := []byte(`package foo

import (
	"fmt"
)
`)
	assertNoDiags(t, src)
}

func assertNoDiags(t *testing.T, src []byte) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	blocks := ExtractBlocks(f)
	diags := Check(fset, "test.go", src, blocks)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestClassify_Std(t *testing.T) {
	assertClassify(t, GroupStd,
		"fmt", "os", "os/exec", "encoding/json", "net/http", "strings",
		"crypto/rand", "math/rand/v2")
}

// TestClassify_Internal covers the mirurobotics org, in any repo.
func TestClassify_Internal(t *testing.T) {
	assertClassify(t, GroupInternal,
		"github.com/mirurobotics/core/pkg/errs",
		"github.com/mirurobotics/core/pkg/files",
		"github.com/mirurobotics/backend/pkg/foo",
		"github.com/mirurobotics/agent/pkg/bar")
}

func TestClassify_External(t *testing.T) {
	assertClassify(t, GroupExternal,
		"github.com/go-git/go-git/v5",
		"cuelang.org/go/cue",
		"golang.org/x/text",
		"go.uber.org/zap",
		"github.com/stretchr/testify/assert")
}

// TestClassify_PrefixBoundary covers orgs that only start with "mirurobotics".
func TestClassify_PrefixBoundary(t *testing.T) {
	assertClassify(t, GroupExternal,
		"github.com/miruroboticsfork/pkg",
		"github.com/miruroboticsx/thing")
}

func assertClassify(t *testing.T, want ImportGroup, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if got := classify(path); got != want {
			t.Errorf("classify(%q) = %d, want %d", path, got, want)
		}
	}
}

func TestCheckerFixerAgreement(t *testing.T) {
	cases := []string{
		"package foo\n\nimport (\n" +
			"\t\"fmt\"\n\t\"strings\"\n\n" +
			"\t\"github.com/mirurobotics/core/pkg/errs\"\n\n" +
			"\t\"github.com/go-git/go-git/v5\"\n)\n",
		"package foo\n\nimport (\n" +
			"\t\"github.com/go-git/go-git/v5\"\n" +
			"\t\"fmt\"\n)\n",
		"package foo\n\nimport (\n" +
			"\t\"fmt\"\n" +
			"\t\"github.com/mirurobotics/core/pkg/errs\"\n)\n",
		"package foo\n\nimport (\n" +
			"\t\"fmt\"\n\n\t\"strings\"\n)\n",
		"package foo\n\nimport \"fmt\"\n",
	}
	for _, src := range cases {
		assertCheckerFixerAgree(t, []byte(src))
	}
}

func assertCheckerFixerAgree(t *testing.T, src []byte) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse error: %v\n%s", err, src)
	}
	changed := string(Fix(fset, src, ExtractBlocks(f))) != string(src)

	fsetCheck := token.NewFileSet()
	fCheck, _ := parser.ParseFile(fsetCheck, "test.go", src, parser.ParseComments)
	diags := Check(fsetCheck, "test.go", src, ExtractBlocks(fCheck))
	hasDiags := len(diags) > 0

	if changed && !hasDiags {
		t.Errorf("fixer changed file but checker reported no diagnostics:\n%s", src)
	}
	if !changed && hasDiags {
		t.Errorf("fixer left file unchanged but checker reported %v:\n%s", diags, src)
	}
}

func assertFixResult(t *testing.T, src []byte, want string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	blocks := ExtractBlocks(f)
	got := string(Fix(fset, src, blocks))
	if got != want {
		t.Errorf("fix produced unexpected result.\ngot:\n%s\nwant:\n%s", got, want)
	}
}
