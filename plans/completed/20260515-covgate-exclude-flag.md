# Add `--exclude` flag to covgate for skipping selected Go packages

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` | read-write | Add `Exclude` option to covgate service, wire `--exclude` CLI flag, add tests |
| `backend/` | read-only | Downstream consumer that motivates the flag (no edits here) |

This plan lives in `gotools/plans/backlog/` because all code changes happen in `internal/services/covgate/` and `internal/commands/` inside the `gotools` repo.

## Purpose / Big Picture

`miru covgate` enforces per-package Go coverage thresholds. Today every package matched by `--packages` is measured, even when some of those packages cannot run their tests in the current environment (e.g., packages that talk to AWS, Cloudflare, or other cloud services during tests). Downstream consumers (notably `backend/`) need a way to skip those packages locally while CI still measures everything.

After this change, a user invoking `miru covgate` may pass a `--exclude` flag whose value is a comma-separated list of Go list patterns. Those patterns are resolved through `go list` and the resulting import paths are removed from the measurement set before tests run. Existing invocations (no `--exclude` flag, or `--exclude=""`) behave identically to today.

Observable behavior after the change, run from a module whose `./...` enumerates `pkg/a`, `pkg/b`, `pkg/c`:

    $ go tool miru covgate --packages ./... --exclude ./pkg/b
    Checking per-package coverage (default minimum: 80.0%)...

    Excluded 1 package from coverage measurement
    STATUS  COVERAGE  REQUIRED      TIME  PACKAGE
    ------  --------  --------  --------  -------
    PASS       90.0%     80.0%      0.5s  pkg/a
    PASS       90.0%     80.0%      0.5s  pkg/c

    Total time: 1.0s
    All packages meet minimum coverage requirement

And the no-exclude form is byte-for-byte unchanged from today:

    $ go tool miru covgate --packages ./...
    Checking per-package coverage (default minimum: 80.0%)...

    STATUS  COVERAGE  REQUIRED      TIME  PACKAGE
    ...

## Progress

- [ ] Milestone 1 — Add `Exclude` field to `Opts`, implement set-subtract logic in `(*runner).run`, including the "Excluded N package(s) from coverage measurement" notice
- [ ] Milestone 2 — Wire `--exclude` flag in `internal/commands/covgate.go` and update `internal/commands/commands_test.go`
- [ ] Milestone 3 — Add table-driven tests in `internal/services/covgate/covgate_test.go` covering the five exclusion scenarios
- [ ] Milestone 4 — Run preflight (lint + covgate + surface lint) and ensure it reports clean; fix anything flagged

Use timestamps when you complete steps. Split partially completed work into "done" and "remaining" as needed.

## Surprises & Discoveries

(Add entries as work proceeds.)

## Decision Log

- Decision: Resolve exclusion patterns by re-invoking `goListPackages` for each comma-separated entry rather than literal string matching against import paths.
  Rationale: The flag's value should accept the same dialect as `--packages` (e.g., `./internal/pkg/aws/...`), so it must go through `go list` to expand wildcards correctly. Treating the flag's elements as raw import paths would silently fail for any `/...` pattern.
  Date/Author: 2026-05-15, planning subagent.

- Decision: Print the "Excluded N package(s)" notice only when exclusion actually removed packages, and only after the header line but before `printHeader`.
  Rationale: A no-op exclusion (pattern matched nothing) should not pollute output. Placing the notice before the table header keeps it adjacent to the existing intro line and away from per-package rows.
  Date/Author: 2026-05-15, planning subagent.

- Decision: Empty/unset `Exclude` preserves byte-identical existing output.
  Rationale: Required to avoid regressing every existing caller and CI invocation.
  Date/Author: 2026-05-15, planning subagent.

- Decision: Trim whitespace around each comma-separated entry, and ignore empty entries after trimming.
  Rationale: Tolerates `"a, b,  c"`-style input without surprising failures, matching common CLI conventions.
  Date/Author: 2026-05-15, planning subagent.

## Outcomes & Retrospective

(Summarize at completion or major milestones.)

## Context and Orientation

Covgate is a Go test-coverage gate exposed as the `covgate` subcommand of the `miru` CLI (built from `cmd/miru/main.go`). The reader should treat the following files as authoritative starting points; all paths are relative to the repo root `gotools/`.

Service layer:

- `internal/services/covgate/covgate.go` — service logic.
  - `Opts` struct currently has fields `Packages`, `SrcPrefix`, `TestDir`, `DefaultThreshold`, `Parallelism`, `TightnessEnabled`, `TightnessTolerance`, `Out`.
  - `runner` struct injects three functions for testability: `goModule`, `goListPackages`, `measure`. Tests substitute their own implementations.
  - `Run(opts Opts) error` wires in the real `gocover.GoListPackages` and `gocover.GoModule`, then calls `(*runner).run`.
  - `(*runner).run` calls `r.goListPackages(opts.Packages)` to get the package list, prints `printHeader`, then `runPackages` → `printResults`.
- `internal/services/gocover/gocover.go` — `GoListPackages(pattern string) ([]string, error)` runs `go list <pattern>` and returns trimmed non-empty lines.

CLI layer:

- `internal/commands/covgate.go` — `NewCovgateCommand()` returns the cobra command. It already binds each `Opts` field to a `--<name>` flag using `cmd.Flags()`. Default `Out` is `os.Stdout`.
- `internal/commands/root.go` — registers `NewCovgateCommand()` under the `miru` root via `cmd.AddCommand`.

Tests:

- `internal/services/covgate/covgate_test.go` — uses a `runner{}` literal with injected `goModule`, `goListPackages`, `measure`. Existing tests like `TestRun_AllPass` and `TestRun_WithFailure` show the established pattern.
- `internal/commands/commands_test.go` — `TestNewCovgateCommand_Flags` and `TestNewCovgateCommand_FlagDefaults` assert each flag's existence, type, and default value. They must be updated for the new `--exclude` flag.

Validation scripts (all under `scripts/`):

- `scripts/preflight.sh` — runs `lint.sh`, `covgate.sh`, and `lint-surface.sh` in parallel; exits non-zero if any fails. **This is the canonical "preflight" command.** A clean preflight is mandatory before publishing changes.
- `scripts/lint.sh` — Go lint (gofumpt + golangci-lint).
- `scripts/covgate.sh` — runs `go tool miru covgate` against this repo's own packages.
- `scripts/lint-surface.sh` — exported-API stability check.

Glossary:

- **Go list pattern**: the string form `go list` accepts on its command line, e.g., `./...`, `./pkg/...`, or `example.com/mod/internal/foo`. Wildcards expand to one import path per matched package.
- **Set-subtract**: build a set `E` of import paths from exclude patterns; rebuild the main list keeping order, dropping any path that appears in `E`.
- **Preflight clean**: `./scripts/preflight.sh` exits 0 and prints `=== All checks passed ===`.

## Plan of Work

### Milestone 1 — Service-layer changes

Edit `internal/services/covgate/covgate.go`:

1. Add a new field to `Opts`:

       Exclude string

   Place it immediately after `Packages` so related fields stay together. Document semantics in a comment: comma-separated Go list patterns; empty means no exclusion.

2. In `(*runner).run`, after the existing `pkgs, err := r.goListPackages(opts.Packages)` block but before `printHeader(w)`, insert logic that:
   - Returns early (no change) when `strings.TrimSpace(opts.Exclude) == ""`.
   - Splits `opts.Exclude` on commas, trims whitespace from each entry, drops empty entries.
   - For each non-empty entry, calls `r.goListPackages(entry)` to resolve the pattern, collecting all returned import paths into a `map[string]struct{}` named `excluded`.
   - If any `r.goListPackages` call returns an error, propagate it with `fmt.Errorf("resolve exclude %q: %w", entry, err)`.
   - Rebuilds `pkgs` by iterating the original slice and keeping only entries whose import path is not in `excluded` (preserves order, deduplicates trivially because the original list contains unique paths).
   - If the rebuilt length is smaller than the original, print one line to `w`:

         Excluded N package(s) from coverage measurement

     using `fmt.Fprintf` and the actual count. Only print when the count is `> 0` — a pattern matching nothing produces no notice.

3. The rebuilt `pkgs` flows into the existing `printHeader(w)` and `runPackages(...)` paths unchanged. When `pkgs` is empty after exclusion, the existing code in `runPackages` (a zero-length loop) and `printResults` (no failures) already produces a clean "All packages meet minimum coverage requirement" outcome with `Total time: 0.0s` — verify by test rather than adding new branches.

### Milestone 2 — CLI wiring

Edit `internal/commands/covgate.go`:

1. Inside `NewCovgateCommand`, after the existing `fl.StringVar(&opts.Packages, ...)` line, add:

       fl.StringVar(
           &opts.Exclude, "exclude", "",
           "comma-separated list of Go list patterns to exclude from coverage measurement",
       )

2. No other CLI changes are needed; cobra already binds the value into `opts.Exclude` and passes it through to `covgate.Run` via the existing `RunE` closure.

Edit `internal/commands/commands_test.go`:

1. In `TestNewCovgateCommand_Flags`, extend the `tests` table with `{"exclude", "string"}`.
2. In `TestNewCovgateCommand_FlagDefaults`, extend `stringDefaults` with `"exclude": ""`.

### Milestone 3 — Service tests

Edit `internal/services/covgate/covgate_test.go`. Add a new `TestRun_Exclude_*` family modeled on `TestRun_AllPass`. The test file already imports `bytes`, `strings`, and `testing`; the new tests need no additional imports.

Use a single table-driven helper plus per-case `t.Run` subtests. For each subtest:

- Provide a stub `goListPackages` that returns a fixed list when called with the `Packages` arg and a different (or empty) list when called with each exclude pattern. Easiest pattern: a `map[string][]string` keyed on the pattern, with a closure that looks it up.
- Inject `goModule` returning `modName`.
- Inject `measure` with `fakeMeasure(90.0)` (always pass).
- Pass `Opts{Out: &buf, DefaultThreshold: 80.0, Packages: "...", Exclude: "..."}`.

Cover these scenarios — each is one subtest in the table:

1. **No exclude → existing behavior unchanged.** `Exclude: ""`, three packages `pkg/a`, `pkg/b`, `pkg/c` from `goListPackages("./...")`. Assert: all three appear in output; "Excluded" notice does **not** appear; `err == nil`.

2. **Subset exclusion.** `Exclude: "./pkg/b"`, three packages from `./...`, exclude pattern returns `["example.com/mod/pkg/b"]`. Assert: `pkg/a` and `pkg/c` appear; `pkg/b` does **not** appear in output; "Excluded 1 package(s)" notice appears; `err == nil`.

3. **No-op exclude pattern.** `Exclude: "./does-not-exist/..."`, exclude pattern returns `[]`. Assert: all three packages appear; "Excluded" notice does **not** appear (exclusion removed nothing); `err == nil`.

4. **Multiple patterns with whitespace.** `Exclude: "./pkg/a, ./pkg/c"` (note the space after the comma). Exclude lookup must trim whitespace. Assert: only `pkg/b` appears; "Excluded 2 package(s)" notice appears; `err == nil`.

5. **Exclude-all.** `Exclude: "./..."` returns the full three-package list. Assert: no per-package output rows; "Excluded 3 package(s)" notice appears; output contains `All packages meet minimum coverage requirement` and `Total time:`; `err == nil`.

Also add one error-path test, `TestRun_Exclude_GoListError`, modeled on `TestRun_GoListError`:

- `goListPackages` returns the package list correctly for `Packages` but returns `error` for the exclude pattern.
- Assert: `err != nil`; error message contains the pattern string and the wrapped underlying error message (sanity-check the `fmt.Errorf("resolve exclude %q: %w", ...)` format).

Use `//nolint:exhaustruct` on `runner` and `Opts` literals to match the existing test style.

### Milestone 4 — Validation pass

Run preflight from the repo root and fix any findings. The plan considers the work complete only when preflight reports clean (see Validation and Acceptance below for the exact expectation).

## Concrete Steps

All commands are run from `/home/ben/miru/workbench2/repos/gotools/` unless stated otherwise. The current branch is `feat/covgate-exclude-flag`; do not switch branches.

### Step 0 — Sanity baseline

From `gotools/`:

    git status

Expected: `On branch feat/covgate-exclude-flag` and a clean working tree (or only this plan file as a tracked addition).

    go build ./...

Expected: completes silently with exit 0.

    go test ./internal/services/covgate/... ./internal/commands/...

Expected: `ok` for both packages.

### Step 1 — Implement Milestone 1 (service)

Edit `internal/services/covgate/covgate.go` per "Plan of Work — Milestone 1" above.

Sanity check the build:

    go build ./internal/services/covgate/...

Expected: exit 0, no output.

Commit:

    git add internal/services/covgate/covgate.go
    git commit -m "feat(covgate): support --exclude to skip packages from coverage measurement"

### Step 2 — Implement Milestone 2 (CLI + cmd tests)

Edit `internal/commands/covgate.go` and `internal/commands/commands_test.go` per "Plan of Work — Milestone 2".

Run:

    go build ./...
    go test ./internal/commands/...

Expected: both pass. The new flag tests should pass against the wired-up flag.

Commit:

    git add internal/commands/covgate.go internal/commands/commands_test.go
    git commit -m "feat(cli): expose covgate --exclude flag"

### Step 3 — Implement Milestone 3 (service tests)

Edit `internal/services/covgate/covgate_test.go` per "Plan of Work — Milestone 3".

Run:

    go test ./internal/services/covgate/...

Expected output (line counts approximate):

    ok  	github.com/mirurobotics/gotools/internal/services/covgate	0.5s

If any subtest fails, inspect the buffer assertions; the most common pitfall is forgetting that the "Excluded" notice should *not* appear when zero packages were removed.

Commit:

    git add internal/services/covgate/covgate_test.go
    git commit -m "test(covgate): cover --exclude semantics across no-op, subset, multi, and full removal"

### Step 4 — Milestone 4 preflight

From `gotools/`:

    ./scripts/preflight.sh

Expected last line:

    === All checks passed ===

and exit code 0.

If preflight is not clean:

- Read the offending output (lint, covgate, or surface lint).
- Apply fixes in place. Common items: gofumpt formatting, golangci-lint findings on the new test file, any new exported symbol catching surface-lint attention.
- Rerun preflight. Repeat until clean.

Commit any fixes:

    git add -A
    git commit -m "chore(covgate): preflight fixes"

Skip this commit if preflight was clean on the first run.

## Validation and Acceptance

Acceptance is observable behavior plus a clean preflight, in this order. **Changes MUST NOT be published (no push, no PR) until preflight reports clean.**

1. **Service tests pass.** From `gotools/`:

       go test ./internal/services/covgate/...

   Expected: `ok` line; the new tests `TestRun_Exclude_NoExclude`, `TestRun_Exclude_Subset`, `TestRun_Exclude_NoOpPattern`, `TestRun_Exclude_MultiplePatternsWithWhitespace`, `TestRun_Exclude_AllPackages`, and `TestRun_Exclude_GoListError` all pass. Before this change those tests do not exist; after, they pass.

2. **Command tests pass.**

       go test ./internal/commands/...

   Expected: `ok`. The updated `TestNewCovgateCommand_Flags` and `TestNewCovgateCommand_FlagDefaults` see the new `exclude` flag with default `""`.

3. **Build is clean.**

       go build ./...

   Expected: exit 0, no output.

4. **Preflight is clean — mandatory gate.** From `gotools/`:

       ./scripts/preflight.sh

   Expected final line: `=== All checks passed ===` with exit 0. **No changes are published, no PR is opened, and no branch is pushed until this command reports clean.** If it fails on any of lint, covgate, or surface lint, fix the underlying issue and rerun. Do not bypass preflight.

5. **End-to-end smoke (manual).** Build the `miru` binary and run against the gotools repo itself, demonstrating the new flag:

       go build -o /tmp/miru ./cmd/miru
       /tmp/miru covgate --packages ./... --exclude ./internal/services/gocover/...

   Expected: the output contains the line `Excluded N package(s) from coverage measurement` for some `N >= 1`, and no row mentions a `gocover` package. Without `--exclude`, those packages appear normally. (This step is for manual verification only — not part of automated tests.)

## Idempotence and Recovery

- Service and CLI edits (Milestones 1–2): purely additive. Re-running the edits is safe; if a step is partially applied, re-read the file and reconcile against the "Plan of Work" instructions.
- Test additions (Milestone 3): if Go reports duplicate function names after a retry, remove the half-applied block and re-paste from this plan.
- Preflight (Milestone 4): idempotent — re-running `./scripts/preflight.sh` repeatedly is safe and reflects the current working tree.
- Commits: each milestone has its own commit (see Concrete Steps). If a commit needs to be redone, prefer adding a new commit over `git commit --amend` so the milestone boundary stays visible in PR review. To discard uncommitted changes for a fresh retry, use `git restore <path>` on specific files — do **not** use `git reset --hard` because it discards work in unrelated files.
- Rollback path: revert the milestone commits in reverse order with `git revert <sha>` (per-commit) if the change must be removed after merge. The change is purely additive (new flag, new tests, no semantic change when flag is unset), so reverting is low-risk.
