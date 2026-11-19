package importalias

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	filename := tt.filename
	if filename == "" {
		filename = "test.go"
	}
	fset, f := linttest.Parse(t, filename, tt.src)
	linttest.AssertCount(t, Check(fset, filename, f), tt.wantN, tt.wantMsg)
}

func TestCheck_NoAlias(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import "fmt"`,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_AliasWithoutUnderscore(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import errs "errors"`,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_UnderscoreAliasRejected(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import my_errs "errors"`,
		wantN:    1,
		wantMsg: `import alias "my_errs" contains` +
			` underscore; use camelCase instead`,
	})
}

func TestCheck_MultipleUnderscoresRejected(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import my_cool_pkg "errors"`,
		wantN:    1,
		wantMsg: `import alias "my_cool_pkg" contains` +
			` underscore; use camelCase instead`,
	})
}

func TestCheck_TestPrefixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import test_helpers "errors"`,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_MockPrefixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import mock_service "errors"`,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_DotImportIgnored(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import . "errors"`,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_BlankImportIgnored(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import _ "errors"`,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_UnderscorePrefixAliasRejected(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      `package foo; import _foo "errors"`,
		wantN:    1,
		wantMsg: `import alias "_foo" contains` +
			` underscore; use camelCase instead`,
	})
}

func TestCheck_MixedGroup(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo

import (
	"fmt"
	my_errs "errors"
	mock_db "database/sql"
)`,
		wantN: 1,
		wantMsg: `import alias "my_errs" contains` +
			` underscore; use camelCase instead`,
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src:      `package foo; import my_errs "errors"`,
		wantN:    1,
		wantMsg:  `import alias "my_errs" contains underscore; use camelCase instead`,
	})
}

func TestCheck_MultipleViolations(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo

import (
	my_errs "errors"
	my_fmt "fmt"
)`,
		wantN:   2,
		wantMsg: "",
	})
}

func TestCheck_NonImportDeclarationsIgnored(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo

import my_fmt "fmt"

var x_y = my_fmt.Sprint()`,
		wantN:   1,
		wantMsg: `import alias "my_fmt" contains underscore; use camelCase instead`,
	})
}
