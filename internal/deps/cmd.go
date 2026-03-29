package deps

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// NewCommand returns a cobra command with dependency management subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deps",
		Short: "Dependency management utilities",
		Long:  "Commands for managing Go module dependencies.",
	}

	cmd.AddCommand(newUpdateCommand())

	return cmd
}

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update all Go dependencies",
		Long:  "Runs go get -u ./..., go mod tidy, and go mod verify.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runUpdate()
		},
	}
}

func runUpdate() error {
	steps := []struct {
		label string
		args  []string
	}{
		{"Updating dependencies...", []string{"get", "-u", "./..."}},
		{"Tidying up dependencies...", []string{"mod", "tidy"}},
		{"Verifying dependencies...", []string{"mod", "verify"}},
	}

	for _, step := range steps {
		fmt.Println(step.label)
		cmd := exec.Command("go", step.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", step.label, err)
		}
		fmt.Println()
	}

	return nil
}
