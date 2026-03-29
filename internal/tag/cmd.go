package tag

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// NewCommand returns a cobra command with git tag utility subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Git tag utilities",
		Long:  "Commands for querying git tags.",
	}

	cmd.AddCommand(newPreviousCommand())
	cmd.AddCommand(newLatestCommand())
	cmd.AddCommand(newCurrentCommand())

	return cmd
}

func newPreviousCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "previous [prefix]",
		Short: "Find most recent semver tag (exact X.Y.Z only)",
		Long:  "Finds the most recent tag matching <prefix>X.Y.Z with no pre-release suffix. Default prefix is 'v'.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			prefix := "v"
			if len(args) > 0 {
				prefix = args[0]
			}
			return runPrevious(prefix)
		},
	}
}

func newLatestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "latest [prefix]",
		Short: "Find latest semver tag (including pre-release)",
		Long:  "Finds the latest tag matching <prefix>X.Y.Z with optional pre-release suffix. Default prefix is 'v'.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			prefix := "v"
			if len(args) > 0 {
				prefix = args[0]
			}
			return runLatest(prefix)
		},
	}
}

func newCurrentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get exact tag on HEAD",
		Long:  "Gets the exact tag pointing at the current HEAD commit.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCurrent()
		},
	}
}

func runPrevious(prefix string) error {
	tags, err := gitTags()
	if err != nil {
		return err
	}

	// Match exact semver: <prefix>X.Y.Z with no trailing characters.
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `[0-9]+\.[0-9]+\.[0-9]+$`)
	var matched []string
	for _, t := range tags {
		if pattern.MatchString(t) {
			matched = append(matched, t)
		}
	}

	if len(matched) == 0 {
		return fmt.Errorf("no tags matching %sX.Y.Z found", prefix)
	}

	sortVersionTags(matched)
	fmt.Println(matched[len(matched)-1])
	return nil
}

func runLatest(prefix string) error {
	tags, err := gitTags()
	if err != nil {
		return err
	}

	// Match semver with optional pre-release: <prefix>X.Y.Z...
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `[0-9]+\.[0-9]+\.[0-9]+`)
	var matched []string
	for _, t := range tags {
		if pattern.MatchString(t) {
			matched = append(matched, t)
		}
	}

	if len(matched) == 0 {
		return fmt.Errorf("no tags matching %sX.Y.Z found", prefix)
	}

	sortVersionTags(matched)
	fmt.Println(matched[len(matched)-1])
	return nil
}

func runCurrent() error {
	cmd := exec.Command("git", "describe", "--exact-match", "--tags", "HEAD")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("no tag on HEAD: %w", err)
	}
	fmt.Print(strings.TrimSpace(string(out)))
	fmt.Println()
	return nil
}

func gitTags() ([]string, error) {
	cmd := exec.Command("git", "tag")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
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

// sortVersionTags sorts tags using the same logic as `sort -V`.
// It splits version strings into comparable parts for proper semver ordering.
func sortVersionTags(tags []string) {
	sort.SliceStable(tags, func(i, j int) bool {
		return versionLess(tags[i], tags[j])
	})
}

// versionLess compares two version strings like sort -V would.
// It extracts numeric parts and compares them numerically.
func versionLess(a, b string) bool {
	pa := splitVersionParts(a)
	pb := splitVersionParts(b)

	minLen := len(pa)
	if len(pb) < minLen {
		minLen = len(pb)
	}

	for k := 0; k < minLen; k++ {
		ai := pa[k]
		bi := pb[k]

		// Try numeric comparison.
		aNum, aIsNum := parseNum(ai)
		bNum, bIsNum := parseNum(bi)

		if aIsNum && bIsNum {
			if aNum != bNum {
				return aNum < bNum
			}
			continue
		}

		// Fall back to string comparison.
		if ai != bi {
			return ai < bi
		}
	}

	return len(pa) < len(pb)
}

func splitVersionParts(s string) []string {
	var parts []string
	cur := ""
	inDigit := false

	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		if cur == "" {
			cur = string(c)
			inDigit = isDigit
		} else if isDigit == inDigit {
			cur += string(c)
		} else {
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

func parseNum(s string) (int, bool) {
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
