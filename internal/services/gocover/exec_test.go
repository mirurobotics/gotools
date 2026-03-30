package gocover

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecOutput_NilErrWriter(t *testing.T) {
	out, err := ExecOutput(nil, "echo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestExecOutput_CustomErrWriter(t *testing.T) {
	var buf bytes.Buffer
	out, err := ExecOutput(&buf, "echo", "world")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out); got != "world" {
		t.Errorf("got %q, want %q", got, "world")
	}
}

func TestExecOutput_Failure(t *testing.T) {
	var buf bytes.Buffer
	_, err := ExecOutput(&buf, "false")
	if err == nil {
		t.Error("expected error from false command")
	}
}
