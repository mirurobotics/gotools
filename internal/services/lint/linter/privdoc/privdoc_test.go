package privdoc

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	src       string
	wantLines []int
}

func TestCheck_UnexportedFuncAndMethodDocsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// f does a thing.
func f() {}

// m does a thing.
func (T) m() {}

// M is exported on an unexported type.
func (*t) M() {}

// G is generic on an unexported type.
func (g[K]) G() {}
`,
		wantLines: []int{3, 6, 9, 12},
	})
}

func TestCheck_LocalDeclarationsLeftToFunccomment(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func F() {
	// x is local.
	type x int
	_ = x(0)
}
`,
		wantLines: nil,
	})
}

func TestCheck_ExportedDocsAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// F does a thing.
func F() {}

// M does a thing.
func (*T) M() {}

// T is a type.
type T struct {
	// A is a field.
	A int // trailing
	// Embedded is embedded.
	Embedded
	// Other is embedded from a package.
	*pkg.Other
	// Generic is a generic embed.
	Generic[int]
	// Pair is a generic embed with two arguments.
	Pair[int, string]
	// Inner is an exported nested struct.
	Inner struct {
		// B is a field.
		B int
	}
}

// I is an interface.
type I interface {
	// Do does.
	Do()
}
`,
		wantLines: nil,
	})
}

func TestCheck_ExportedValueDocsAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// C is a constant.
const C = 1

// Values mixes exported and unexported names.
var (
	// V is exported.
	V, v = 1, 2
)

// Group has an exported spec and an unexported one.
var (
	A = 1
	b = 2
)

// State is an exported var with a struct type.
var State struct {
	// Count is exported.
	Count int
}

// m is a method on an unknown receiver.
func (*[2]T) M() {}
`,
		wantLines: nil,
	})
}

func TestCheck_UnexportedTypesAndFieldsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// t is a type.
type t struct {
	// A is on an unexported type.
	A int
}

type T struct {
	a int // trailing
	// embedded is embedded.
	embedded
	// Inner is exported.
	Inner struct {
		// b is unexported.
		b int
	}
	hidden struct {
		// B is in an unexported field.
		B int
	}
	Items []struct {
		c int // in a slice
	}
	Ptr *struct {
		d int // in a pointer
	}
	Map map[string]struct {
		e int // in a map
	}
	Ch chan struct {
		f int // in a chan
	}
}
`,
		wantLines: []int{3, 5, 10, 11, 15, 19, 23, 26, 29, 32},
	})
}

func TestCheck_UnexportedValuesFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type I interface {
	// do does.
	do()
}

// c is a constant.
const c = 1

// group of unexported values.
var (
	// x is unexported.
	x = 1
	y = 2 // trailing
)
`,
		wantLines: []int{4, 8, 11, 13, 15},
	})
}

func TestCheck_ProseMixedWithDirectivesFlaggedOnTheProseLine(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// f does a thing.
//
//nolint:funclen // reason
func f() {}

//nolint:funclen // reason
// g does a thing.
func g() {}

var state struct {
	count int // hits
}

var cfg = struct {
	port int // listen port
}{8080}

type K map[struct {
	k int // key field
}]int
`,
		wantLines: []int{3, 9, 13, 17, 21},
	})
}

func TestCheck_DirectivesAndBlankLinesAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

//nolint:funclen // reason
func f() {}

//go:noinline
//nolint:funclen // reason
func g() {}

//export callback
func callback() {}

//nolint:privdoc // why h exists
// h keeps a longer note.
func h() {}

//
//go:generate echo hi
type t int

var x = 1 //nolint:gochecknoglobals
`,
		wantLines: nil,
	})
}

func TestCheck_GroupedImportsAllowedParamsLeftToStraycomment(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

import (
	// fmt is imported.
	"fmt" // for printing
)

func F(
	// a is documented.
	a int,
) {
}

func f(
	// a is documented.
	a int,
) {
}

type G[
	// K is documented.
	K any,
] struct{}

type g[
	// K is documented.
	K any,
] struct{}
`,
		wantLines: nil,
	})
}

func TestCheck_InlineConstraintsFollowTheirType(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Set[T interface {
	// Key returns the key.
	Key() string
}] struct{}

type set[T interface {
	// Key returns the key.
	Key() string
}] struct{}
`,
		wantLines: []int{9},
	})
}

func TestCheck_ImportsIgnored(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// fmt is imported.
import "fmt"
`,
		wantLines: nil,
	})
}

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	fset, file := linttest.Parse(t, "foo.go", tc.src)
	linttest.AssertLines(t, Check(fset, "foo.go", file), tc.wantLines, message)
}
