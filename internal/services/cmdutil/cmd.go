package cmdutil

import (
	"os"
	"os/exec"
)

// GoCommand returns an exec.Cmd for the go toolchain with GOWORK=off.
//
//nolint:gosec,noctx // G204: trusted subprocess
func GoCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	return cmd
}
