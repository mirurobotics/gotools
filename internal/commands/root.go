package commands

import "github.com/spf13/cobra"

// NewRootCommand returns the root cobra command with all
// subcommands attached.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "miru",
		Short: "Miru Go development tools",
		Long: "A collection of Go development tools " +
			"for Miru projects.",
	}

	cmd.AddCommand(NewLintCommand())
	cmd.AddCommand(NewCheckCommand())
	cmd.AddCommand(NewCovgateCommand())
	cmd.AddCommand(NewCovratchetCommand())
	cmd.AddCommand(NewCoverageCommand())
	cmd.AddCommand(NewTestCommand())
	cmd.AddCommand(NewDepsCommand())
	cmd.AddCommand(NewTagCommand())

	return cmd
}
