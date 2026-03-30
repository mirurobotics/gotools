package lint

import (
	"fmt"
	"io"
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
	Out             io.Writer
	Err             io.Writer
}

// RunLint runs the full lint suite: custom linter,
// gofumpt, and golangci-lint.
func RunLint(opts LintOpts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

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
		if err := RunGofumpt(opts.Out, opts.Err, opts.DoFix); err != nil {
			return fmt.Errorf("gofumpt: %w", err)
		}
	}

	if !opts.NoGolangci {
		if err := RunGolangci(opts.Out, opts.Err); err != nil {
			failures = append(failures, "golangci-lint")
		}
	}

	if opts.Deadcode {
		if err := RunDeadcode(opts.Out, opts.DeadcodeExclude); err != nil {
			failures = append(failures, "deadcode")
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("lint failed: %s", strings.Join(failures, ", "))
	}

	_, _ = fmt.Fprintln(opts.Out, "\nLint complete")
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
	cfg.Out = opts.Out
	cfg.Err = opts.Err

	totalDiags := 0
	for _, p := range strings.Split(opts.Paths, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, _ = fmt.Fprintf(opts.Out, "Running custom linter on %s...\n", p)
		diags, fixed, runErr := linter.Run(p, opts.DoFix, cfg)
		if runErr != nil {
			return false, fmt.Errorf("custom linter on %s: %w", p, runErr)
		}
		if opts.DoFix && fixed > 0 {
			_, _ = fmt.Fprintf(opts.Out, "%d file(s) fixed in %s.\n", fixed, p)
		}
		if diags > 0 {
			_, _ = fmt.Fprintf(opts.Out, "%d violation(s) found in %s.\n", diags, p)
		}
		totalDiags += diags
	}
	return totalDiags > 0, nil
}

// RunGolangci runs golangci-lint.
func RunGolangci(out io.Writer, errW io.Writer) error {
	_, _ = fmt.Fprintln(out, "Running golangci-lint...")
	if err := RunExternal(out, errW, "go", "tool", "golangci-lint", "run"); err != nil {
		_, _ = fmt.Fprintf(errW, "golangci-lint failed: %v\n", err)
		return err
	}
	return nil
}

// RunGofumpt runs gofumpt in fix or check mode.
func RunGofumpt(out io.Writer, errW io.Writer, fix bool) error {
	if fix {
		_, _ = fmt.Fprintln(out, "Running gofumpt...")
		return RunExternal(out, errW, "go", "tool", "gofumpt", "-w", ".")
	}

	_, _ = fmt.Fprintln(out, "Checking gofumpt...")
	cmdOut, err := exec.Command("go", "tool", "gofumpt", "-l", ".").CombinedOutput()
	if err != nil {
		return fmt.Errorf("gofumpt failed: %w\n%s", err, cmdOut)
	}
	trimmed := strings.TrimSpace(string(cmdOut))
	if trimmed != "" {
		_, _ = fmt.Fprintln(out, "Files need formatting:")
		_, _ = fmt.Fprintln(out, trimmed)
		return fmt.Errorf("gofumpt found unformatted files")
	}
	return nil
}

// RunDeadcode runs the deadcode checker, optionally
// filtering output.
func RunDeadcode(out io.Writer, excludePattern string) error {
	_, _ = fmt.Fprintln(out, "Running deadcode...")
	cmdOut, err := exec.Command("go", "tool", "deadcode", "-test", "./...").CombinedOutput()

	filtered := FilterDeadcodeOutput(string(cmdOut), excludePattern)
	if len(filtered) > 0 {
		for _, line := range filtered {
			_, _ = fmt.Fprintln(out, line)
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
// stdout/stderr from the provided writers.
func RunExternal(out io.Writer, errW io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = out
	cmd.Stderr = errW
	return cmd.Run()
}
