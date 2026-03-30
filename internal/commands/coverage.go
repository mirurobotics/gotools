package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/coverage"

	"github.com/spf13/cobra"
)

// NewCoverageCommand returns a cobra command that
// generates coverage reports.
func NewCoverageCommand() *cobra.Command {
	var opts coverage.Opts

	cmd := &cobra.Command{
		Use:   "coverage [sub-package]",
		Short: "Generate Go coverage report",
		Long: "Runs tests with coverage instrumentation " +
			"and generates a coverage report.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SubPkg = args[0]
			}
			opts.Out = os.Stdout
			opts.Err = os.Stderr
			return coverage.Run(opts)
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&opts.SrcPrefix, "src-prefix", "pkg", "source package prefix")
	fl.StringVar(&opts.TestPrefix, "test-prefix", "tests", "external test prefix")
	fl.BoolVar(&opts.NoHTML, "no-html", false, "skip opening HTML report")

	return cmd
}
