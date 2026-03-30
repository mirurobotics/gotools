package deps

import (
	"fmt"
	"os"
	"os/exec"
)

// RunUpdate updates all Go dependencies.
func RunUpdate() error {
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
