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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VersionLess(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("VersionLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
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

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SplitVersionParts(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
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

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			n, ok := ParseNum(tt.input)
			if n != tt.wantN || ok != tt.wantOK {
				t.Errorf(
					"ParseNum(%q) = (%d, %v), want (%d, %v)",
					tt.input, n, ok,
					tt.wantN, tt.wantOK,
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

func fakeGitTags(tags []string, err error) func(io.Writer) ([]string, error) {
	return func(io.Writer) ([]string, error) { return tags, err }
}

func TestPrevious_Success(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		gitTags: fakeGitTags([]string{
			"v1.0.0", "v2.0.0", "v1.5.0",
			"v2.0.0-beta", "v1.0.0-rc1",
		}, nil),
	}

	got, err := r.previous("v", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v2.0.0" {
		t.Errorf("got %q, want %q", got, "v2.0.0")
	}
}

func TestPrevious_NoMatch(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{gitTags: fakeGitTags([]string{"v1.0.0-beta", "release-1.0"}, nil)}

	_, err := r.previous("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPrevious_GitError(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{gitTags: fakeGitTags(nil, fmt.Errorf("git failed"))}

	_, err := r.previous("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLatest_IncludesPrerelease(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		gitTags: fakeGitTags([]string{"v1.0.0", "v2.0.0", "v2.1.0-beta.1"}, nil),
	}

	got, err := r.latest("v", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v2.1.0-beta.1" {
		t.Errorf("got %q, want %q", got, "v2.1.0-beta.1")
	}
}

func TestLatest_NoMatch(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{gitTags: fakeGitTags([]string{"release-1.0"}, nil)}

	_, err := r.latest("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLatest_GitError(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{gitTags: fakeGitTags(nil, fmt.Errorf("git failed"))}

	_, err := r.latest("v", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCurrent_Success(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		currentTag: func(io.Writer) (string, error) { return "v1.2.3", nil },
	}

	got, err := r.current(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.2.3" {
		t.Errorf("got %q, want %q", got, "v1.2.3")
	}
}

func TestCurrent_Error(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		currentTag: func(io.Writer) (string, error) {
			return "", fmt.Errorf("no tag on HEAD")
		},
	}

	_, err := r.current(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
