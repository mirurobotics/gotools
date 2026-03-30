package commands

import (
	"os"

	"github.com/mirurobotics/gotools/internal/services/deps"

	"github.com/spf13/cobra"
)

// NewDepsCommand returns a cobra command with dependency
// management subcommands.
func NewDepsCommand() *cobra.Command {
	//nolint:exhaustruct // cobra uses partial initialization
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Dependency management utilities",
		Long: "Commands for managing Go module " +
			"dependencies.",
	}

	cmd.AddCommand(newUpdateCommand())

	return cmd
}

func newUpdateCommand() *cobra.Command {
	//nolint:exhaustruct // cobra uses partial initialization
	return &cobra.Command{
		Use:   "update",
		Short: "Update all Go dependencies",
		Long: "Runs go get -u ./..., go mod tidy, " +
			"and go mod verify.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return deps.RunUpdate(os.Stdout, os.Stderr)
		},
	}
}
