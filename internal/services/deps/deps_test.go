package deps

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func fakeExec(err error) func(io.Writer, io.Writer, ...string) error {
	return func(io.Writer, io.Writer, ...string) error { return err }
}

func fakeExecFailAt(n int) func(io.Writer, io.Writer, ...string) error {
	call := 0
	return func(io.Writer, io.Writer, ...string) error {
		call++
		if call == n {
			return fmt.Errorf("step %d failed", n)
		}
		return nil
	}
}

func TestRunUpdate_Success(t *testing.T) {
	var out, errW bytes.Buffer
	r := runner{exec: fakeExec(nil)}

	err := r.runUpdate(&out, &errW)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	labels := []string{
		"Updating dependencies...",
		"Tidying up dependencies...",
		"Verifying dependencies...",
	}
	for _, label := range labels {
		if !strings.Contains(out.String(), label) {
			t.Errorf("output missing label %q: %s", label, out.String())
		}
	}
}

func TestRunUpdate_FirstStepFails(t *testing.T) {
	var out, errW bytes.Buffer
	r := runner{exec: fakeExecFailAt(1)}

	err := r.runUpdate(&out, &errW)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Updating dependencies...") {
		t.Errorf("error should contain first step label: %v", err)
	}
	if !strings.Contains(err.Error(), "step 1 failed") {
		t.Errorf("error should wrap cause: %v", err)
	}
}

func TestRunUpdate_MiddleStepFails(t *testing.T) {
	var out, errW bytes.Buffer
	r := runner{exec: fakeExecFailAt(2)}

	err := r.runUpdate(&out, &errW)
	if err == nil {
		t.Fatal("expected error")
	}

	// First step should have run and printed its label.
	if !strings.Contains(out.String(), "Updating dependencies...") {
		t.Errorf("first step label missing from output: %s", out.String())
	}

	// Error should reference the second step label.
	if !strings.Contains(err.Error(), "Tidying up dependencies...") {
		t.Errorf("error should contain second step label: %v", err)
	}
	if !strings.Contains(err.Error(), "step 2 failed") {
		t.Errorf("error should wrap cause: %v", err)
	}

	// Third step should not have run.
	if strings.Contains(out.String(), "Verifying dependencies...") {
		t.Error("third step should not have run")
	}
}

func TestRunUpdate_NilWriters(t *testing.T) {
	r := runner{exec: fakeExec(nil)}

	// Should not panic with nil writers.
	err := r.runUpdate(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
