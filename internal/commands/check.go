package commands

import (
	"errors"
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
			opts.Out = os.Stdout
			opts.Err = os.Stderr
			diags, fixed, err := lint.RunCheck(opts)
			if err != nil {
				return err
			}
			exitCode := lint.SummarizeOutcome(os.Stdout, opts.DoFix, diags, fixed)
			if exitCode != 0 {
				return errors.New("check failed")
			}
			return nil
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&opts.Path, "path", ".", "directory to scan")
	fl.BoolVar(&opts.DoFix, "fix", false, "auto-fix violations")
	bindLinterConfigFlags(
		fl,
		&opts.MaxLineWidth, &opts.TabWidth,
		&opts.MaxFuncLen, &opts.MaxNestDepth, &opts.MaxParamCount,
		&opts.Exclude, &opts.Rule,
	)

	return cmd
}
