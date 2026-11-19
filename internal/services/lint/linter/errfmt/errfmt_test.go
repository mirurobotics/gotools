package errfmt

import (
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diags := Check(fset, "test.go", f)
	if len(diags) != tt.wantN {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
	}
	if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
		t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
	}
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
