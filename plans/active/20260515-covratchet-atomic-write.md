# covratchet: atomic write to `.covgate`

## Scope

| Repo       | Access     | Notes                                                    |
|------------|------------|----------------------------------------------------------|
| `gotools/` | read-write | Edits one source file and one test file in `internal/services/covratchet/`. |

No other repositories are read or modified.

## Purpose / Big Picture

`covratchet` maintains per-package coverage baselines by writing a single number (e.g. `78.4\n`) to a file named `.covgate` next to each Go package. Today that write is `os.WriteFile`, which truncates the destination before writing the new bytes. If the process is interrupted between truncate and final write — by SIGINT, OOM kill, crash, or a host-level power loss — the package is left with an empty or partial `.covgate`. The next run reads `""` and treats the package as having no baseline, silently dropping the floor.

After this change, `writeCovgate` writes to a tempfile in the same directory and then `os.Rename`s the tempfile over the target. POSIX guarantees `rename(2)` is atomic within a filesystem, so a reader always sees either the prior valid content or the new valid content, never a truncated file.

**How to see it working:** in a scratch repo with a tracked package, run covratchet under `kill -INT` mid-run for a few iterations. Before the fix, `.covgate` will occasionally be empty (`wc -c .covgate` reports `0`). After the fix, `.covgate` always contains a well-formed `NN.N\n` line.

## Progress

- [x] Milestone 1: replace `writeCovgate` body with tempfile + rename. (2026-05-15)
- [x] Milestone 2: add two new tests, update one existing test. (2026-05-15)

## Surprises & Discoveries

_None yet. Populate during implementation._

## Decision Log

_None yet. Populate during implementation as non-trivial decisions arise._

## Outcomes & Retrospective

_To be filled at completion._

## Context and Orientation

**Term: `.covgate`.** A per-package text file holding a single float (one decimal place) followed by a newline — the coverage percentage that covratchet has ratcheted up to for that package. It lives at `<pkgDir>/.covgate` and is committed to the repo so the floor moves forward across runs and CI invocations.

**Files of interest** (all absolute, relative to repo root `/home/ben/miru/workbench1/repos/gotools`):

- `internal/services/covratchet/covratchet.go` — the writer and its callers.
  - `writeCovgate` at lines 206–210 — the function to modify.
  - `readCovgate` at lines 197–204 — returns `""` on any read error; takes the first line trimmed. Unchanged by this work, but relevant: it is the function that silently treats a truncated `.covgate` as "no baseline".
  - Callsites at line 170 (`ratchetPackage`, ratchet-up path) and line 186 (`writeNewCovgate`, called when `readCovgate` returned `""`).
- `internal/services/covratchet/covratchet_test.go` — the existing test file.
  - `TestWriteCovgate` lines 47–63 (happy path) and `TestWriteCovgate_Error` lines 81–86 (writes to `/nonexistent/dir/`) — leave alone.
  - `TestRatchetPackage_New_WriteError` lines 142–162 — already chmods the parent directory to `0o555`. Reference pattern for the test we will update.
  - `TestRatchetPackage_Up_WriteError` lines 216–238 — currently chmods the file `.covgate` itself to `0o444`. Must be updated (see Milestone 2).
  - `TestRun_Parallelism` lines 334–388 — exercises the goroutine pool. Should continue to pass.

**Concurrency model.** `(*runner).run` (`covratchet.go:89-102`) fans out one goroutine per package, bounded by a semaphore. Each goroutine writes to a distinct package directory, so there is **no cross-goroutine contention on any single `.covgate` path**. Atomic rename here only needs to defend against interrupted writes (SIGINT, kill, crash) and partial disk writes — not against concurrent writers.

**No existing atomic-write helper exists in this repo.** `gocover.go:146` uses `os.CreateTemp` for a transient coverage profile but not as a reusable helper. We will inline the temp-file-and-rename pattern in `writeCovgate`; it is a few lines and a dedicated helper would be premature abstraction for one callsite.

**Error wrapping idiom.** Use `fmt.Errorf("... %w", err)` — no custom error package.

**Lint markers.** The current `writeCovgate` carries `//nolint:gosec // G306: intentional 0644`. The rewritten version sets the mode explicitly via `tmpFile.Chmod(0o644)` and carries the same marker on the `Chmod` line.

## Plan of Work

### Milestone 1 — atomic write

Replace the body of `writeCovgate` in `internal/services/covratchet/covratchet.go` with a tempfile-and-rename sequence:

1. Compute `dir := filepath.Dir(path)`. The tempfile must live in the same directory as the destination so that the eventual `os.Rename` stays inside one filesystem (cross-filesystem renames fail with `EXDEV`).
2. Create the tempfile with `os.CreateTemp(dir, ".covgate.tmp-*")`. The `.covgate.tmp-*` prefix is hidden, package-scoped, and obviously transient — easy to spot if one leaks.
3. Set mode `0o644` via `tmpFile.Chmod(0o644)` so the result matches today's permissions regardless of umask. Carry the existing `//nolint:gosec // G306: intentional 0644` marker on this line.
4. Write the formatted content (`fmt.Sprintf("%.1f\n", coverage)`) to the tempfile.
5. Close the tempfile.
6. `os.Rename(tmpFile.Name(), path)`.
7. On any error after the tempfile is created, `os.Remove(tmpFile.Name())` before returning. Wrap the returned error with `fmt.Errorf("... %w", err)` so callers see context. Use a `defer` with a named return error, or an explicit cleanup branch — either is fine; pick whichever keeps the function short.

The function signature stays `func writeCovgate(path string, coverage float64) error`. No callers change. Add `"path/filepath"` to the import block — it is not currently imported in `covratchet.go` (the existing imports are `fmt`, `io`, `os`, `runtime`, `strconv`, `strings`, `sync`, and `github.com/mirurobotics/gotools/internal/services/gocover`).

### Milestone 2 — tests

In `internal/services/covratchet/covratchet_test.go`:

1. **Add `TestWriteCovgate_NoLeftoverTempfile`.** Happy-path test: create a temp dir via `t.TempDir()`, call `writeCovgate(filepath.Join(dir, ".covgate"), 42.5)`, assert no error, assert `.covgate` contains `"42.5\n"`, then list `dir` and assert no entry matches `.covgate.tmp-*`. This guards against the "we forgot to clean up on success" failure mode.

2. **Add `TestWriteCovgate_PreservesExistingOnFailure`.** Create a temp dir, pre-populate `.covgate` with `"55.0\n"` (use the existing `testutil.WriteCovgateFile` helper or a direct `os.WriteFile`). Chmod the directory to `0o555` so `os.CreateTemp` fails (no write permission on the directory). Register a `t.Cleanup` that chmods the directory back to `0o755` so the test framework can remove it. Call `writeCovgate(filepath.Join(dir, ".covgate"), 99.9)`; assert it returns a non-nil error. Re-chmod the directory back to `0o755` in-test (before reading) and assert that `.covgate` still contains `"55.0\n"` — the original value is untouched because the rename never happened.

3. **Update `TestRatchetPackage_Up_WriteError`** at lines 216–238. Today it pre-writes `.covgate` and then `os.Chmod(covgatePath, 0o444)` to force the write to fail. Under atomic rename, this no longer fails — POSIX `rename(2)` only requires write permission on the destination *directory*, not the destination file. Change the chmod to target the package directory: `os.Chmod(pkgDir, 0o555)`. Add a `t.Cleanup` that restores `0o755` so test teardown can remove the dir. This matches the existing pattern in `TestRatchetPackage_New_WriteError` (lines 142–162).

No other tests need updating. `TestWriteCovgate_Error` (writing to `/nonexistent/dir/`) still fails for the right reason — `os.CreateTemp` on a missing directory returns an error.

## Concrete Steps

All commands run from the repo root `/home/ben/miru/workbench1/repos/gotools` unless noted. Branch `fix/covratchet-atomic-write` is already checked out.

### Milestone 1: implement atomic write

1. Open `internal/services/covratchet/covratchet.go` and replace the body of `writeCovgate` (currently lines 206–210). Target shape:

        func writeCovgate(path string, coverage float64) (err error) {
                content := fmt.Sprintf("%.1f\n", coverage)
                dir := filepath.Dir(path)
                tmpFile, err := os.CreateTemp(dir, ".covgate.tmp-*")
                if err != nil {
                        return fmt.Errorf("create tempfile for %s: %w", path, err)
                }
                tmpName := tmpFile.Name()
                defer func() {
                        if err != nil {
                                _ = os.Remove(tmpName)
                        }
                }()
                //nolint:gosec // G306: intentional 0644
                if err = tmpFile.Chmod(0o644); err != nil {
                        _ = tmpFile.Close()
                        return fmt.Errorf("chmod tempfile %s: %w", tmpName, err)
                }
                if _, err = tmpFile.WriteString(content); err != nil {
                        _ = tmpFile.Close()
                        return fmt.Errorf("write tempfile %s: %w", tmpName, err)
                }
                if err = tmpFile.Close(); err != nil {
                        return fmt.Errorf("close tempfile %s: %w", tmpName, err)
                }
                if err = os.Rename(tmpName, path); err != nil {
                        return fmt.Errorf("rename tempfile to %s: %w", path, err)
                }
                return nil
        }

   Add `"path/filepath"` to the import block at the top of `covratchet.go`; it is not currently imported (existing imports: `fmt`, `io`, `os`, `runtime`, `strconv`, `strings`, `sync`, plus the internal `gocover` package). Place it alphabetically between `"os"` and `"runtime"`.

2. Run the covratchet package tests:

        go test ./internal/services/covratchet/...

   Expected: all tests pass. Note that `TestRatchetPackage_Up_WriteError` will now FAIL after this implementation change — that is expected and is fixed in Milestone 2. If only that single test fails, proceed; if anything else fails, stop and investigate.

3. Commit. Stage and commit only the source file in this milestone:

        git add internal/services/covratchet/covratchet.go
        git commit -m "fix(covratchet): atomic write to .covgate"

   Commit message body (one short paragraph) should note: write to tempfile and rename so interrupted writes (SIGINT, crash) cannot leave a truncated `.covgate`.

### Milestone 2: tests

1. Open `internal/services/covratchet/covratchet_test.go`. Add the two new tests described in Plan of Work (Milestone 2 items 1 and 2). Place them near the existing `TestWriteCovgate` / `TestWriteCovgate_Error` block for proximity.

   Sketch for `TestWriteCovgate_NoLeftoverTempfile`:

        func TestWriteCovgate_NoLeftoverTempfile(t *testing.T) {
                dir := t.TempDir()
                path := filepath.Join(dir, ".covgate")
                if err := writeCovgate(path, 42.5); err != nil {
                        t.Fatalf("writeCovgate: %v", err)
                }
                //nolint:gosec // G304: test file read
                content, err := os.ReadFile(path)
                if err != nil {
                        t.Fatalf("read .covgate: %v", err)
                }
                if string(content) != "42.5\n" {
                        t.Fatalf("got %q, want %q", string(content), "42.5\n")
                }
                entries, err := os.ReadDir(dir)
                if err != nil {
                        t.Fatalf("read dir: %v", err)
                }
                for _, e := range entries {
                        if strings.HasPrefix(e.Name(), ".covgate.tmp-") {
                                t.Fatalf("tempfile leaked: %s", e.Name())
                        }
                }
        }

   Sketch for `TestWriteCovgate_PreservesExistingOnFailure`:

        func TestWriteCovgate_PreservesExistingOnFailure(t *testing.T) {
                dir := t.TempDir()
                path := filepath.Join(dir, ".covgate")
                //nolint:gosec // G306: test file
                if err := os.WriteFile(path, []byte("55.0\n"), 0o644); err != nil {
                        t.Fatalf("seed .covgate: %v", err)
                }
                //nolint:gosec // G302: test file permissions
                if err := os.Chmod(dir, 0o555); err != nil {
                        t.Fatalf("chmod dir: %v", err)
                }
                //nolint:gosec // G302: test file permissions
                t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
                if err := writeCovgate(path, 99.9); err == nil {
                        t.Fatalf("expected error on read-only dir, got nil")
                }
                //nolint:gosec // G302: test file permissions
                if err := os.Chmod(dir, 0o755); err != nil {
                        t.Fatalf("restore dir mode: %v", err)
                }
                //nolint:gosec // G304: test file read
                content, err := os.ReadFile(path)
                if err != nil {
                        t.Fatalf("read .covgate: %v", err)
                }
                if string(content) != "55.0\n" {
                        t.Fatalf("original .covgate clobbered: got %q, want %q", string(content), "55.0\n")
                }
        }

   `"strings"` is already imported in the test file (line 9) — no import change needed.

2. Update `TestRatchetPackage_Up_WriteError` (lines 216–238). The existing test uses local variables `dir` (the package directory from `testutil.MakePkgDir`) and `covFile` (the joined `.covgate` path). Replace the two-line chmod of `covFile` (lines 221–222) and its cleanup (lines 225–226) so the chmod targets `dir` instead, mirroring `TestRatchetPackage_New_WriteError`:

        // before:
        // //nolint:gosec // G302: test file permissions
        // if err := os.Chmod(covFile, 0o444); err != nil {
        //         t.Fatal(err)
        // }
        // //nolint:gosec // G302: test file permissions
        // t.Cleanup(func() { _ = os.Chmod(covFile, 0o644) })
        // after:
        //nolint:gosec // G302: test file permissions
        if err := os.Chmod(dir, 0o555); err != nil {
                t.Fatal(err)
        }
        //nolint:gosec // G302: test file permissions
        t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

   The `covFile` local can be removed if no longer referenced after this edit.

3. Run the covratchet tests:

        go test ./internal/services/covratchet/...

   Expected output (abridged):

        ok      github.com/.../internal/services/covratchet     0.0XXs

   With PASS for `TestWriteCovgate`, `TestWriteCovgate_Error`, `TestWriteCovgate_NoLeftoverTempfile`, `TestWriteCovgate_PreservesExistingOnFailure`, `TestRatchetPackage_New_WriteError`, `TestRatchetPackage_Up_WriteError`, `TestRun_Parallelism`, and the rest of the package.

4. Run the full test suite from the repo root:

        go test ./...

   Expected: all packages PASS.

5. Commit. Stage and commit only the test file:

        git add internal/services/covratchet/covratchet_test.go
        git commit -m "test(covratchet): cover atomic .covgate write"

## Validation and Acceptance

- `go test ./...` from the repo root must report no failures.
- The two new tests (`TestWriteCovgate_NoLeftoverTempfile`, `TestWriteCovgate_PreservesExistingOnFailure`) must FAIL against the unmodified `writeCovgate` (sanity check the test value: temporarily revert the implementation, run the tests, confirm failures, then restore) and PASS against the new implementation.
- The updated `TestRatchetPackage_Up_WriteError` must continue to exercise the write-error path (the package directory is read-only, so `os.CreateTemp` fails).
- **`./scripts/preflight.sh` MUST report clean before pushing the branch or opening a PR.** Preflight mirrors CI (lint + tests + any repo-specific checks); a green local preflight is the gate.
- PR title: `fix(covratchet): atomic write to .covgate`.
- PR body must reference the 2026-05-15 covratchet design audit that surfaced this gap, and must briefly note the test update for `TestRatchetPackage_Up_WriteError` (chmod target moved from file to directory).

## Idempotence and Recovery

- The two edit-and-commit milestones are idempotent on a clean working tree: re-running the steps after a successful commit produces a no-op (nothing to commit). If a commit step is repeated mid-milestone, `git status` will show which files remain unstaged.
- `go test` is purely a read of the working tree; re-run freely on failure.
- If Milestone 1's implementation introduces a regression, rollback is `git checkout -- internal/services/covratchet/covratchet.go` (before commit) or `git reset --hard HEAD~1` for the most recent commit (only on the feature branch, never on `main`).
- If Milestone 2's tests don't compile, `git checkout -- internal/services/covratchet/covratchet_test.go` restores the prior state. The Milestone 1 commit remains intact.
- The tempfile cleanup branch in the new `writeCovgate` is itself idempotent: `os.Remove` on a path that never existed (or was already renamed away) returns an error we deliberately ignore via `_ =`, so re-invocation under partial failure does not poison the directory.
- No destructive filesystem operations occur in any step. All chmods in tests are scoped to `t.TempDir()` paths, which are torn down automatically.
