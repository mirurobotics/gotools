package covgate

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRun_Exclude_GoListError(t *testing.T) {
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(pattern string) ([]string, error) {
			if pattern == "./..." {
				return []string{"example.com/mod/pkg/a"}, nil
			}
			return nil, fmt.Errorf("list failed")
		},
		measure: fakeMeasure(90.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		Packages:         "./...",
		Exclude:          "./pkg/bogus",
	})
	if err == nil {
		t.Fatal("expected error from exclude pattern resolution")
	}
	msg := err.Error()
	if !strings.Contains(msg, "./pkg/bogus") {
		t.Errorf("error missing pattern %q: %v", "./pkg/bogus", err)
	}
	if !strings.Contains(msg, "list failed") {
		t.Errorf("error missing wrapped cause %q: %v", "list failed", err)
	}
	if !strings.Contains(msg, "resolve exclude") {
		t.Errorf("error missing context prefix %q: %v", "resolve exclude", err)
	}
}

const (
	pkgA = "example.com/mod/pkg/a"
	pkgB = "example.com/mod/pkg/b"
	pkgC = "example.com/mod/pkg/c"
)

func allThree() []string { return []string{pkgA, pkgB, pkgC} }

func runExclude(t *testing.T, exclude string, lookup map[string][]string) string {
	t.Helper()
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(pattern string) ([]string, error) {
			out, ok := lookup[pattern]
			if !ok {
				return nil, fmt.Errorf("unexpected pattern %q", pattern)
			}
			return out, nil
		},
		measure: fakeMeasure(90.0),
	}
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{
		Out:              &buf,
		DefaultThreshold: 80.0,
		Packages:         "./...",
		Exclude:          exclude,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return buf.String()
}

func TestRun_Exclude_NoExclude(t *testing.T) {
	out := runExclude(t, "", map[string][]string{"./...": allThree()})
	assertOutput(t, out,
		[]string{"pkg/a", "pkg/b", "pkg/c"},
		[]string{"Excluded", "SKIPPED"})
}

func TestRun_Exclude_Subset(t *testing.T) {
	lookup := map[string][]string{"./...": allThree(), "./pkg/b": {pkgB}}
	out := runExclude(t, "./pkg/b", lookup)
	assertOutput(t, out, []string{
		"pkg/a", "pkg/b", "pkg/c", "SKIPPED",
		"Excluded 1 package(s) from coverage measurement",
		"All packages meet minimum coverage requirement",
	}, []string{"FAIL"})
}

func TestRun_Exclude_NoOpPattern(t *testing.T) {
	lookup := map[string][]string{"./...": allThree(), "./does-not-exist/...": {}}
	out := runExclude(t, "./does-not-exist/...", lookup)
	assertOutput(t, out,
		[]string{"pkg/a", "pkg/b", "pkg/c"},
		[]string{"Excluded", "SKIPPED"})
}

func TestRun_Exclude_MultiplePatternsWithWhitespace(t *testing.T) {
	lookup := map[string][]string{
		"./...":   allThree(),
		"./pkg/a": {pkgA},
		"./pkg/c": {pkgC},
	}
	out := runExclude(t, ", ./pkg/a, , ./pkg/c,", lookup)
	assertOutput(t, out, []string{
		"pkg/a", "pkg/b", "pkg/c", "SKIPPED",
		"Excluded 2 package(s) from coverage measurement",
	}, []string{"FAIL"})
}

func TestRun_Exclude_AllPackages(t *testing.T) {
	out := runExclude(t, "./...", map[string][]string{"./...": allThree()})
	assertOutput(t, out, []string{
		"pkg/a", "pkg/b", "pkg/c", "SKIPPED",
		"Excluded 3 package(s) from coverage measurement",
		"All packages meet minimum coverage requirement",
		"Total time:",
	}, []string{"FAIL"})
}

func TestRun_Exclude_OrderingAndCount(t *testing.T) {
	lookup := map[string][]string{"./...": allThree(), "./pkg/b": {pkgB}}
	out := runExclude(t, "./pkg/b", lookup)
	assertOutput(t, out,
		[]string{"pkg/a", "pkg/b", "pkg/c", "SKIPPED"},
		[]string{"FAIL"})
	assertSkippedAfterMeasured(t, out)
	assertSkippedLine(t, out)
}

func assertSkippedAfterMeasured(t *testing.T, out string) {
	t.Helper()
	idxSkipped := strings.Index(out, "SKIPPED")
	idxA := strings.Index(out, "pkg/a")
	idxC := strings.Index(out, "pkg/c")
	if idxA < 0 || idxC < 0 || idxSkipped < 0 {
		t.Fatalf("expected pkg/a, pkg/c, and SKIPPED in output:\n%s", out)
	}
	if idxA >= idxSkipped {
		t.Errorf("expected measured pkg/a row (%d) before SKIPPED block (%d):\n%s",
			idxA, idxSkipped, out)
	}
	if idxC >= idxSkipped {
		t.Errorf("expected measured pkg/c row (%d) before SKIPPED block (%d):\n%s",
			idxC, idxSkipped, out)
	}
	if got := strings.Count(out, "SKIPPED"); got != 1 {
		t.Errorf("expected exactly 1 SKIPPED, got %d:\n%s", got, out)
	}
}

func assertSkippedLine(t *testing.T, out string) {
	t.Helper()
	var skippedLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "pkg/b") {
			skippedLine = line
			break
		}
	}
	if skippedLine == "" {
		t.Fatalf("no line contained pkg/b:\n%s", out)
	}
	if !strings.Contains(skippedLine, "SKIPPED") {
		t.Errorf("pkg/b line missing SKIPPED: %q", skippedLine)
	}
	if n := strings.Count(skippedLine, "---"); n != 3 {
		t.Errorf("pkg/b SKIPPED line expected 3 '---' placeholders, got %d: %q",
			n, skippedLine)
	}
}
