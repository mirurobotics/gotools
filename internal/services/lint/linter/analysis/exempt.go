package analysis

import (
	"bytes"
	"go/ast"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsGeneratedFile reports whether a Code generated marker precedes the package line.
func IsGeneratedFile(src []byte) bool {
	for _, line := range bytes.Split(src, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if bytes.Contains(trimmed, []byte("// Code generated")) {
			return true
		}
		if bytes.HasPrefix(trimmed, []byte("package ")) {
			return false
		}
	}
	return false
}

// IsLintSource returns true if the file lives under the linter's own source tree.
func IsLintSource(filename string) bool {
	return strings.Contains(filename, "services/lint/")
}

// IsCmdSource returns true if the file lives under a cmd/ directory.
func IsCmdSource(filename string) bool { return strings.Contains(filename, "cmd/") }

// IsExampleFunc reports whether decl is a go test Example function in a test file.
func IsExampleFunc(filename string, decl *ast.FuncDecl) bool {
	if decl.Recv != nil || !IsTestFile(filename) {
		return false
	}
	return isTestName(decl.Name.Name, "Example")
}

// IsTestFile returns true if the filename ends with _test.go.
func IsTestFile(filename string) bool { return strings.HasSuffix(filename, "_test.go") }

// IsTestFunc reports whether decl is a go test Test, Benchmark, Fuzz or Example func.
func IsTestFunc(decl *ast.FuncDecl) bool {
	if decl.Recv != nil {
		return false
	}
	for _, prefix := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
		if isTestName(decl.Name.Name, prefix) {
			return true
		}
	}
	return false
}

func isTestName(name, prefix string) bool {
	suffix, ok := strings.CutPrefix(name, prefix)
	if !ok || suffix == "" {
		return ok
	}
	char, _ := utf8.DecodeRuneInString(suffix)
	return !unicode.IsLower(char)
}
