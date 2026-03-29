package gotest

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// NewCommand returns a cobra command that runs Go tests.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "test [-- extra args]",
		Short:              "Run Go tests",
		Long:               "Runs go test ./... from the repo root. Arguments after -- are passed through to go test.",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return run(args)
		},
	}

	return cmd
}

func run(extraArgs []string) error {
	args := []string{"test", "./..."}

	// Strip leading "--" if present (cobra passes it through with DisableFlagParsing).
	if len(extraArgs) > 0 && extraArgs[0] == "--" {
		extraArgs = extraArgs[1:]
	}
	args = append(args, extraArgs...)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "go test failed: %v\n", err)
		os.Exit(1)
	}

	return nil
}
