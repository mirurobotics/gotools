package imports

import (
	"go/parser"
	"go/token"
	"testing"
)

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
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	blocks := ExtractBlocks(file)
	got := string(Fix(fset, src, blocks))
	if got != string(src) {
		t.Errorf("expected no change, got:\n%s", got)
	}
}
