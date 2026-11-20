package todoissue

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	src       string
	wantLines []int
}

func TestCheck_TodoWithoutLinkFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// TODO: handle retries
func f() {}

func g() {
	// TODO(ben): clean up
}
`,
		wantLines: []int{3, 7},
	})
}

func TestCheck_TodoWithLinkOnSameOrContinuationLineAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// TODO: handle retries https://github.com/org/repo/issues/1
func f() {}

// F does things.
// TODO(ben): clean up once the API settles.
// See (https://linear.app/org/issue/ENG-1).
func F() {}

/* TODO: block https://example.com/issues/2 */
var v = 0

// TODO(https://github.com/org/repo/issues/12): fix
// TODO:HTTPS://github.com/org/repo/issues/13
// TODO: see http:old|https://example.com/issues/14.
var x = 1

/*
 * TODO: multiline block
 * http://example.com/issues/3
 */
var y = 2
`,
		wantLines: nil,
	})
}

func TestCheck_MiruPrefixedNolintSuppressesAGroup(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// TODO: known gap
//miru:nolint:todoissue // tracked offline
var n = 2
`,
		wantLines: nil,
	})
}

func TestCheck_LinkBelongsToALaterTodoOnly(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// TODO: first
// TODO: second https://github.com/org/repo/issues/4
// TODO: third
func f() {}
`,
		wantLines: []int{3, 5},
	})
}

func TestCheck_NonTodoCommentsWordsAndDirectivesSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// TODOS are tracked elsewhere.
// Returns context.TODO when unset.
// TODO_LIST is a name.
//nolint:all // see the TODO list
func f() {}

/* TODO */
var z = 3

// TODO: support https:// URLs
// and http://localhost too.
var w = 4
`,
		wantLines: []int{9, 12},
	})
}

func TestCheck_StarPrefixedAndInnerBlockLinesFlaggedAtTheirOwnLine(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

/*
 * TODO: fix
 */
var a = 1

/*
notes
TODO: fix
*/
var b = 2
`,
		wantLines: []int{4, 10},
	})
}

func TestCheck_LinksInDirectivesOrOtherGroupsDoNotCount(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// TODO: fix
//go:generate curl https://example.com/issues/5
func f() {}

// TODO: unrelated

// See https://example.com/issues/6.
func g() {}
`,
		wantLines: []int{3, 7},
	})
}

func TestCheck_NolintExplanationsAreChecked(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

//nolint:gosec // TODO: remove once fixed
var c = 1

//nolint:gosec // TODO: remove https://example.com/issues/7
var d = 2
`,
		wantLines: []int{3},
	})
}

func TestCheck_NosecNotesCheckedAndBlankLinesEndATodo(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// #nosec G304 -- TODO: validate path
var e = 1

// TODO: handle retries.
//
// See https://pkg.go.dev/net/http.
var g = 2
`,
		wantLines: []int{3, 6},
	})
}

func TestCheck_DirectiveNotesAreReadPerKind(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

//lint:ignore SA1000 TODO: fix
var h = 1

// TODO: regenerate
//go:generate go run ./gen -- https://example.com/spec
var i = 2

//lint:ignore SA1000
var j = 3

//lint:file-ignore SA1000 TODO: fix
var k = 4

// TODO: x
//lint:ignore SA1000
// https://example.com/issues/1
var l = 5
`,
		wantLines: []int{3, 6, 13},
	})
}

func TestCheck_ListMarkerTodosCheckedAndNolintSuppressesAGroup(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// Plan:
// - TODO: split this up
var m = 1

// TODO: known gap
//nolint:todoissue // tracked offline
var n = 2

//nolint:gosec, todoissue // TODO: known gap
var o = 3

// TODO: drop nolint:todoissue below
var p = 4
`,
		wantLines: []int{4, 14},
	})
}

func TestCheck_TodoWithNoSpaceAfterSlashesIsNotADirective(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

//TODO:fix
var q = 1
`,
		wantLines: []int{3},
	})
}

func TestCheck_ProseStartingWithTODOPunctuationIsNotATodo(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// Unlike
// TODO, it signals intent.
// TODO. Not a marker either.
// TODO fix the retry loop
var r = 1
`,
		wantLines: []int{6},
	})
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, "foo.go", tt.src)
	diags := Check(fset, "foo.go", f)
	linttest.AssertLines(t, diags, tt.wantLines, message)
	for _, d := range diags {
		if d.File != "foo.go" {
			t.Errorf("diag file = %q, want foo.go", d.File)
		}
	}
}
