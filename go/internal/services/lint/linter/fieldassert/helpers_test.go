package fieldassert

import (
	"testing"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/linttest"
)

const header = `package foo

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

`

func assertLines(t *testing.T, filename, body string, want ...int) {
	t.Helper()
	fset, f := linttest.Parse(t, filename, header+body)
	linttest.AssertLines(t, Check(fset, filename, f), want, "")
}

func assertGotFlagged(t *testing.T, body string, want ...int) {
	t.Helper()
	fset, f := linttest.Parse(t, "foo_test.go", header+body)
	diags := Check(fset, "foo_test.go", f)
	linttest.AssertLines(t, diags, want, wantMsg("got", "got"))
}

func wantMsg(base, root string) string {
	msg := "3 asserts compare fields of " + base + " one at a time;" +
		" compare the whole value in one assert"
	if root != "" {
		msg += ", or move the checks into a helper that takes " + root
	}
	return msg
}
