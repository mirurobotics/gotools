package pkgname

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func TestCheck_CleanName(t *testing.T) {
	runCheckCase(t, checkCase{src: `package foo`, wantN: 0, wantMsg: ""})
}

func TestCheck_CamelCaseName(t *testing.T) {
	runCheckCase(t, checkCase{src: `package fooBar`, wantN: 0, wantMsg: ""})
}

func TestCheck_TestPrefixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{src: `package test_helpers`, wantN: 0, wantMsg: ""})
}

func TestCheck_MockPrefixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{src: `package mock_service`, wantN: 0, wantMsg: ""})
}

func TestCheck_TestSuffixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{src: `package foo_test`, wantN: 0, wantMsg: ""})
}

func TestCheck_UnderscoreInMiddleRejected(t *testing.T) {
	runCheckCase(t, checkCase{
		src:   `package my_package`,
		wantN: 1,
		wantMsg: "package name \"my_package\" contains" +
			" underscore; use camelCase instead",
	})
}

func TestCheck_MultipleUnderscoresRejected(t *testing.T) {
	runCheckCase(t, checkCase{
		src:   `package my_cool_package`,
		wantN: 1,
		wantMsg: "package name \"my_cool_package\" contains" +
			" underscore; use camelCase instead",
	})
}

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	fset, file := linttest.Parse(t, "test.go", tc.src)
	linttest.AssertCount(t, Check(fset, "test.go", file), tc.wantN, tc.wantMsg)
}
