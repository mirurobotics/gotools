package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/lint"

	"github.com/spf13/cobra"
)

// NewCheckCommand returns a cobra command that runs the
// custom Go linter.
func NewCheckCommand() *cobra.Command {
	var opts lint.CheckOpts

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run the custom Go linter",
		Long: "Run Miru's custom Go linter checks " +
			"on the specified directory.",
		RunE: func(_ *cobra.Command, _ []string) error {
			diags, fixed, err := lint.RunCheck(opts)
			if err != nil {
				return err
			}
			exitCode := lint.SummarizeOutcome(os.Stdout, opts.DoFix, diags, fixed)
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&opts.Path, "path", ".", "directory to scan")
	fl.BoolVar(&opts.DoFix, "fix", false, "auto-fix violations")
	fl.IntVar(
		&opts.MaxLineWidth, "max-line-width", 88,
		"maximum line width for collapsing multi-line calls",
	)
	fl.IntVar(
		&opts.TabWidth, "tab-width", 4,
		"tab width for calculating visual line width",
	)
	fl.IntVar(
		&opts.MaxFuncLen, "max-func-len", 50,
		"maximum function length (non-blank, non-comment)",
	)
	fl.IntVar(
		&opts.MaxNestDepth, "max-nest-depth", 4,
		"maximum nesting depth within functions",
	)
	fl.IntVar(
		&opts.MaxParamCount, "max-param-count", 5,
		"maximum param count excluding context.Context",
	)
	fl.StringVar(
		&opts.Exclude, "exclude", "",
		"comma-separated rules to exclude (empty=run all)",
	)
	fl.StringVar(&opts.Rule, "rule", "", "only run a specific rule")

	return cmd
}
