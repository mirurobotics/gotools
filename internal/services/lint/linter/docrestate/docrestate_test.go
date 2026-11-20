package docrestate

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

const restateMsg = "doc comment for Run only restates its name; " +
	"say what it does or remove it"

func TestCheck_FuncDocRestatesName(t *testing.T) {
	src := "package foo\n\n// Run runs.\nfunc Run() {}\n"
	assertCheck(t, src, restateMsg, 3)
}

func TestCheck_FuncDocSaysSomething(t *testing.T) {
	src := `package foo

// Run starts the worker pool and blocks until ctx is done.
func Run() {}
`
	assertCheck(t, src, "")
}

func TestCheck_ConstructorAndGetterDocsFlagged(t *testing.T) {
	src := `package foo

// NewClient creates a new client.
func NewClient() {}

// GetUserID gets the user ID.
func GetUserID() {}

// UserIDs returns the user IDs.
func UserIDs() {}

// Close closes it.
func (c *C) Close() {}

type Closer interface {
	// Close closes.
	Close()
}
`
	assertCheck(t, src, "", 3, 6, 9, 12, 16)
}

func TestCheck_TypeValueAndFieldDocsFlagged(t *testing.T) {
	src := `package foo

// Client is a client.
type Client struct {
	// Name is the name.
	Name string
	// Retries counts how often a request is retried.
	Retries int
}

const (
	// MaxSize is the max size.
	MaxSize = 1
)

// Timeout.
var Timeout = 1
`
	assertCheck(t, src, "", 3, 5, 12, 16)
}

func TestCheck_GroupedAndMultiNameDeclarations(t *testing.T) {
	src := `package foo

// A is a.
var A, B = 1, 2

type (
	// Server is a server.
	Server struct{}
)

// X is x.
var (
	X = 1
)
`
	assertCheck(t, src, "", 7)
}

func TestCheck_DirectivesIgnoredAndBlockCommentsRead(t *testing.T) {
	src := `package foo

// Run runs.
//
//nolint:funclen // reason
func Run() {}

/* Stop stops. */
func Stop() {}
`
	assertCheck(t, src, "", 3, 8)
}

func TestCheck_FullCompoundNameRepeated(t *testing.T) {
	src := `package foo

// ClientConfig is a ClientConfig.
type ClientConfig struct{}

// NewClient creates a new NewClient.
func NewClient() {}
`
	assertCheck(t, src, "", 3, 6)
}

func TestCheck_QuotedValuesAreMeaningful(t *testing.T) {
	src := `package foo

// Separator is "/".
const Separator = "/"

// Tick is ` + "`" + `tick` + "`" + `.
const Tick = 1

// Letter is 'a'.
const Letter = 'a'

// Initial is ('a').
const Initial = 'a'

// MaxSize is 'max size'.
const MaxSize = 1

// Name is "name".
const Name = "name"
`
	assertCheck(t, src, "")
}

func TestCheck_SymbolsAreMeaningful(t *testing.T) {
	src := `package foo

// Plus returns +.
func Plus() {}

// Dash is - the dash.
const Dash = 1

// Set is {'a'}.
var Set = 1

// Empty returns [].
func Empty() {}

// LParen is (.
const LParen = 1
`
	assertCheck(t, src, "")
}

func TestCheck_DocNotStartingWithNameSkipped(t *testing.T) {
	src := "package foo\n\n// runs the run.\nfunc Run() {}\n\nfunc NoDoc() {}\n"
	assertCheck(t, src, "")
}

func TestCheck_NolintDirectiveSkipped(t *testing.T) {
	src := `package foo

// Run runs.
//
//nolint:docrestate // name says it all
func Run() {}
`
	assertCheck(t, src, "")
}

func TestCheck_BlankIdentifierSkipped(t *testing.T) {
	assertCheck(t, "package foo\n\n// _ is _.\nvar _ = 1\n", "")
}

func TestCheck_MultiNameFieldSkipped(t *testing.T) {
	src := `package foo

type Pair struct {
	// A is a.
	A, B int
}
`
	assertCheck(t, src, "")
}

func TestCheck_DeclarationInFunctionBodySkipped(t *testing.T) {
	src := `package foo

func Local() {
	// count is the count.
	var count int
	_ = count
}
`
	assertCheck(t, src, "")
}

func TestCheck_PluralNameWithSingularWordFlagged(t *testing.T) {
	src := "package foo\n\n// Settings is the setting.\nvar Settings = 1\n"
	assertCheck(t, src, "", 3)
}

func TestCheck_ShortNameWordTakesNoIngForm(t *testing.T) {
	assertCheck(t, "package foo\n\n// Str returns the string.\nfunc Str() {}\n", "")
}

func TestCheck_StarredBlockCommentFlagged(t *testing.T) {
	src := "package foo\n\n/*\n * Run runs.\n */\nfunc Run() {}\n"
	assertCheck(t, src, restateMsg, 3)
}

func assertCheck(t *testing.T, src, wantMsg string, wantLines ...int) {
	t.Helper()
	fset, f := linttest.Parse(t, "foo.go", src)
	linttest.AssertLines(t, Check(fset, "foo.go", f), wantLines, wantMsg)
}
