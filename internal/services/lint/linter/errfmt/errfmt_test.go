package errfmt

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantN   int
		wantMsg string
	}{
		{
			name: "lowercase error string",
			src: `package foo; import "errors"` +
				`; var e = errors.New("something failed")`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "capitalized error string",
			src: `package foo; import "errors"` +
				`; var e = errors.New("Failed to connect")`,
			wantN: 1,
			wantMsg: `error string "Failed to connect"` +
				` should not be capitalized`,
		},
		{
			name: "trailing period",
			src: `package foo; import "errors"` +
				`; var e = errors.New("connection failed.")`,
			wantN: 1,
			wantMsg: `error string "connection failed."` +
				` should not end with punctuation`,
		},
		{
			name: "trailing exclamation",
			src: `package foo; import "errors"` +
				`; var e = errors.New("connection failed!")`,
			wantN: 1,
			wantMsg: `error string "connection failed!"` +
				` should not end with punctuation`,
		},
		{
			name: "acronym start exempt",
			src: `package foo; import "errors"` +
				`; var e = errors.New("HTTP request failed")`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "format verb start exempt",
			src: `package foo; import "fmt"` +
				`; var e = fmt.Errorf("%s failed", "x")`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "fmt.Errorf lowercase",
			src: `package foo; import "fmt"` +
				`; var e = fmt.Errorf("something went wrong: %w", nil)`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "fmt.Errorf capitalized",
			src: `package foo; import "fmt"` +
				`; var e = fmt.Errorf("Something went wrong")`,
			wantN: 1,
			wantMsg: `error string "Something went wrong"` +
				` should not be capitalized`,
		},
		{
			name: "JSON acronym exempt",
			src: `package foo; import "errors"` +
				`; var e = errors.New("JSON parse error")`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "capitalized and trailing period",
			src: `package foo; import "errors"` +
				`; var e = errors.New("Bad request.")`,
			wantN:   2,
			wantMsg: "",
		},
		{
			name: "empty string arg",
			src: `package foo; import "errors"` +
				`; var e = errors.New("")`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "non-literal arg",
			src: `package foo; import "errors"` +
				"; func f(msg string) error {" +
				" return errors.New(msg) }",
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "single-char uppercase not acronym",
			src: `package foo; import "errors"` +
				`; var e = errors.New("I failed")`,
			wantN: 1,
			wantMsg: `error string "I failed"` +
				` should not be capitalized`,
		},
		{
			name: "single word no separator",
			src: `package foo; import "errors"` +
				`; var e = errors.New("Word")`,
			wantN: 1,
			wantMsg: `error string "Word"` +
				` should not be capitalized`,
		},
		{
			name: "bare call not errors or fmt",
			src: `package foo` +
				`; func foo(s string) {}` +
				`; func f() { foo("Test") }`,
			wantN:   0,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}
