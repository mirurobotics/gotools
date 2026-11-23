package commands

import (
	"fmt"
	"strings"

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
			" in the working directory: one section per command path " +
			"(such as \"deps update\"), keyed by flag name. " +
			"Flags passed on the command line win; --help shows built-in defaults.",
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
	if isCobraBuiltin(cmd) {
		return nil
	}
	cfg, err := repoconfig.Load(".")
	if err != nil {
		return err
	}
	for path, values := range cfg {
		if err := validateSection(cmd.Root(), path, values); err != nil {
			return err
		}
	}
	return applyFlagValues(cmd, cfg[commandPath(cmd)])
}

func isCobraBuiltin(cmd *cobra.Command) bool {
	name, _, _ := strings.Cut(commandPath(cmd), " ")
	switch name {
	case "help", "completion",
		cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return true
	}
	return false
}

func validateSection(root *cobra.Command, path string, values map[string]string) error {
	target, _, err := root.Find(strings.Fields(path))
	if err != nil || target == root || commandPath(target) != path {
		return fmt.Errorf("%s: unknown command %q", repoconfig.FileName, path)
	}
	for name := range values {
		if target.Flags().Lookup(name) == nil {
			return fmt.Errorf(
				"%s: %s.%s: unknown flag", repoconfig.FileName, path, name,
			)
		}
	}
	return nil
}

func applyFlagValues(cmd *cobra.Command, values map[string]string) error {
	flags := cmd.Flags()
	for name, value := range values {
		if flags.Changed(name) {
			continue
		}
		if err := flags.Set(name, value); err != nil {
			return fmt.Errorf(
				"%s: %s.%s: %w", repoconfig.FileName, commandPath(cmd), name, err,
			)
		}
	}
	return nil
}

func commandPath(cmd *cobra.Command) string {
	return strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
}
