package deps

import (
	"fmt"
	"io"
	"os/exec"
)

// RunUpdate updates all Go dependencies.
func RunUpdate(out io.Writer, errW io.Writer) error {
	steps := []struct {
		label string
		args  []string
	}{
		{"Updating dependencies...", []string{"get", "-u", "./..."}},
		{"Tidying up dependencies...", []string{"mod", "tidy"}},
		{"Verifying dependencies...", []string{"mod", "verify"}},
	}

	for _, step := range steps {
		_, _ = fmt.Fprintln(out, step.label)
		//nolint:gosec,noctx // G204: trusted subprocess
		cmd := exec.Command("go", step.args...)
		cmd.Stdout = out
		cmd.Stderr = errW
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", step.label, err)
		}
		_, _ = fmt.Fprintln(out)
	}

	return nil
}
