package deps

import (
	"fmt"
	"io"

	"github.com/mirurobotics/gotools/internal/services/cmdutil"
)

type runner struct {
	exec func(out io.Writer, errW io.Writer, args ...string) error
}

func defaultExec(out io.Writer, errW io.Writer, args ...string) error {
	cmd := cmdutil.GoCommand(args...)
	cmd.Stdout = out
	cmd.Stderr = errW
	return cmd.Run()
}

// RunUpdate updates all Go dependencies.
func RunUpdate(out io.Writer, errW io.Writer) error {
	r := runner{exec: defaultExec}
	return r.runUpdate(out, errW)
}

func (r *runner) runUpdate(out io.Writer, errW io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errW == nil {
		errW = io.Discard
	}

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
		if err := r.exec(out, errW, step.args...); err != nil {
			return fmt.Errorf("%s: %w", step.label, err)
		}
		_, _ = fmt.Fprintln(out)
	}

	return nil
}
