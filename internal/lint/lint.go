package lint

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mirurobotics/gotools/internal/lint/linter"
)

// NewLintCommand returns a cobra command for the top-level "miru lint"
// orchestrator. It runs the custom linter, gofumpt, and golangci-lint.
func NewLintCommand() *cobra.Command {
	var (
		paths           string
		doFix           bool
		maxLineWidth    int
		tabWidth        int
		maxFuncLen      int
		maxNestDepth    int
		maxParamCount   int
		exclude         string
		rule            string
		deadcode        bool
		deadcodeExclude string
		noGofumpt       bool
		noGolangci      bool
	)

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Run all Go linters",
		Long: `Run the full Go lint suite: custom linter, gofumpt, and golangci-lint.

By default, runs in fix mode. Use --fix=false for CI (check-only) mode.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			failed := false

			// Run custom linter on each path.
			if paths != "" {
				excl := parseExclusions(exclude)
				if err := linter.ValidateExclusions(excl); err != nil {
					return err
				}

				if rule != "" {
					r := linter.Rule(rule)
					if err := linter.ValidateExclusions(map[linter.Rule]bool{r: true}); err != nil {
						return fmt.Errorf("unknown rule: %q", rule)
					}
					excl = make(map[linter.Rule]bool)
					for _, ar := range linter.AllRules {
						if ar != r {
							excl[ar] = true
						}
					}
				}

				cfg := linter.Config{
					MaxLineWidth:  maxLineWidth,
					TabWidth:      tabWidth,
					MaxFuncLen:    maxFuncLen,
					MaxNestDepth:  maxNestDepth,
					MaxParamCount: maxParamCount,
					Exclude:       excl,
				}

				for _, p := range strings.Split(paths, ",") {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					fmt.Printf("Running custom linter on %s...\n", p)
					diags, fixed, err := linter.Run(p, doFix, cfg)
					if err != nil {
						return fmt.Errorf("custom linter error on %s: %w", p, err)
					}
					if doFix && fixed > 0 {
						fmt.Printf("%d file(s) fixed in %s.\n", fixed, p)
					}
					if diags > 0 {
						fmt.Printf("%d violation(s) found in %s.\n", diags, p)
						failed = true
					}
				}
			}

			// Run gofumpt.
			if !noGofumpt {
				if err := runGofumpt(doFix); err != nil {
					return err
				}
			}

			// Run golangci-lint.
			if !noGolangci {
				fmt.Println("Running golangci-lint...")
				if err := runExternal("golangci-lint", "run"); err != nil {
					failed = true
					fmt.Fprintf(os.Stderr, "golangci-lint failed: %v\n", err)
				}
			}

			// Run deadcode.
			if deadcode {
				if err := runDeadcode(deadcodeExclude); err != nil {
					failed = true
					fmt.Fprintf(os.Stderr, "deadcode failed: %v\n", err)
				}
			}

			if failed {
				return fmt.Errorf("lint checks failed")
			}

			fmt.Println("\nLint complete")
			return nil
		},
	}

	cmd.Flags().StringVar(&paths, "paths", "", "comma-separated directories for the custom linter")
	cmd.Flags().BoolVar(&doFix, "fix", true, "auto-fix violations (false for CI check-only mode)")
	cmd.Flags().IntVar(&maxLineWidth, "max-line-width", 88, "maximum line width for collapsing multi-line calls")
	cmd.Flags().IntVar(&tabWidth, "tab-width", 4, "tab width for calculating visual line width")
	cmd.Flags().IntVar(&maxFuncLen, "max-func-len", 50, "maximum function length in non-blank, non-comment lines")
	cmd.Flags().IntVar(&maxNestDepth, "max-nest-depth", 4, "maximum nesting depth within functions")
	cmd.Flags().IntVar(&maxParamCount, "max-param-count", 5, "maximum parameter count excluding context.Context")
	cmd.Flags().StringVar(&exclude, "exclude", "", "comma-separated rules to exclude (empty=run all)")
	cmd.Flags().StringVar(&rule, "rule", "", "only run a specific rule")
	cmd.Flags().BoolVar(&deadcode, "deadcode", false, "run deadcode checker")
	cmd.Flags().StringVar(&deadcodeExclude, "deadcode-exclude", "", "grep-v pattern for deadcode false positives")
	cmd.Flags().BoolVar(&noGofumpt, "no-gofumpt", false, "skip gofumpt")
	cmd.Flags().BoolVar(&noGolangci, "no-golangci", false, "skip golangci-lint")

	return cmd
}

// runGofumpt runs gofumpt in fix or check mode.
func runGofumpt(fix bool) error {
	if fix {
		fmt.Println("Running gofumpt...")
		return runExternal("gofumpt", "-w", ".")
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

// runDeadcode runs the deadcode checker, optionally filtering output.
func runDeadcode(excludePattern string) error {
	fmt.Println("Running deadcode...")
	out, err := exec.Command("deadcode", "-test", "./...").CombinedOutput()
	if err != nil {
		// deadcode may exit non-zero when it finds dead code; that's expected.
		// We still process the output.
		_ = err
	}

	// Filter out module cache noise.
	var filtered []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "/go/pkg/mod/") {
			continue
		}
		if line == "" {
			continue
		}
		filtered = append(filtered, line)
	}

	// Apply exclude pattern.
	if excludePattern != "" {
		var kept []string
		for _, line := range filtered {
			if !strings.Contains(line, excludePattern) {
				kept = append(kept, line)
			}
		}
		filtered = kept
	}

	if len(filtered) > 0 {
		for _, line := range filtered {
			fmt.Println(line)
		}
		return fmt.Errorf("deadcode found issues")
	}
	return nil
}

// runExternal runs an external command, inheriting stdout/stderr.
func runExternal(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
