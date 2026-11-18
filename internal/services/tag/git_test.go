package tag

import (
	"io"
	"os/exec"
	"testing"
)

// initRepo creates a git repository in a fresh temp directory, makes it
// the working directory for the test, and adds one commit.
func initRepo(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	runGit(t, "init", "-q")
	runGit(t,
		"-c", "user.name=test", "-c", "user.email=test@example.com",
		"commit", "-q", "--allow-empty", "-m", "init",
	)
}

func runGit(t *testing.T, args ...string) {
	t.Helper()
	//nolint:gosec // G204: git args are fixed by the test
	cmd := exec.CommandContext(t.Context(), "git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGitBackedHelpers(t *testing.T) {
	initRepo(t)
	runGit(t, "tag", "v1.2.0")
	runGit(t, "tag", "v1.10.0")
	runGit(t, "tag", "v1.11.0-rc1")

	tags, err := GitTags(nil)
	if err != nil {
		t.Fatalf("GitTags: %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("GitTags = %v, want 3 tags", tags)
	}
	if got, err := Previous("v", io.Discard); err != nil || got != "v1.10.0" {
		t.Errorf("Previous = %q, %v; want v1.10.0", got, err)
	}
	if got, err := Latest("v", io.Discard); err != nil || got != "v1.11.0-rc1" {
		t.Errorf("Latest = %q, %v; want v1.11.0-rc1", got, err)
	}
	if got, err := Current(nil); err != nil || got == "" {
		t.Errorf("Current = %q, %v; want a tag on HEAD", got, err)
	}
}

func TestGitBackedHelpers_Errors(t *testing.T) {
	initRepo(t)
	if _, err := Current(io.Discard); err == nil {
		t.Error("Current: want error when HEAD has no tag")
	}
	t.Chdir(t.TempDir())
	if _, err := GitTags(io.Discard); err == nil {
		t.Error("GitTags: want error outside a git repository")
	}
}
