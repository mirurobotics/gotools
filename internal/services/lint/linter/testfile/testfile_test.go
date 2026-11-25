package testfile

import (
	"os"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
	"github.com/mirurobotics/gotools/internal/testutil"
)

const pkgSrc = "package foo\n"

func TestCheck(t *testing.T) {
	testutil.MakePkgDirs(t, "foo/sub", "internal/foo", "tests/foo")
	writeSources(t, "foo/bar.go", "foo/qux_test.go")
	writeSources(t, "internal/foo/bar.go", "tests/foo/own.go")
	tests := []struct {
		name      string
		filename  string
		src       string
		srcPrefix string
		testDir   string
		wantLines []int
		wantMsg   string
	}{
		{"non-test file", "gone/bar.go", pkgSrc, "", "", nil, ""},
		{"stem equals source", "foo/bar_test.go", pkgSrc, "", "", nil, ""},
		{"stem extends source", "foo/bar_extra_test.go", pkgSrc, "", "", nil, ""},
		{
			"no matching source",
			"foo/baz_test.go",
			pkgSrc,
			"",
			"",
			[]int{1},
			"no source file in foo matches baz_test.go",
		},
		{
			"stem extends source without underscore",
			"foo/barbaz_test.go",
			pkgSrc,
			"",
			"",
			[]int{1},
			"no source file in foo matches barbaz_test.go",
		},
		{
			"test file is not a source",
			"foo/qux_test_util_test.go",
			pkgSrc,
			"",
			"",
			[]int{1},
			"no source file in foo matches qux_test_util_test.go",
		},
		{
			"directory is not a source",
			"foo/sub_test.go",
			pkgSrc,
			"",
			"",
			[]int{1},
			"no source file in foo matches sub_test.go",
		},
		{
			"package nolint",
			"foo/baz_test.go",
			"//nolint:testfile\npackage foo\n",
			"",
			"",
			nil,
			"",
		},
		{
			"reported on package clause",
			"foo/baz_test.go",
			"// Package foo.\npackage foo\n",
			"",
			"",
			[]int{2},
			"no source file in foo matches baz_test.go",
		},
		{
			"mirrored source",
			"tests/foo/bar_test.go",
			pkgSrc,
			"internal",
			"tests",
			nil,
			"",
		},
		{
			"mirror ignores own dir",
			"tests/foo/own_test.go",
			pkgSrc,
			"internal",
			"tests",
			[]int{1},
			"no source file in internal/foo matches own_test.go",
		},
		{
			"mirror package missing",
			"tests/gone/bar_test.go",
			pkgSrc,
			"internal",
			"tests",
			[]int{1},
			"source package internal/gone for bar_test.go is missing",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset, file := linttest.Parse(t, tc.filename, tc.src)
			diags := Check(fset, tc.filename, file, tc.srcPrefix, tc.testDir)
			linttest.AssertLines(t, diags, tc.wantLines, tc.wantMsg)
		})
	}
}

func writeSources(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.WriteFile(path, []byte(pkgSrc), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceDir(t *testing.T) {
	tests := []struct {
		name        string
		testFileDir string
		srcPrefix   string
		testDir     string
		want        string
	}{
		{"no test-dir", "tests/foo", "internal", "", "tests/foo"},
		{"no src-prefix", "tests/foo", "", "tests", "tests/foo"},
		{"under test-dir", "tests/foo/bar", "internal", "tests", "internal/foo/bar"},
		{"test-dir root", "tests", "internal", "tests", "internal"},
		{"unclean test-dir", "tests/foo", "internal", "./tests/", "internal/foo"},
		{"outside test-dir", "cmd/foo", "internal", "tests", "cmd/foo"},
		{"test-dir name prefix", "testsuite/foo", "internal", "tests", "testsuite/foo"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sourceDir(tc.testFileDir, tc.srcPrefix, tc.testDir)
			if got != tc.want {
				t.Errorf("sourceDir = %q, want %q", got, tc.want)
			}
		})
	}
}
