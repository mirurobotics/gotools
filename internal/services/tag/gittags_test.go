package tag

import (
	"bytes"
	"io"
	"testing"
)

func TestGitTags_NilErrWriter(t *testing.T) {
	// We're in a git repo, so this should succeed.
	tags, err := GitTags(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Just verify it returns without error; tag list
	// content depends on the repo state.
	_ = tags
}

func TestGitTags_CustomErrWriter(t *testing.T) {
	var buf bytes.Buffer
	_, err := GitTags(&buf)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCurrent_NilErrWriter(t *testing.T) {
	// HEAD likely has no tag; we just exercise the
	// nil-writer default path.
	_, _ = Current(nil)
}

func TestCurrent_CustomErrWriter(t *testing.T) { _, _ = Current(io.Discard) }
