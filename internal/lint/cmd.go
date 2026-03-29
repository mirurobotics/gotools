package lint

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mirurobotics/gotools/internal/lint/linter"
)

// NewCommand returns a cobra command that runs the custom Go linter.
// The command name is "check" since "lint" is the orchestrator command
// that also runs gofumpt/golangci-lint.
func NewCommand() *cobra.Command {
	var (
		path          string
		doFix         bool
		maxLineWidth  int
		tabWidth      int
		maxFuncLen    int
		maxNestDepth  int
		maxParamCount int
		exclude       string
		rule          string
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run the custom Go linter",
		Long:  "Run Miru's custom Go linter checks on the specified directory.",
		RunE: func(_ *cobra.Command, _ []string) error {
			excl := parseExclusions(exclude)
			if err := linter.ValidateExclusions(excl); err != nil {
				return err
			}

			if rule != "" {
				// When --rule is set, exclude everything except that rule.
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

			diags, fixed, err := linter.Run(path, doFix, cfg)
			if err != nil {
				return err
			}

			exitCode := summarizeOutcome(os.Stdout, doFix, diags, fixed)
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "directory to scan for .go files")
	cmd.Flags().BoolVar(&doFix, "fix", false, "auto-fix violations in place")
	cmd.Flags().IntVar(&maxLineWidth, "max-line-width", 88, "maximum line width for collapsing multi-line calls")
	cmd.Flags().IntVar(&tabWidth, "tab-width", 4, "tab width for calculating visual line width")
	cmd.Flags().IntVar(&maxFuncLen, "max-func-len", 50, "maximum function length in non-blank, non-comment lines")
	cmd.Flags().IntVar(&maxNestDepth, "max-nest-depth", 4, "maximum nesting depth within functions")
	cmd.Flags().IntVar(&maxParamCount, "max-param-count", 5, "maximum parameter count excluding context.Context")
	cmd.Flags().StringVar(&exclude, "exclude", "", "comma-separated rules to exclude (empty=run all)")
	cmd.Flags().StringVar(&rule, "rule", "", "only run a specific rule")

	return cmd
}

func parseExclusions(csv string) map[linter.Rule]bool {
	excl := make(map[linter.Rule]bool)
	if csv == "" {
		return excl
	}
	for _, r := range strings.Split(csv, ",") {
		excl[linter.Rule(strings.TrimSpace(r))] = true
	}
	return excl
}

func summarizeOutcome(w io.Writer, doFix bool, diags, fixed int) int {
	if doFix && fixed > 0 {
		_, _ = fmt.Fprintf(w, "\n%d file(s) fixed.\n", fixed)
	}
	if diags > 0 {
		_, _ = fmt.Fprintf(w, "\n%d violation(s) found.\n", diags)
		return 1
	}
	return 0
}
