# ExecPlan: Timing Output + --new-from-rev for golangci-lint

## Goal

Two enhancements to the `miru` CLI:

1. **Timing output** — Show elapsed time for each lint step and each covgate package measurement so users can identify bottlenecks.
2. **`--new-from-rev` flag** — Allow `miru lint` to pass `--new-from-rev=<rev>` to golangci-lint, enabling differential linting on PRs.

## Milestones

### M1: Lint step timing

**Files:**
- `internal/services/lint/lint.go` — Add timing around each step in `RunLint`, print summary.
- `internal/services/lint/lint_test.go` — Update tests that assert output content.

**Steps:**
1. Add `"time"` import to `lint.go`.
2. In `RunLint`, record `time.Now()` before each step (custom linter, gofumpt, golangci-lint, deadcode) and compute duration after.
3. Collect step timings into a slice of `{name, duration}` pairs (only for steps that actually ran).
4. After the failure check, print a timing summary section:
   ```
   Timings
   -------
     custom linter    3.2s
     gofumpt          0.4s
     golangci-lint    7m45s
     deadcode         1.1s
   ```
5. Update the `"Lint complete"` message to also include total time.
6. Update tests: `TestRunLint_AllSkipped` and `TestRunLint_EmptyPaths` currently check for `"Lint complete"` — they should still pass since no steps ran and timings section is still printed (empty or with 0 entries).

### M2: Covgate per-package timing

**Files:**
- `internal/services/covgate/covgate.go` — Add duration to results, TIME column, total time.
- `internal/services/covgate/covgate_test.go` — Update tests that assert output format.

**Steps:**
1. Add `"time"` import to `covgate.go`.
2. Add `duration time.Duration` field to `checkResult` struct.
3. In `checkPackage`, wrap the `r.measure()` call with `time.Now()` / `time.Since()`, store in result.
4. Update `printHeader` to add a `TIME` column.
5. Update the format strings in `checkPackage` (both pass and fail paths) to include the duration.
6. In `printResults`, compute and print total elapsed time from sum of all durations.
7. Update `TestPrintHeader` to check for the new `TIME` column.
8. Update `TestCheckPackage_*` tests — the `fakeMeasure` function returns instantly, so durations will be ~0. Assertions on output format should account for the TIME column.

### M3: --new-from-rev flag for golangci-lint

**Files:**
- `internal/services/lint/lint.go` — Add `NewFromRev` to `LintOpts`, modify `RunGolangci` signature.
- `internal/commands/lint.go` — Bind the new flag.
- `internal/services/lint/lint_test.go` — Add test for the new flag plumbing.

**Steps:**
1. Add `NewFromRev string` field to `LintOpts`.
2. Change `RunGolangci(out, errW io.Writer)` signature to `RunGolangci(out, errW io.Writer, newFromRev string)`.
3. In `RunGolangci`, build the args slice conditionally: `["go", "tool", "golangci-lint", "run"]` + `["--new-from-rev=<rev>"]` if non-empty.
4. Update the call site in `RunLint` to pass `opts.NewFromRev`.
5. In `internal/commands/lint.go`, add `fl.StringVar(&opts.NewFromRev, "new-from-rev", "", ...)` in `bindLintFlags`.
6. Add a test that verifies the flag is wired (command-level test or unit test).

## Validation

- All existing tests pass: `go test ./...` from the gotools root.
- New tests cover the timing output format and `--new-from-rev` flag wiring.
- Preflight must report clean before changes are published.

## Test Steps

1. Run `go test ./internal/services/lint/...` — all pass.
2. Run `go test ./internal/services/covgate/...` — all pass.
3. Run `go test ./...` — full suite passes.
4. Manual smoke: `go run ./cmd/miru lint --no-golangci --no-gofumpt` shows timing output.
