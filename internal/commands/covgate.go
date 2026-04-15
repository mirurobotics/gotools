package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/covgate"

	"github.com/spf13/cobra"
)

// NewCovgateCommand returns a cobra command that checks
// per-package coverage against thresholds.
func NewCovgateCommand() *cobra.Command {
	var opts covgate.Opts

	//nolint:exhaustruct // cobra uses partial initialization
	cmd := &cobra.Command{
		Use:   "covgate",
		Short: "Check per-package coverage thresholds",
		Long: "Discovers .covgate files, runs tests " +
			"with coverage instrumentation, and checks " +
			"per-package coverage against thresholds.",
		RunE: func(_ *cobra.Command, _ []string) error {
			opts.Out = os.Stdout
			return covgate.Run(opts)
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&opts.Packages, "packages", "./...", "Go package pattern for go list")
	fl.StringVar(
		&opts.SrcPrefix, "src-prefix", "pkg",
		"source prefix for mapping external test dirs",
	)
	fl.StringVar(
		&opts.TestDir, "test-dir", "",
		"external test directory (empty = none)",
	)
	fl.Float64Var(
		&opts.DefaultThreshold, "default-threshold", 80.0,
		"fallback threshold when no .covgate exists",
	)
	fl.IntVarP(
		&opts.Parallelism, "parallelism", "p", 0,
		"max concurrent package measurements (0 = NumCPU)",
	)
	fl.BoolVar(
		&opts.TightnessEnabled, "tightness", true,
		"fail when required threshold lags actual coverage "+
			"by more than --tightness-tolerance",
	)
	fl.Float64Var(
		&opts.TightnessTolerance, "tightness-tolerance", 0.5,
		"max allowed gap in percentage points between "+
			"actual coverage and required threshold",
	)

	return cmd
}
