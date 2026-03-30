package gotest

import (
	"fmt"
	"os"
	"os/exec"
)

// Opts holds the options for the gotest service.
type Opts struct {
	ExtraArgs []string
}

// Run executes go test ./... with optional extra args.
func Run(opts Opts) error {
	args := []string{"test", "./..."}

	extra := opts.ExtraArgs
	// Strip leading "--" if present (cobra passes it
	// through with DisableFlagParsing).
	if len(extra) > 0 && extra[0] == "--" {
		extra = extra[1:]
	}
	args = append(args, extra...)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "go test failed: %v\n", err)
		os.Exit(1)
	}

	return nil
}
