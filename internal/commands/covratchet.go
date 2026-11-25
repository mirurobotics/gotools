package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/covratchet"

	"github.com/spf13/cobra"
)

// NewCovratchetCommand returns a cobra command that ratchets up .covgate thresholds.
func NewCovratchetCommand() *cobra.Command {
	var opts covratchet.Opts

	//nolint:exhaustruct // cobra uses partial initialization
	cmd := &cobra.Command{
		Use:   "covratchet",
		Short: "Ratchet up per-package coverage thresholds",
		Long: "Discovers .covgate files, runs tests " +
			"with coverage instrumentation, and ratchets " +
			"thresholds up when actual coverage exceeds " +
			"the current threshold.",
		RunE: func(_ *cobra.Command, _ []string) error {
			opts.Out = os.Stdout
			return covratchet.Run(opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(
		&opts.Packages, "packages", "./...", "Go package pattern for go list",
	)
	bindLayoutFlags(flags, &opts.SrcPrefix, &opts.TestDir)
	flags.IntVarP(
		&opts.Parallelism, "parallelism", "p", 0,
		"max concurrent package measurements (0 = NumCPU; emits progress)",
	)

	return cmd
}
