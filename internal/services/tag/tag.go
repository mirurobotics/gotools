package tag

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Previous finds the latest <prefix>X.Y.Z tag without a pre-release suffix.
func Previous(prefix string, errW io.Writer) (string, error) {
	//nolint:exhaustruct // only gitTags needed
	run := runner{gitTags: GitTags}
	return run.previous(prefix, errW)
}

// Latest finds the latest tag matching <prefix>X.Y.Z with optional pre-release suffix.
func Latest(prefix string, errW io.Writer) (string, error) {
	//nolint:exhaustruct // only gitTags needed
	run := runner{gitTags: GitTags}
	return run.latest(prefix, errW)
}

// GitTags returns all git tags.
func GitTags(errW io.Writer) ([]string, error) {
	if errW == nil {
		errW = os.Stderr
	}
	//nolint:noctx // CLI tool, no context needed
	cmd := exec.Command("git", "tag")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = errW
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git tag: %w", err)
	}
	var tags []string
	for _, line := range strings.Split(buf.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

// Current returns the exact tag on HEAD.
func Current(errW io.Writer) (string, error) {
	//nolint:exhaustruct // only currentTag needed
	run := runner{currentTag: defaultCurrentTag}
	return run.current(errW)
}

func defaultCurrentTag(errW io.Writer) (string, error) {
	if errW == nil {
		errW = os.Stderr
	}
	//nolint:noctx // CLI tool, no context needed
	cmd := exec.Command("git", "describe", "--exact-match", "--tags", "HEAD")
	cmd.Stderr = errW
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no tag on HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

type runner struct {
	gitTags    func(errW io.Writer) ([]string, error)
	currentTag func(errW io.Writer) (string, error)
}

func (r *runner) previous(prefix string, errW io.Writer) (string, error) {
	tags, err := r.gitTags(errW)
	if err != nil {
		return "", err
	}

	pat := `^` + regexp.QuoteMeta(prefix) +
		`[0-9]+\.[0-9]+\.[0-9]+$`
	pattern := regexp.MustCompile(pat)
	matched := FilterTags(tags, pattern)

	if len(matched) == 0 {
		return "", fmt.Errorf("no tags matching %sX.Y.Z found", prefix)
	}

	SortVersionTags(matched)
	return matched[len(matched)-1], nil
}

func (r *runner) latest(prefix string, errW io.Writer) (string, error) {
	tags, err := r.gitTags(errW)
	if err != nil {
		return "", err
	}

	pat := `^` + regexp.QuoteMeta(prefix) +
		`[0-9]+\.[0-9]+\.[0-9]+`
	pattern := regexp.MustCompile(pat)
	matched := FilterTags(tags, pattern)

	if len(matched) == 0 {
		return "", fmt.Errorf("no tags matching %sX.Y.Z found", prefix)
	}

	SortVersionTags(matched)
	return matched[len(matched)-1], nil
}

func (r *runner) current(errW io.Writer) (string, error) { return r.currentTag(errW) }

// FilterTags returns tags matching the given pattern.
func FilterTags(tags []string, pattern *regexp.Regexp) []string {
	var matched []string
	for _, tag := range tags {
		if pattern.MatchString(tag) {
			matched = append(matched, tag)
		}
	}
	return matched
}

// SortVersionTags sorts tags using the same logic as sort -V.
func SortVersionTags(tags []string) {
	sort.SliceStable(tags, func(a, b int) bool { return VersionLess(tags[a], tags[b]) })
}

// VersionLess compares two version strings like sort -V.
func VersionLess(a, b string) bool {
	partsA, partsB := SplitVersionParts(a), SplitVersionParts(b)
	return slices.CompareFunc(partsA, partsB, comparePart) < 0
}

// SplitVersionParts splits a version into numeric and non-numeric segments.
func SplitVersionParts(version string) []string {
	var parts []string
	cur := ""
	inDigit := false

	for _, char := range version {
		isDigit := char >= '0' && char <= '9'
		switch {
		case cur == "":
			cur = string(char)
			inDigit = isDigit
		case isDigit == inDigit:
			cur += string(char)
		default:
			parts = append(parts, cur)
			cur = string(char)
			inDigit = isDigit
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

func comparePart(a, b string) int {
	aNum, aOK := ParseNum(a)
	bNum, bOK := ParseNum(b)
	if aOK && bOK {
		return cmp.Compare(aNum, bNum)
	}
	return strings.Compare(a, b)
}

// ParseNum returns the integer value of text if it is all digits.
func ParseNum(text string) (int, bool) {
	num := 0
	for _, char := range text {
		if char < '0' || char > '9' {
			return 0, false
		}
		num = num*10 + int(char-'0')
	}
	if len(text) == 0 {
		return 0, false
	}
	return num, true
}
