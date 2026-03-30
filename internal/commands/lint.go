package commands

import (
	"github.com/mirurobotics/gotools/internal/services/lint"

	"github.com/spf13/cobra"
)

// NewLintCommand returns a cobra command for the
// top-level "miru lint" orchestrator.
func NewLintCommand() *cobra.Command {
	var opts lint.LintOpts

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Run all Go linters",
		Long: "Run the full Go lint suite: custom linter, " +
			"gofumpt, and golangci-lint.\n\n" +
			"By default, runs in fix mode. " +
			"Use --fix=false for CI (check-only) mode.",
		RunE: func(_ *cobra.Command, _ []string) error { return lint.RunLint(opts) },
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
	fl.BoolVar(&opts.Deadcode, "deadcode", false, "run deadcode checker")
	fl.StringVar(
		&opts.DeadcodeExclude, "deadcode-exclude", "",
		"grep-v pattern for deadcode false positives",
	)
	fl.BoolVar(&opts.NoGofumpt, "no-gofumpt", false, "skip gofumpt")
	fl.BoolVar(&opts.NoGolangci, "no-golangci", false, "skip golangci-lint")
}
