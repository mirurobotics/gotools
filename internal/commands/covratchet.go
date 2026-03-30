package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/covratchet"

	"github.com/spf13/cobra"
)

// NewCovratchetCommand returns a cobra command that
// ratchets up .covgate thresholds.
func NewCovratchetCommand() *cobra.Command {
	var opts covratchet.Opts

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

	return cmd
}
