package errfmt

import (
	"testing"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/linttest"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func TestCheck_LowercaseErrorString(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("something failed")`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_CapitalizedErrorString(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("Failed to connect")`,
		wantN: 1,
		wantMsg: `error string "Failed to connect"` +
			` should not be capitalized`,
	})
}

func TestCheck_TrailingPeriod(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("connection failed.")`,
		wantN: 1,
		wantMsg: `error string "connection failed."` +
			` should not end with punctuation`,
	})
}

func TestCheck_TrailingExclamation(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("connection failed!")`,
		wantN: 1,
		wantMsg: `error string "connection failed!"` +
			` should not end with punctuation`,
	})
}

func TestCheck_AcronymStartExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("HTTP request failed")`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FormatVerbStartExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "fmt"` +
			`; var e = fmt.Errorf("%s failed", "x")`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FmtErrorfLowercase(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "fmt"` +
			`; var e = fmt.Errorf("something went wrong: %w", nil)`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FmtErrorfCapitalized(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "fmt"` +
			`; var e = fmt.Errorf("Something went wrong")`,
		wantN: 1,
		wantMsg: `error string "Something went wrong"` +
			` should not be capitalized`,
	})
}

func TestCheck_JSONAcronymExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("JSON parse error")`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_CapitalizedAndTrailingPeriod(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("Bad request.")`,
		wantN:   2,
		wantMsg: "",
	})
}

func TestCheck_EmptyStringArg(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("")`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NonLiteralArg(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			"; func f(msg string) error {" +
			" return errors.New(msg) }",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_SingleCharUppercaseNotAcronym(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("I failed")`,
		wantN: 1,
		wantMsg: `error string "I failed"` +
			` should not be capitalized`,
	})
}

func TestCheck_SingleWordNoSeparator(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo; import "errors"` +
			`; var e = errors.New("Word")`,
		wantN: 1,
		wantMsg: `error string "Word"` +
			` should not be capitalized`,
	})
}

func TestCheck_BareCallNotErrorsOrFmt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo` +
			`; func foo(s string) {}` +
			`; func f() { foo("Test") }`,
		wantN:   0,
		wantMsg: "",
	})
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, "test.go", tt.src)
	linttest.AssertCount(t, Check(fset, "test.go", f), tt.wantN, tt.wantMsg)
}
