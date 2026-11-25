package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/covgate"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewCovgateCommand returns a command that checks per-package coverage.
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

	flags := cmd.Flags()
	bindCovgatePackageFlags(flags, &opts)
	bindCovgateThresholdFlags(flags, &opts)

	return cmd
}

func bindCovgatePackageFlags(flags *pflag.FlagSet, opts *covgate.Opts) {
	flags.StringVar(
		&opts.Packages, "packages", "./...", "Go package pattern for go list",
	)
	flags.StringVar(
		&opts.Exclude, "exclude", "",
		"comma-separated list of Go list patterns to exclude from coverage measurement",
	)
	bindLayoutFlags(flags, &opts.SrcPrefix, &opts.TestDir)
	flags.IntVarP(
		&opts.Parallelism, "parallelism", "p", 0,
		"max concurrent package measurements (0 = NumCPU; emits progress)",
	)
}

func bindCovgateThresholdFlags(flags *pflag.FlagSet, opts *covgate.Opts) {
	flags.Float64Var(
		&opts.DefaultThreshold, "default-threshold", 80.0,
		"fallback threshold when no .covgate exists",
	)
	flags.BoolVar(
		&opts.TightnessEnabled, "tightness", true,
		"fail when required threshold lags actual coverage "+
			"by more than --tightness-tolerance",
	)
	flags.Float64Var(
		&opts.TightnessTolerance, "tightness-tolerance", 0.5,
		"max allowed gap in percentage points between "+
			"actual coverage and required threshold",
	)
}
