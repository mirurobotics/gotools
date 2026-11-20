package filelen

import (
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename string
	src      string
	maxLen   int
	wantN    int
	wantLine int
	wantMsg  string
}

func TestCheck_DisabledWhenLimitZero(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "package foo\n" + body(100),
		maxLen:   0,
		wantN:    0,
		wantLine: 0,
		wantMsg:  "",
	})
}

func TestCheck_AtExactlyLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "package foo\n" + body(4),
		maxLen:   5,
		wantN:    0,
		wantLine: 0,
		wantMsg:  "",
	})
}

func TestCheck_OverLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "package foo\n" + body(5),
		maxLen:   5,
		wantN:    1,
		wantLine: 1,
		wantMsg:  "file is 6 lines (limit 5)",
	})
}

func TestCheck_BlankAndCommentLinesNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "// Package foo does things.\npackage foo\n\n" +
			"// a comment\n \t \n" + body(1) + "\t// indented\n" +
			body(1) + "\n\n",
		maxLen:   3,
		wantN:    0,
		wantLine: 0,
		wantMsg:  "",
	})
}

func TestCheck_BlockCommentLinesNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "/*\nLicense header\nspanning lines\n*/\n\npackage foo\n" +
			"/* inline */ var x = 1\nvar y = 2 // why\n",
		maxLen:   2,
		wantN:    1,
		wantLine: 6,
		wantMsg:  "file is 3 lines (limit 2)",
	})
}

func TestCheck_CommentMarkersInsideStringsCountedAsCode(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nconst s = `\n" +
			"// not a comment\n/* nor this */\n`\n",
		maxLen:   4,
		wantN:    1,
		wantLine: 1,
		wantMsg:  "file is 5 lines (limit 4)",
	})
}

func TestCheck_CRLFBlockCommentNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "/*\r\nheader\r\n*/\r\n" +
			"package foo\r\nvar x = 1\r\n",
		maxLen:   1,
		wantN:    1,
		wantLine: 4,
		wantMsg:  "file is 2 lines (limit 1)",
	})
}

func TestCheck_BlockCommentOpeningWithSlashNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "/*/ note\nmore\n*/\npackage foo\nvar x = 1\n",
		maxLen:   1,
		wantN:    1,
		wantLine: 4,
		wantMsg:  "file is 2 lines (limit 1)",
	})
}

func TestCheck_ReportedOnPackageClauseLine(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "// Package foo does things.\npackage foo\n" + body(5),
		maxLen:   5,
		wantN:    1,
		wantLine: 2,
		wantMsg:  "file is 6 lines (limit 5)",
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src:      "package foo\n" + body(100),
		maxLen:   5,
		wantN:    1,
		wantLine: 1,
		wantMsg:  "file is 101 lines (limit 5)",
	})
}

func TestCheck_TestHelperNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "tests/helpers/foo.go",
		src:      "package helpers\n" + body(100),
		maxLen:   5,
		wantN:    1,
		wantLine: 1,
		wantMsg:  "file is 101 lines (limit 5)",
	})
}

func TestCheck_NolintFilelenAbovePackageClauseExemptsFile(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src:      "//nolint:filelen // big table\npackage foo\n" + body(100),
		maxLen:   5,
		wantN:    0,
		wantLine: 0,
		wantMsg:  "",
	})
}

func TestCheck_MiruNolintFilelenOnPackageClauseExemptsFile(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src:      "package foo //miru:nolint:filelen\n" + body(100),
		maxLen:   5,
		wantN:    0,
		wantLine: 0,
		wantMsg:  "",
	})
}

func TestCheck_FilelenInANolintListExemptsFile(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src:      "package foo //nolint:gosec,filelen // reason\n" + body(100),
		maxLen:   5,
		wantN:    0,
		wantLine: 0,
		wantMsg:  "",
	})
}

func TestCheck_RuleNameAsAPrefixDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src:      "package foo //nolint:filelenx\n" + body(100),
		maxLen:   5,
		wantN:    1,
		wantLine: 1,
		wantMsg:  "file is 101 lines (limit 5)",
	})
}

func TestCheck_NolintHigherInPackageDocCommentExemptsFile(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: "//nolint:filelen // big table\n" +
			"//go:generate stringer -type=Kind\npackage foo\n" + body(100),
		maxLen:   5,
		wantN:    0,
		wantLine: 0,
		wantMsg:  "",
	})
}

func TestCheck_NolintTwoLinesAbovePackageClauseDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src:      "//nolint:filelen\n\npackage foo\n" + body(100),
		maxLen:   5,
		wantN:    1,
		wantLine: 3,
		wantMsg:  "file is 101 lines (limit 5)",
	})
}

func TestCheck_NolintElsewhereInFileDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src:      "package foo\n//nolint:filelen\n" + body(100),
		maxLen:   5,
		wantN:    1,
		wantLine: 1,
		wantMsg:  "file is 101 lines (limit 5)",
	})
}

func TestCheck_NolintOtherRuleDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src:      "//nolint:funclen\npackage foo\n" + body(100),
		maxLen:   5,
		wantN:    1,
		wantLine: 2,
		wantMsg:  "file is 101 lines (limit 5)",
	})
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	filename := tt.filename
	if filename == "" {
		filename = "test.go"
	}
	fset, f := linttest.Parse(t, filename, tt.src)
	diags := Check(fset, filename, f, []byte(tt.src), tt.maxLen)
	linttest.AssertCount(t, diags, tt.wantN, tt.wantMsg)
	if tt.wantN > 0 && diags[0].Line != tt.wantLine {
		t.Errorf("line = %d, want %d", diags[0].Line, tt.wantLine)
	}
}

func body(n int) string { return strings.Repeat("var _ = 1\n", n) }
