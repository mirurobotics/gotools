package lint

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter"
)

// LintOpts holds the options for the lint orchestrator.
type LintOpts struct {
	Paths           string
	DoFix           bool
	MaxLineWidth    int
	TabWidth        int
	MaxFuncLen      int
	MaxNestDepth    int
	MaxParamCount   int
	Exclude         string
	Rule            string
	Deadcode        bool
	DeadcodeExclude string
	NoGofumpt       bool
	NoGolangci      bool
}

// RunLint runs the full lint suite: custom linter,
// gofumpt, and golangci-lint.
func RunLint(opts LintOpts) error {
	var failures []string

	if opts.Paths != "" {
		hadIssues, err := runCustomLinter(opts)
		if err != nil {
			return err
		}
		if hadIssues {
			failures = append(failures, "custom linter")
		}
	}

	if !opts.NoGofumpt {
		if err := RunGofumpt(opts.DoFix); err != nil {
			return fmt.Errorf("gofumpt: %w", err)
		}
	}

	if !opts.NoGolangci {
		if err := RunGolangci(); err != nil {
			failures = append(failures, "golangci-lint")
		}
	}

	if opts.Deadcode {
		if err := RunDeadcode(opts.DeadcodeExclude); err != nil {
			failures = append(failures, "deadcode")
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("lint failed: %s", strings.Join(failures, ", "))
	}

	fmt.Println("\nLint complete")
	return nil
}

func runCustomLinter(opts LintOpts) (bool, error) {
	cfg, err := BuildLinterConfig(
		opts.Exclude, opts.Rule,
		opts.MaxLineWidth, opts.TabWidth,
		opts.MaxFuncLen, opts.MaxNestDepth, opts.MaxParamCount,
	)
	if err != nil {
		return false, err
	}

	totalDiags := 0
	for _, p := range strings.Split(opts.Paths, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fmt.Printf("Running custom linter on %s...\n", p)
		diags, fixed, runErr := linter.Run(p, opts.DoFix, cfg)
		if runErr != nil {
			return false, fmt.Errorf("custom linter on %s: %w", p, runErr)
		}
		if opts.DoFix && fixed > 0 {
			fmt.Printf("%d file(s) fixed in %s.\n", fixed, p)
		}
		if diags > 0 {
			fmt.Printf("%d violation(s) found in %s.\n", diags, p)
		}
		totalDiags += diags
	}
	return totalDiags > 0, nil
}

// RunGolangci runs golangci-lint.
func RunGolangci() error {
	fmt.Println("Running golangci-lint...")
	if err := RunExternal("golangci-lint", "run"); err != nil {
		fmt.Fprintf(os.Stderr, "golangci-lint failed: %v\n", err)
		return err
	}
	return nil
}

// RunGofumpt runs gofumpt in fix or check mode.
func RunGofumpt(fix bool) error {
	if fix {
		fmt.Println("Running gofumpt...")
		return RunExternal("gofumpt", "-w", ".")
	}

	fmt.Println("Checking gofumpt...")
	out, err := exec.Command("gofumpt", "-l", ".").CombinedOutput()
	if err != nil {
		return fmt.Errorf("gofumpt failed: %w\n%s", err, out)
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed != "" {
		fmt.Println("Files need formatting:")
		fmt.Println(trimmed)
		return fmt.Errorf("gofumpt found unformatted files")
	}
	return nil
}

// RunDeadcode runs the deadcode checker, optionally
// filtering output.
func RunDeadcode(excludePattern string) error {
	fmt.Println("Running deadcode...")
	out, err := exec.Command("deadcode", "-test", "./...").CombinedOutput()

	filtered := FilterDeadcodeOutput(string(out), excludePattern)
	if len(filtered) > 0 {
		for _, line := range filtered {
			fmt.Println(line)
		}
		return fmt.Errorf("deadcode found issues")
	}
	if err != nil {
		return fmt.Errorf("deadcode: %w", err)
	}
	return nil
}

// FilterDeadcodeOutput filters deadcode output, removing
// module paths and optional exclude patterns.
func FilterDeadcodeOutput(raw, excludePattern string) []string {
	var filtered []string
	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, "/go/pkg/mod/") {
			continue
		}
		if line == "" {
			continue
		}
		filtered = append(filtered, line)
	}

	if excludePattern != "" {
		var kept []string
		for _, line := range filtered {
			if !strings.Contains(line, excludePattern) {
				kept = append(kept, line)
			}
		}
		filtered = kept
	}
	return filtered
}

// RunExternal runs an external command, inheriting
// stdout/stderr.
func RunExternal(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
