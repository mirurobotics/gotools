# Render excluded packages as SKIPPED rows in `miru covgate` output

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` | read-write | Extend covgate service to emit `SKIPPED` rows for `--exclude`d packages; update tests. |

This plan lives in `gotools/plans/backlog/` because all edits happen in `internal/services/covgate/` inside the `gotools` repo. No other repos are read or written.

## Purpose / Big Picture

`miru covgate --exclude <patterns>` already removes matched packages from the measurement set and prints a single summary line `Excluded N package(s) from coverage measurement`. Today the excluded packages never appear by name in the per-package table, so a user who mistypes a pattern only sees the count drop — not which packages are missing. This change keeps the summary line and additionally renders one `SKIPPED` row per excluded package in the same table, so users can confirm at a glance exactly which packages were skipped.

After this change, the user running:

    $ go tool miru covgate --packages ./... --exclude ./pkg/b

against a module that enumerates `pkg/a`, `pkg/b`, `pkg/c` will see:

    Checking per-package coverage (default minimum: 80.0%)...

    Excluded 1 package(s) from coverage measurement
    STATUS   COVERAGE  REQUIRED      TIME  PACKAGE
    ------   --------  --------  --------  -------
    PASS        90.0%     80.0%      0.5s  pkg/a
    PASS        90.0%     80.0%      0.5s  pkg/c
    SKIPPED       ---       ---       ---  pkg/b

    Total time: 1.0s
    All packages meet minimum coverage requirement

Key user-visible properties:

- Each excluded package gets one `SKIPPED` row.
- `COVERAGE`, `REQUIRED`, and `TIME` columns are `---` (literal dashes; not `0.0s`).
- `SKIPPED` rows appear grouped at the end of the table, after all measured rows.
- `SKIPPED` packages do not count toward the failure tally — if every measured package passes, the run succeeds even when packages are skipped.
- When `--exclude` is unset or empty, output is byte-identical to today.

## Progress

- [ ] Milestone 1 — Service: thread excluded list through the run and emit `SKIPPED` rows; ignore skipped packages in the failure tally.
- [ ] Milestone 2 — Tests: update `TestRun_Exclude` cases to assert `SKIPPED` rows and tally behavior; add an all-excluded case; verify summary line still prints.
- [ ] Milestone 3 — Validation: run `./scripts/preflight.sh`; iterate until it reports `=== All checks passed ===`.

Use timestamps as steps are completed. Split partially completed work into "done" and "remaining" as needed.

## Surprises & Discoveries

(Add entries as work proceeds.)

## Decision Log

- Decision: Place `SKIPPED` rows at the **end** of the per-package table, after all measured rows.
  Rationale: Skipped packages have no coverage data and would visually intrude on the scannable PASS/FAIL/LOOSE block if interleaved with measured rows. Grouping them at the end keeps the measured results dense and the skipped block easy to scan as a separate "what got dropped" section. The summary line at the top already signals upfront *how many* were skipped; the table tail answers *which ones*.
  Date/Author: 2026-05-15, planning subagent.

- Decision: Return the excluded list from `applyExclude` alongside the kept list, rather than marking-and-skipping inside iteration.
  Rationale: Keeps the existing `runPackages` concurrency loop untouched (no synthetic "skip" branch competing with `measure`). Building a separate slice of pre-formatted `SKIPPED` rows and appending them after the measured results in `printResults` is the smallest, most local change. The alternative (a `skipped bool` on `checkResult` plus a branch in `checkPackage`) would mean either feeding fake packages into the goroutine pool or adding a fork in `runPackages`, both of which expand the diff for no observable benefit.
  Date/Author: 2026-05-15, planning subagent.

- Decision: Render the SKIPPED row's coverage, required, and time columns as the literal string `---`.
  Rationale: `0.0s` for time is actively misleading (no test ran), and a blank column would mis-align under the existing fixed-width format. `---` is already used elsewhere in this file for the FAIL-with-test-error case (see `covgate.go` line 254) so the convention is established.
  Date/Author: 2026-05-15, planning subagent.

- Decision: Widen the `STATUS` column from 6 to 7 characters so `SKIPPED` fits without breaking alignment.
  Rationale: `SKIPPED` is 7 characters; the current format width is 6 (`%-6s`). Keeping the width at 6 would force `SKIPPE` truncation or unaligned output. Update the header, separator, and every status `Fprintf` together so all rows stay aligned.
  Date/Author: 2026-05-15, planning subagent.

## Outcomes & Retrospective

(Summarize at completion or major milestones.)

## Context and Orientation

Covgate is a Go test-coverage gate exposed as the `covgate` subcommand of the `miru` CLI (built from `cmd/miru/main.go`). All edits in this plan happen in two files inside `/home/ben/miru/workbench2/repos/gotools/`:

- `internal/services/covgate/covgate.go` — service logic.
- `internal/services/covgate/covgate_test.go` — tests.

Relevant types and functions in `covgate.go`:

- `Opts` (lines 15–29) carries options including `Exclude string` (added previously in `20260515-covgate-exclude-flag.md`).
- `runner` (lines 31–36) injects `goModule`, `goListPackages`, `measure` for testability.
- `(*runner).run` (lines 70–117) is the orchestration entry: resolves packages, calls `applyExclude`, prints the header, calls `runPackages`, then `printResults`.
- `applyExclude` (lines 124–161) currently:
  - Returns the kept slice and `nil` for empty/whitespace `exclude`.
  - For each comma-separated entry, calls `r.goListPackages(entry)` and unions the results into a `map[string]struct{}`.
  - Walks `pkgs` keeping the non-excluded ones in original order.
  - Prints `Excluded N package(s) from coverage measurement` to `w` when `N > 0`.
  - Returns the kept slice and any error.
- `runPackages` (lines 163–181) walks `pkgs` concurrently bounded by `parallelism`, returning a `[]checkResult` indexed by the original slice order.
- `printResults` (lines 183–206) prints each `checkResult.output`, sets `hasFailures` from `!res.passed`, prints `Total time`, and emits the final pass/fail line.
- `printHeader` (lines 208–217) uses the format string `"%-6s  %8s  %8s  %8s  %s\n"` with columns `STATUS  COVERAGE  REQUIRED  TIME  PACKAGE` and a matching dashed separator.
- `checkResult` (lines 219–224) is `{output string; passed bool; duration time.Duration}`.
- `checkPackage` (lines 236–292) is the per-package measure + formatter. The FAIL-with-test-error branch (lines 252–262) already uses `"---"` for coverage and required when no measurement could be taken; SKIPPED reuses that convention plus `"---"` for time.

Test fixtures in `covgate_test.go`:

- `TestRun_Exclude` (lines 167–281) is the existing table-driven test for `--exclude`. The five subtests are `NoExclude`, `Subset`, `NoOpPattern`, `MultiplePatternsWithWhitespace`, and `AllPackages`. Each case asserts `wantContains` (substrings that must appear) and `wantNotContain` (substrings that must NOT appear).
- `TestRun_Exclude_GoListError` (lines 283–317) covers the error path when an exclude pattern fails to resolve.

Validation:

- `scripts/preflight.sh` runs `lint.sh`, `covgate.sh`, and `lint-surface.sh` in parallel; the final success line is `=== All checks passed ===`. This is the canonical "preflight" command.

Glossary:

- **SKIPPED row**: a single line in the per-package table representing a package that was excluded from measurement by `--exclude`. Format: `SKIPPED       ---       ---       ---  <relative package path>`.
- **Failure tally**: the `hasFailures` boolean in `printResults` (line 186). Set when any `checkResult.passed` is false. SKIPPED rows have `passed: true` so they do not flip it.
- **Preflight clean**: `./scripts/preflight.sh` exits 0 and prints `=== All checks passed ===`.
- **Relative package path**: the import path with the module prefix stripped, e.g., `pkg/b` for `example.com/mod/pkg/b`. Computed by `gocover.RelPkg(pkg, module)` in the existing code.

## Plan of Work

### Milestone 1 — Service changes (`internal/services/covgate/covgate.go`)

1. Change `applyExclude` to return both the kept slice and the excluded slice:

   - New signature: `func (r *runner) applyExclude(pkgs []string, exclude string, w io.Writer) (kept []string, excluded []string, err error)`.
   - When `exclude` is empty/whitespace, return `pkgs, nil, nil`.
   - Otherwise, build the same `excluded` map as today, then in one pass over `pkgs` produce both the `kept` slice (paths not in the map, preserving original order) and the `excluded` slice (paths that are in the map, also in original `pkgs` order so output stays deterministic).
   - Keep the existing `Excluded N package(s) from coverage measurement` notice; it fires when `len(excluded) > 0`.
   - Update the doc comment to reflect the new return shape.

2. Update the call site in `(*runner).run` (line 97):

       pkgs, excluded, err := r.applyExclude(pkgs, opts.Exclude, w)

   Pass `excluded` through to `printResults`.

3. Widen the `STATUS` column from 6 to 7 characters. Edit `printHeader` (lines 208–217) so the format string becomes:

       "%-7s  %8s  %8s  %8s  %s\n"

   and update the separator literal `"------"` to `"-------"` (seven dashes). Apply the same `%-7s` width to every status `Fprintf` inside `checkPackage` (the FAIL-tests-failed branch at line 254, the FAIL-below-threshold branch at line 266, the LOOSE branch at line 277, and the PASS branch at line 288). After this edit, every row's first field is left-padded to 7 columns so `SKIPPED` and the existing statuses all align.

4. Add a small helper that produces a SKIPPED row for a given package import path. Inside `covgate.go`, below `checkPackage`, add:

       // skippedRow formats a single SKIPPED row using the same
       // column widths as PASS/FAIL/LOOSE. relPkg is the
       // module-relative import path. Used for packages removed
       // by --exclude so the user can see which ones were skipped.
       func skippedRow(relPkg string) string {
           return fmt.Sprintf(
               "%-7s  %8s  %8s  %8s  %s\n",
               "SKIPPED", "---", "---", "---", relPkg,
           )
       }

5. Update `printResults` to accept the module name and the excluded slice, and to emit SKIPPED rows after the measured rows:

   - New signature: `func (r *runner) printResults(w io.Writer, results []checkResult, excluded []string, module string, totalTime time.Duration) error`.
   - Existing behavior: print each `res.output`, set `hasFailures` if any `!res.passed`.
   - New behavior: after the measured-results loop, iterate `excluded` in order and print `skippedRow(gocover.RelPkg(pkg, module))` for each. Do **not** touch `hasFailures`.
   - The remaining tail (`Total time:`, pass/fail line) is unchanged.

6. Update the caller in `(*runner).run` (line 116):

       return r.printResults(w, results, excluded, module, wallTime)

7. The `module` value is already in scope at the call site (resolved earlier in `run` from `r.goModule()`), so no additional plumbing is required.

After these edits, the failure tally only considers `results` (measured packages), and skipped packages affect only the visible table.

### Milestone 2 — Test changes (`internal/services/covgate/covgate_test.go`)

Goal: assert `SKIPPED` rows now appear and that the failure tally ignores them.

1. Update `TestRun_Exclude` cases (lines 167–281). For each subtest, expand `wantContains` / `wantNotContain` so the assertions match the new behavior:

   - **NoExclude**: unchanged. Still expects `pkg/a`, `pkg/b`, `pkg/c` to appear; still expects no `Excluded` notice; still expects no `SKIPPED` row.
   - **Subset** (`exclude: "./pkg/b"`):
     - `wantContains` adds: `"SKIPPED"`, the substring `"pkg/b"` (it now reappears as a SKIPPED row).
     - `wantContains` keeps: `"pkg/a"`, `"pkg/c"`, `"Excluded 1 package(s) from coverage measurement"`.
     - `wantNotContain` removes `"pkg/b"` (it is now expected to appear) and **adds** `"FAIL"` (no failures: SKIPPED must not count toward the tally; the `All packages meet minimum coverage requirement` line is the success indicator and is already an implicit assertion via the absence of an error).
   - **NoOpPattern**: unchanged. Pattern matches no packages, so no SKIPPED row and no `Excluded` notice.
   - **MultiplePatternsWithWhitespace** (`exclude: ", ./pkg/a, , ./pkg/c,"`):
     - `wantContains` adds: `"SKIPPED"`, `"pkg/a"`, `"pkg/c"` (both reappear as SKIPPED rows).
     - `wantContains` keeps: `"pkg/b"`, `"Excluded 2 package(s) from coverage measurement"`.
     - `wantNotContain` removes `"pkg/a"` and `"pkg/c"` (now expected to appear).
   - **AllPackages** (`exclude: "./..."`, all three packages excluded):
     - `wantContains` adds: `"SKIPPED"`, `"pkg/a"`, `"pkg/b"`, `"pkg/c"` (all three appear as SKIPPED rows).
     - `wantContains` keeps: `"Excluded 3 package(s) from coverage measurement"`, `"All packages meet minimum coverage requirement"`, `"Total time:"`.
     - `wantNotContain` removes `"pkg/a"`, `"pkg/b"`, `"pkg/c"` (now expected to appear) and **adds** `"FAIL"` (run must succeed — when *all* packages are excluded the tally is empty and pass/fail logic must report success).

2. Add a new subtest **OrderingAndCount** to `TestRun_Exclude`'s cases table that asserts the SKIPPED rows appear *after* the measured rows. Use `strings.Index` for ordering:

       {
           name:    "OrderingAndCount",
           exclude: "./pkg/b",
           lookup:  map[string][]string{"./...": allThree, "./pkg/b": {pkgB}},
           // assertions handled below in the test loop, see note
       },

   Because the existing table-driven helper does substring `Contains` only, add a small `extra func(t *testing.T, out string)` hook on the case struct (default `nil`) and call it from the loop after the contains/not-contains assertions. For this case, the hook asserts:

   - `strings.Index(out, "pkg/a") < strings.Index(out, "SKIPPED")` — measured pkg/a row precedes the SKIPPED block.
   - `strings.Index(out, "pkg/c") < strings.Index(out, "SKIPPED")` — measured pkg/c row precedes the SKIPPED block.
   - The substring `"SKIPPED"` appears exactly once (so we are not accidentally double-rendering).
   - The line containing `"pkg/b"` also contains `"SKIPPED"` and the three `"---"` placeholders, confirming the row format. The simplest implementation scans `strings.Split(out, "\n")` for the line containing `pkg/b` and asserts it matches the expected pattern (contains `SKIPPED`, contains `---`).

3. Keep `TestRun_Exclude_GoListError` (lines 283–317) unchanged — the error path is independent of the row-rendering change.

4. Sanity-check that no other tests in the file rely on the old `STATUS` column width of 6. Grep mentally: existing tests use `strings.Contains` against substrings like `"PASS"`, `"FAIL"`, `"LOOSE"`, `"0.0%"`, `"80.0%"`, `"TIME"`, `"pkg/a"` — none of those care about column padding. `TestPrintHeader` (lines 17–31) checks `strings.Contains(out, "------")`; six dashes still appear as a substring of seven dashes, so the assertion remains true after the width change. No edits required there.

### Milestone 3 — Validation

Run preflight from the gotools repo root and iterate until clean. See "Concrete Steps" for exact commands.

## Concrete Steps

All commands run from `/home/ben/miru/workbench2/repos/gotools/`. The current branch is `feat/covgate-skipped-rows` (already created upstream by the orchestrator). Do not switch branches.

### Step 0 — Sanity baseline

    git status
    go build ./...
    go test ./internal/services/covgate/...

Expected: clean working tree (this plan file may be untracked/added), build exits 0 silently, tests pass.

### Step 1 — Implement Milestone 1 (service edits)

Edit `internal/services/covgate/covgate.go` per "Plan of Work — Milestone 1".

Then:

    go build ./internal/services/covgate/...

Expected: exits 0 with no output.

Run the existing tests — many will now fail because their expectations have not been updated yet:

    go test ./internal/services/covgate/...

Expected: failures in `TestRun_Exclude/Subset`, `TestRun_Exclude/MultiplePatternsWithWhitespace`, and `TestRun_Exclude/AllPackages` because their `wantNotContain` lists still reject `pkg/a`/`pkg/b`/`pkg/c` substrings which now reappear as SKIPPED rows. This is expected — the test updates land in Step 2.

Commit:

    git add internal/services/covgate/covgate.go
    git commit -m "feat(covgate): render --exclude packages as SKIPPED rows"

### Step 2 — Implement Milestone 2 (test updates)

Edit `internal/services/covgate/covgate_test.go` per "Plan of Work — Milestone 2".

    go test ./internal/services/covgate/...

Expected: `ok  	github.com/mirurobotics/gotools/internal/services/covgate` with no failures. All five updated cases plus the new `OrderingAndCount` case pass.

Commit:

    git add internal/services/covgate/covgate_test.go
    git commit -m "test(covgate): assert SKIPPED rows render for --exclude and don't fail the run"

### Step 3 — Milestone 3 preflight

    ./scripts/preflight.sh

Expected final line: `=== All checks passed ===` with exit 0.

If preflight is not clean, read the offending output (lint, covgate, or surface lint), apply fixes in place, and rerun. Common items: gofumpt format, golangci-lint findings on the new helper, surface-lint reaction to any newly exported symbol (this plan adds none — `skippedRow` is lowercase and `printResults`'s new args do not change its export status because it is already lowercase).

If fixes were applied, commit them:

    git add -A
    git commit -m "chore(covgate): preflight fixes"

Skip this commit if preflight was clean on the first run.

## Validation and Acceptance

Acceptance is observable behavior plus a clean preflight, in this order. **Changes MUST NOT be published (no push, no PR) until preflight reports clean — preflight clean is the gate.**

1. **Service tests pass.** From `gotools/`:

       go test ./internal/services/covgate/...

   Expected: `ok` line. Specifically:
   - `TestRun_Exclude/NoExclude` — passes unchanged.
   - `TestRun_Exclude/Subset` — passes with the updated assertions (`SKIPPED` present, `pkg/b` reappears as a SKIPPED row, no `FAIL`).
   - `TestRun_Exclude/NoOpPattern` — passes unchanged (no SKIPPED row when pattern matches nothing).
   - `TestRun_Exclude/MultiplePatternsWithWhitespace` — passes (`pkg/a` and `pkg/c` reappear as SKIPPED rows).
   - `TestRun_Exclude/AllPackages` — passes (all three packages appear as SKIPPED rows; run succeeds; tally not flipped).
   - `TestRun_Exclude/OrderingAndCount` — new subtest passes: SKIPPED block is at the end, exactly one `SKIPPED` substring, row contains the `---` placeholders.
   - `TestRun_Exclude_GoListError` — passes unchanged.
   - All other existing tests in the file still pass; the STATUS-column widening from 6 to 7 must not break any substring-based assertion.

   Before this change the new subtest does not exist; after, it passes. The updated subtests fail under the old assertions (because `SKIPPED` rows reintroduce excluded package names) and pass under the new ones.

2. **Build is clean.**

       go build ./...

   Expected: exit 0, no output.

3. **Preflight is clean — mandatory gate.** From `gotools/`:

       ./scripts/preflight.sh

   Expected final line: `=== All checks passed ===` with exit 0. **No changes are published, no PR is opened, and no branch is pushed until this command reports clean.** If it fails on any of lint, covgate, or surface lint, fix the underlying issue and rerun. Do not bypass preflight.

4. **End-to-end smoke (manual, optional).** Build the `miru` binary and run against the gotools repo itself:

       go build -o /tmp/miru ./cmd/miru
       /tmp/miru covgate --packages ./... --exclude ./internal/services/gocover/...

   Expected: the output contains `Excluded N package(s) from coverage measurement` for some `N >= 1`, and the table contains one `SKIPPED` row per excluded gocover package, each with `---  ---  ---` in the coverage/required/time columns. The overall run succeeds (exit 0). Without `--exclude`, the gocover packages appear as PASS/FAIL rows as before.

## Idempotence and Recovery

- Service edits (Milestone 1): purely additive in semantics. If a partial edit leaves the file in a broken state, `git restore internal/services/covgate/covgate.go` recovers the pre-edit version; re-apply from the Plan of Work.
- Test edits (Milestone 2): if a partial edit leaves stale assertions, `git restore internal/services/covgate/covgate_test.go` recovers the previous state; re-apply.
- Preflight (Milestone 3): idempotent — re-running `./scripts/preflight.sh` repeatedly is safe and always reflects the current working tree.
- Commits: one commit per milestone per Concrete Steps. If a commit needs to be redone, prefer a new commit over `git commit --amend` so milestone boundaries stay visible in PR review. Do **not** use `git reset --hard` because it discards work in unrelated files; prefer `git restore <path>` on specific files.
- Rollback path: revert milestone commits in reverse order with `git revert <sha>`. The change is purely visual (a new row type and a wider STATUS column) with no semantics change for callers that do not parse covgate output; reverting is low-risk.
