package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/gotest"

	"github.com/spf13/cobra"
)

// NewTestCommand returns a cobra command that runs Go
// tests.
func NewTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [-- extra args]",
		Short: "Run Go tests",
		Long: "Runs go test ./... from the repo root. " +
			"Arguments after -- are passed through.",
		DisableFlagParsing: true,
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		for _, a := range args {
			if a == "--help" || a == "-h" {
				return cmd.Help()
			}
		}
		return gotest.Run(gotest.Opts{
			ExtraArgs: args,
			Out:       os.Stdout,
			Err:       os.Stderr,
		})
	}

	return cmd
}
