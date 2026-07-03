package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/lint"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewLintCommand returns a cobra command for the
// top-level "miru lint" orchestrator.
func NewLintCommand() *cobra.Command {
	var opts lint.LintOpts

	//nolint:exhaustruct // cobra uses partial initialization
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Run all Go linters",
		Long: "Run the full Go lint suite: custom linter, " +
			"gofumpt, and golangci-lint.\n\n" +
			"By default, runs in fix mode. " +
			"Use --fix=false for CI (check-only) mode.",
		RunE: func(_ *cobra.Command, _ []string) error {
			opts.Out = os.Stdout
			opts.Err = os.Stderr
			return lint.RunLint(opts)
		},
	}

	bindLintFlags(cmd, &opts)
	return cmd
}

func bindLintFlags(cmd *cobra.Command, opts *lint.LintOpts) {
	fl := cmd.Flags()
	fl.StringVar(
		&opts.Paths, "paths", "",
		"comma-separated directories for the custom linter",
	)
	fl.BoolVar(
		&opts.DoFix, "fix", true,
		"auto-fix violations (false for CI check-only mode)",
	)
	bindLinterConfigFlags(fl, &opts.LinterFlags)
	fl.BoolVar(&opts.Deadcode, "deadcode", true, "run deadcode checker")
	fl.StringVar(
		&opts.DeadcodeExclude, "deadcode-exclude", "",
		"grep-v pattern for deadcode false positives",
	)
	fl.BoolVar(&opts.NoGofumpt, "no-gofumpt", false, "skip gofumpt")
	fl.BoolVar(&opts.NoGolangci, "no-golangci", false, "skip golangci-lint")
	fl.StringVar(
		&opts.NewFromRev, "new-from-rev", "",
		"only report new golangci-lint issues since this git revision",
	)
}

// bindLinterConfigFlags binds the shared linter configuration
// flags used by both check and lint commands.
func bindLinterConfigFlags(fl *pflag.FlagSet, flags *lint.LinterFlags) {
	fl.IntVar(
		&flags.MaxLineWidth, "max-line-width", 88,
		"maximum line width for collapsing multi-line calls",
	)
	fl.IntVar(
		&flags.TabWidth, "tab-width", 4,
		"tab width for calculating visual line width",
	)
	fl.IntVar(
		&flags.MaxFuncLen, "max-func-len", 50,
		"maximum function length (non-blank, non-comment)",
	)
	fl.IntVar(
		&flags.MaxNestDepth, "max-nest-depth", 4,
		"maximum nesting depth within functions",
	)
	fl.IntVar(
		&flags.MaxParamCount, "max-param-count", 6,
		"maximum param count excluding context.Context",
	)
	fl.StringVar(
		&flags.Exclude, "exclude", "",
		"comma-separated rules to exclude (empty=run all)",
	)
	fl.StringVar(&flags.Rule, "rule", "", "only run a specific rule")
}
