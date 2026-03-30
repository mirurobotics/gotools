package imports

import (
	"go/parser"
	"go/token"
	"testing"
)

// --- classifier tests ---

func TestClassify(t *testing.T) {
	tests := []struct {
		path string
		want ImportGroup
	}{
		// Standard library
		{"fmt", GroupStd},
		{"os", GroupStd},
		{"os/exec", GroupStd},
		{"encoding/json", GroupStd},
		{"net/http", GroupStd},
		{"strings", GroupStd},
		{"crypto/rand", GroupStd},
		{"math/rand/v2", GroupStd},

		// Internal (mirurobotics org — any repo)
		{"github.com/mirurobotics/core/pkg/errs", GroupInternal},
		{"github.com/mirurobotics/core/pkg/files", GroupInternal},
		{"github.com/mirurobotics/backend/pkg/foo", GroupInternal},
		{"github.com/mirurobotics/agent/pkg/bar", GroupInternal},

		// External
		{"github.com/go-git/go-git/v5", GroupExternal},
		{"cuelang.org/go/cue", GroupExternal},
		{"golang.org/x/text", GroupExternal},
		{"go.uber.org/zap", GroupExternal},
		{"github.com/stretchr/testify/assert", GroupExternal},

		// Prefix boundary: must NOT match orgs that start with "mirurobotics"
		{"github.com/miruroboticsfork/pkg", GroupExternal},
		{"github.com/miruroboticsx/thing", GroupExternal},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := classify(tt.path)
			if got != tt.want {
				t.Errorf("classify(%q) = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

// --- checker tests ---

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

// --- fixer tests ---

func TestFix_ReordersAndGroups(t *testing.T) {
	src := []byte(`package foo

import (
	"github.com/go-git/go-git/v5"
	"fmt"
	"github.com/mirurobotics/core/pkg/errs"
	"strings"
)
`)
	want := `package foo

import (
	"fmt"
	"strings"

	"github.com/mirurobotics/core/pkg/errs"

	"github.com/go-git/go-git/v5"
)
`
	assertFixResult(t, src, want)
}

func TestFix_PreservesAliases(t *testing.T) {
	src := []byte(`package foo

import (
	errs "github.com/mirurobotics/core/pkg/errs"
	"fmt"
)
`)
	want := `package foo

import (
	"fmt"

	errs "github.com/mirurobotics/core/pkg/errs"
)
`
	assertFixResult(t, src, want)
}

func TestFix_AddsMissingSeparator(t *testing.T) {
	src := []byte(`package foo

import (
	"fmt"
	"github.com/mirurobotics/core/pkg/errs"
)
`)
	want := `package foo

import (
	"fmt"

	"github.com/mirurobotics/core/pkg/errs"
)
`
	assertFixResult(t, src, want)
}

func TestFix_RemovesExtraBlankLineWithinGroup(t *testing.T) {
	src := []byte(`package foo

import (
	"github.com/mirurobotics/core/pkg/errs"

	"github.com/mirurobotics/core/pkg/files"
)
`)
	want := `package foo

import (
	"github.com/mirurobotics/core/pkg/errs"
	"github.com/mirurobotics/core/pkg/files"
)
`
	assertFixResult(t, src, want)
}

func TestFix_Idempotent(t *testing.T) {
	src := []byte(`package foo

import (
	"fmt"
	"strings"

	"github.com/mirurobotics/core/pkg/errs"

	"github.com/go-git/go-git/v5"
)
`)
	assertFixResult(t, src, string(src))
}

func TestFix_SingleGroupOnly(t *testing.T) {
	src := []byte(`package foo

import (
	"strings"
	"fmt"
)
`)
	want := `package foo

import (
	"fmt"
	"strings"
)
`
	assertFixResult(t, src, want)
}

func TestFix_TwoGroupsOnly(t *testing.T) {
	src := []byte(`package foo

import (
	"github.com/mirurobotics/core/pkg/errs"
	"fmt"
)
`)
	want := `package foo

import (
	"fmt"

	"github.com/mirurobotics/core/pkg/errs"
)
`
	assertFixResult(t, src, want)
}

func TestFix_SingleImport_Unchanged(t *testing.T) {
	src := []byte(`package foo

import "fmt"
`)
	assertFixResult(t, src, string(src))
}

func TestFix_DotImport(t *testing.T) {
	src := []byte(`package foo

import (
	. "github.com/mirurobotics/core/pkg/assert"
	"fmt"
)
`)
	want := `package foo

import (
	"fmt"

	. "github.com/mirurobotics/core/pkg/assert"
)
`
	assertFixResult(t, src, want)
}

func TestFix_AllThreeGroups_Reversed(t *testing.T) {
	src := []byte(`package foo

import (
	"github.com/stretchr/testify/assert"
	"github.com/mirurobotics/core/pkg/errs"
	"fmt"
)
`)
	want := `package foo

import (
	"fmt"

	"github.com/mirurobotics/core/pkg/errs"

	"github.com/stretchr/testify/assert"
)
`
	assertFixResult(t, src, want)
}

func TestFix_PreservesCodeAfterImports(t *testing.T) {
	src := []byte(`package foo

import (
	"github.com/go-git/go-git/v5"
	"fmt"
)

func main() {
	fmt.Println("hello")
}
`)
	want := `package foo

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

func main() {
	fmt.Println("hello")
}
`
	assertFixResult(t, src, want)
}

func TestFix_MultipleAliases(t *testing.T) {
	src := []byte(`package foo

import (
	jsonx "github.com/mirurobotics/core/pkg/json"
	errs "github.com/mirurobotics/core/pkg/errs"
	"fmt"
)
`)
	want := `package foo

import (
	"fmt"

	errs "github.com/mirurobotics/core/pkg/errs"
	jsonx "github.com/mirurobotics/core/pkg/json"
)
`
	assertFixResult(t, src, want)
}

func TestFix_NoBlocks(t *testing.T) {
	src := []byte(`package foo

func main() {}
`)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	blocks := ExtractBlocks(f)
	got := string(Fix(fset, src, blocks))
	if got != string(src) {
		t.Errorf("expected no change, got:\n%s", got)
	}
}

// --- checker/fixer agreement ---

func TestCheckerFixerAgreement(t *testing.T) {
	cases := [][]byte{
		[]byte(
			"package foo\n\nimport (\n" +
				"\t\"fmt\"\n\t\"strings\"\n\n" +
				"\t\"github.com/mirurobotics/core/pkg/errs\"\n\n" +
				"\t\"github.com/go-git/go-git/v5\"\n)\n",
		),
		[]byte(
			"package foo\n\nimport (\n" +
				"\t\"github.com/go-git/go-git/v5\"\n" +
				"\t\"fmt\"\n)\n",
		),
		[]byte(
			"package foo\n\nimport (\n" +
				"\t\"fmt\"\n" +
				"\t\"github.com/mirurobotics/core/pkg/errs\"\n)\n",
		),
		[]byte(
			"package foo\n\nimport (\n" +
				"\t\"fmt\"\n\n\t\"strings\"\n)\n",
		),
		[]byte("package foo\n\nimport \"fmt\"\n"),
	}

	for i, src := range cases {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
		if err != nil {
			t.Fatalf("case %d: parse error: %v", i, err)
		}

		blocks := ExtractBlocks(f)
		fixed := Fix(fset, src, blocks)
		changed := string(fixed) != string(src)

		fsetCheck := token.NewFileSet()
		fCheck, _ := parser.ParseFile(fsetCheck, "test.go", src, parser.ParseComments)
		blocksCheck := ExtractBlocks(fCheck)
		diags := Check(fsetCheck, "test.go", src, blocksCheck)
		hasDiags := len(diags) > 0

		if changed && !hasDiags {
			t.Errorf(
				"case %d: fixer changed file but checker reported no diagnostics",
				i,
			)
		}
		if !changed && hasDiags {
			t.Errorf(
				"case %d: fixer left file unchanged but checker reported diagnostics: %v",
				i, diags,
			)
		}
	}
}

// --- helpers ---

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
