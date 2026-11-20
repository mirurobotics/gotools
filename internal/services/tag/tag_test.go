package tag

import (
	"fmt"
	"io"
	"regexp"
	"testing"
)

func TestVersionLess(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"major", "v1.0.0", "v2.0.0", true},
		{"major reverse", "v2.0.0", "v1.0.0", false},
		{"minor", "v1.1.0", "v1.2.0", true},
		{"patch", "v1.0.1", "v1.0.2", true},
		{"numeric not lexicographic", "v1.9.0", "v1.10.0", true},
		{"equal", "v1.0.0", "v1.0.0", false},
		{"prerelease shorter", "v1.0.0", "v1.0.0-beta", true},
		{"prerelease comparison", "v1.0.0-alpha", "v1.0.0-beta", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VersionLess(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("VersionLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestSortVersionTags(t *testing.T) {
	tags := []string{"v1.10.0", "v1.2.0", "v2.0.0", "v1.0.0", "v1.9.0"}
	SortVersionTags(tags)
	want := []string{"v1.0.0", "v1.2.0", "v1.9.0", "v1.10.0", "v2.0.0"}
	for i, got := range tags {
		if got != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got, want[i])
		}
	}
}

func TestSplitVersionParts(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"v1.2.3", []string{"v", "1", ".", "2", ".", "3"}},
		{"v1.2.3-beta.1", []string{"v", "1", ".", "2", ".", "3", "-beta.", "1"}},
		{"abc", []string{"abc"}},
		{"123", []string{"123"}},
		{"", nil},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := SplitVersionParts(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseNum(t *testing.T) {
	tests := []struct {
		input  string
		wantN  int
		wantOK bool
	}{
		{"123", 123, true},
		{"0", 0, true},
		{"abc", 0, false},
		{"", 0, false},
		{"12abc", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			num, ok := ParseNum(tc.input)
			if num != tc.wantN || ok != tc.wantOK {
				t.Errorf(
					"ParseNum(%q) = (%d, %v), want (%d, %v)",
					tc.input, num, ok,
					tc.wantN, tc.wantOK,
				)
			}
		})
	}
}

func TestFilterTags(t *testing.T) {
	tags := []string{"v1.0.0", "v1.0.0-beta", "v2.0.0", "release-1.0"}
	pat := regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	got := FilterTags(tags, pat)
	want := []string{"v1.0.0", "v2.0.0"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPrevious_Success(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{
		gitTags: fakeGitTags([]string{
			"v1.0.0", "v2.0.0", "v1.5.0",
			"v2.0.0-beta", "v1.0.0-rc1",
		}, nil),
	}

	got, err := run.previous("v", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v2.0.0" {
		t.Errorf("got %q, want %q", got, "v2.0.0")
	}
}

func TestPrevious_NoMatch(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{gitTags: fakeGitTags([]string{"v1.0.0-beta", "release-1.0"}, nil)}

	_, err := run.previous("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPrevious_GitError(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{gitTags: fakeGitTags(nil, fmt.Errorf("git failed"))}

	_, err := run.previous("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLatest_IncludesPrerelease(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{
		gitTags: fakeGitTags([]string{"v1.0.0", "v2.0.0", "v2.1.0-beta.1"}, nil),
	}

	got, err := run.latest("v", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v2.1.0-beta.1" {
		t.Errorf("got %q, want %q", got, "v2.1.0-beta.1")
	}
}

func TestLatest_NoMatch(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{gitTags: fakeGitTags([]string{"release-1.0"}, nil)}

	_, err := run.latest("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLatest_GitError(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{gitTags: fakeGitTags(nil, fmt.Errorf("git failed"))}

	_, err := run.latest("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func fakeGitTags(tags []string, err error) func(io.Writer) ([]string, error) {
	return func(io.Writer) ([]string, error) { return tags, err }
}

func TestCurrent_Success(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{
		currentTag: func(io.Writer) (string, error) { return "v1.2.3", nil },
	}

	got, err := run.current(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.2.3" {
		t.Errorf("got %q, want %q", got, "v1.2.3")
	}
}

func TestCurrent_Error(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	run := runner{
		currentTag: func(io.Writer) (string, error) {
			return "", fmt.Errorf("no tag on HEAD")
		},
	}

	_, err := run.current(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
