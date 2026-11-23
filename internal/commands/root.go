package commands

import (
	"fmt"

	"github.com/mirurobotics/gotools/internal/services/repoconfig"

	"github.com/spf13/cobra"
)

// NewRootCommand returns the root cobra command with all subcommands attached.
func NewRootCommand() *cobra.Command {
	//nolint:exhaustruct // cobra uses partial initialization
	cmd := &cobra.Command{
		Use:   "miru",
		Short: "Miru Go development tools",
		Long: "A collection of Go development tools for Miru projects.\n\n" +
			"Flag defaults are read from " + repoconfig.FileName +
			" in the working directory: one section per command, " +
			"keyed by flag name. Flags passed on the command line win.",
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: applyRepoConfig,
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

func applyRepoConfig(cmd *cobra.Command, _ []string) error {
	cfg, err := repoconfig.Load(".")
	if err != nil {
		return err
	}
	for name := range cfg {
		if !hasSubcommand(cmd.Root(), name) {
			return fmt.Errorf("%s: unknown command %q", repoconfig.FileName, name)
		}
	}
	return applyFlagValues(cmd, cfg[cmd.Name()])
}

func hasSubcommand(root *cobra.Command, name string) bool {
	for _, sub := range root.Commands() {
		if sub.Name() == name {
			return true
		}
	}
	return false
}

func applyFlagValues(cmd *cobra.Command, values map[string]string) error {
	flags := cmd.Flags()
	for name, value := range values {
		if flags.Changed(name) {
			continue
		}
		if err := flags.Set(name, value); err != nil {
			return fmt.Errorf(
				"%s: %s.%s: %w", repoconfig.FileName, cmd.Name(), name, err,
			)
		}
	}
	return nil
}
