package gotest

import (
	"io"
	"os"
	"os/exec"
)

// Opts holds the options for the gotest service.
type Opts struct {
	ExtraArgs []string
	Out       io.Writer
	Err       io.Writer
}

// Run executes go test ./... with optional extra args.
func Run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

	args := buildArgs(opts.ExtraArgs)

	//nolint:gosec,noctx // G204: trusted subprocess
	cmd := exec.Command("go", args...)
	cmd.Stdout = opts.Out
	cmd.Stderr = opts.Err

	return cmd.Run()
}

// buildArgs constructs the go test argument list, stripping
// a leading "--" separator if present.
func buildArgs(extra []string) []string {
	if len(extra) > 0 && extra[0] == "--" {
		extra = extra[1:]
	}
	args := make([]string, 0, 2+len(extra))
	args = append(args, "test", "./...")
	return append(args, extra...)
}
