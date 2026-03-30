package cmdutil

import (
	"os"
	"os/exec"
)

// GoCommand creates an exec.Cmd for a Go toolchain
// command with GOWORK=off set in the environment.
//
//nolint:gosec,noctx // G204: trusted subprocess
func GoCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	return cmd
}
