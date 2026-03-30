package tag

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// FilterTags returns tags matching the given pattern.
func FilterTags(tags []string, pattern *regexp.Regexp) []string {
	var matched []string
	for _, t := range tags {
		if pattern.MatchString(t) {
			matched = append(matched, t)
		}
	}
	return matched
}

// SortVersionTags sorts tags using the same logic as
// sort -V.
func SortVersionTags(tags []string) {
	sort.SliceStable(tags, func(i, j int) bool { return VersionLess(tags[i], tags[j]) })
}

// VersionLess compares two version strings like sort -V.
func VersionLess(a, b string) bool {
	pa := SplitVersionParts(a)
	pb := SplitVersionParts(b)

	minLen := len(pa)
	if len(pb) < minLen {
		minLen = len(pb)
	}

	for k := 0; k < minLen; k++ {
		ai := pa[k]
		bi := pb[k]

		aNum, aOK := ParseNum(ai)
		bNum, bOK := ParseNum(bi)

		if aOK && bOK {
			if aNum != bNum {
				return aNum < bNum
			}
			continue
		}

		if ai != bi {
			return ai < bi
		}
	}

	return len(pa) < len(pb)
}

// SplitVersionParts splits a version string into
// alternating numeric and non-numeric segments.
func SplitVersionParts(s string) []string {
	var parts []string
	cur := ""
	inDigit := false

	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		switch {
		case cur == "":
			cur = string(c)
			inDigit = isDigit
		case isDigit == inDigit:
			cur += string(c)
		default:
			parts = append(parts, cur)
			cur = string(c)
			inDigit = isDigit
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

// ParseNum returns the integer value of s if it contains
// only digits, or false otherwise.
func ParseNum(s string) (int, bool) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	if len(s) == 0 {
		return 0, false
	}
	return n, true
}

type runner struct {
	gitTags    func(errW io.Writer) ([]string, error)
	currentTag func(errW io.Writer) (string, error)
}

// Previous finds the most recent semver tag matching
// <prefix>X.Y.Z with no pre-release suffix.
func Previous(prefix string, errW io.Writer) (string, error) {
	//nolint:exhaustruct // only gitTags needed
	r := runner{gitTags: GitTags}
	return r.previous(prefix, errW)
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

// Latest finds the latest tag matching <prefix>X.Y.Z
// with optional pre-release suffix.
func Latest(prefix string, errW io.Writer) (string, error) {
	//nolint:exhaustruct // only gitTags needed
	r := runner{gitTags: GitTags}
	return r.latest(prefix, errW)
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

// Current returns the exact tag on HEAD.
func Current(errW io.Writer) (string, error) {
	//nolint:exhaustruct // only currentTag needed
	r := runner{currentTag: defaultCurrentTag}
	return r.current(errW)
}

func (r *runner) current(errW io.Writer) (string, error) { return r.currentTag(errW) }

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
