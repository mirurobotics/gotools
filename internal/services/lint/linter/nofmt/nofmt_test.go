package nofmt

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		wantN    int
		wantMsg  string
	}{
		{
			name:     "fmt.Println in library",
			filename: "pkg/foo/bar.go",
			src: `package foo; import "fmt"` +
				`; func f() { fmt.Println("hi") }`,
			wantN: 1,
			wantMsg: "fmt.Println should not be used" +
				" outside main/test;" +
				" use structured logging",
		},
		{
			name:     "fmt.Printf in library",
			filename: "pkg/foo/bar.go",
			src: `package foo; import "fmt"` +
				`; func f() { fmt.Printf("%s", "hi") }`,
			wantN: 1,
			wantMsg: "fmt.Printf should not be used" +
				" outside main/test;" +
				" use structured logging",
		},
		{
			name:     "fmt.Println in package main",
			filename: "main.go",
			src: `package main; import "fmt"` +
				`; func run() { fmt.Println("hi") }`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "println builtin in library",
			filename: "pkg/foo/bar.go",
			src: `package foo` +
				`; func f() { println("hi") }`,
			wantN: 1,
			wantMsg: "println should not be used" +
				" outside main/test;" +
				" use structured logging",
		},
		{
			name:     "print builtin in library",
			filename: "pkg/foo/bar.go",
			src: `package foo` +
				`; func f() { print("hi") }`,
			wantN: 1,
			wantMsg: "print should not be used" +
				" outside main/test;" +
				" use structured logging",
		},
		{
			name:     "log.Printf in library",
			filename: "pkg/foo/bar.go",
			src: `package foo; import "log"` +
				`; func f() { log.Printf("hi") }`,
			wantN: 1,
			wantMsg: "log.Printf should not be used" +
				" outside main/test;" +
				" use structured logging",
		},
		{
			name:     "log.Fatal in library",
			filename: "pkg/foo/bar.go",
			src: `package foo; import "log"` +
				`; func f() { log.Fatal("hi") }`,
			wantN: 1,
			wantMsg: "log.Fatal should not be used" +
				" outside main/test;" +
				" use structured logging",
		},
		{
			name:     "test file exempt",
			filename: "pkg/foo/bar_test.go",
			src: `package foo; import "fmt"` +
				`; func f() { fmt.Println("hi") }`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "tools/lint exempt",
			filename: "tools/lint/linter/check.go",
			src: `package linter; import "fmt"` +
				`; func f() { fmt.Println("hi") }`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "cmd directory exempt",
			filename: "cmd/server/run.go",
			src: `package server; import "fmt"` +
				`; func f() { fmt.Println("hi") }`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "function literal call no diagnostic",
			filename: "pkg/foo/bar.go",
			src:      `package foo; func f() { func() {}() }`,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name:     "fmt.Fprintf not banned",
			filename: "pkg/foo/bar.go",
			src: `package foo; import "fmt"` +
				`; func f() { fmt.Fprintf(nil, "x") }`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint:nofmt exempts call",
			filename: "pkg/foo/bar.go",
			src: "package foo\nimport \"fmt\"\n" +
				"func f() { fmt.Println(\"hi\") " +
				"//nolint:nofmt\n}",
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint on different line does not exempt",
			filename: "pkg/foo/bar.go",
			src: "package foo\nimport \"fmt\"\n" +
				"//nolint:nofmt\n" +
				"func f() { fmt.Println(\"hi\") }",
			wantN: 1,
			wantMsg: "fmt.Println should not be used" +
				" outside main/test;" +
				" use structured logging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tt.filename, tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, tt.filename, f)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
