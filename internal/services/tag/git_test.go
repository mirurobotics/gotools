package tag

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// initRepo creates a git repository in a fresh temp directory, makes it
// the working directory for the test, and adds one commit.
func initRepo(t *testing.T) {
	t.Helper()
	isolateGit(t)
	enterDir(t, t.TempDir())
	runGit(t, "init", "-q")
	runGit(t,
		"-c", "user.name=test", "-c", "user.email=test@example.com",
		"commit", "-q", "--allow-empty", "-m", "init",
	)
}

// isolateGit keeps git away from the caller's repository and config:
// no inherited GIT_DIR/GIT_WORK_TREE, and no global or system config
// (e.g. commit.gpgsign).
func isolateGit(t *testing.T) {
	t.Helper()
	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// enterDir makes dir the working directory and stops git from
// discovering a repository above it.
func enterDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	t.Chdir(dir)
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
	if got, err := Current(nil); err != nil || !slices.Contains(tags, got) {
		t.Errorf("Current = %q, %v; want one of %v", got, err, tags)
	}
}

func TestGitBackedHelpers_Errors(t *testing.T) {
	initRepo(t)
	if _, err := Current(io.Discard); err == nil {
		t.Error("Current: want error when HEAD has no tag")
	}
	enterDir(t, t.TempDir())
	if _, err := GitTags(io.Discard); err == nil {
		t.Error("GitTags: want error outside a git repository")
	}
}
