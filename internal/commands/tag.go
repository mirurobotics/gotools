package commands

import (
	"fmt"

	"github.com/mirurobotics/gotools/internal/services/tag"

	"github.com/spf13/cobra"
)

// NewTagCommand returns a cobra command with git tag
// utility subcommands.
func NewTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Git tag utilities",
		Long:  "Commands for querying git tags.",
	}

	cmd.AddCommand(newPreviousCommand())
	cmd.AddCommand(newLatestCommand())
	cmd.AddCommand(newCurrentCommand())

	return cmd
}

func newPreviousCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "previous [prefix]",
		Short: "Find most recent semver tag (exact only)",
		Long: "Finds the most recent tag matching " +
			"<prefix>X.Y.Z with no pre-release suffix. " +
			"Default prefix is 'v'.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			prefix := "v"
			if len(args) > 0 {
				prefix = args[0]
			}
			result, err := tag.Previous(prefix)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func newLatestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "latest [prefix]",
		Short: "Find latest semver tag (with pre-release)",
		Long: "Finds the latest tag matching " +
			"<prefix>X.Y.Z with optional pre-release " +
			"suffix. Default prefix is 'v'.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			prefix := "v"
			if len(args) > 0 {
				prefix = args[0]
			}
			result, err := tag.Latest(prefix)
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}

func newCurrentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get exact tag on HEAD",
		Long: "Gets the exact tag pointing at the " +
			"current HEAD commit.",
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := tag.Current()
			if err != nil {
				return err
			}
			fmt.Println(result)
			return nil
		},
	}
}
