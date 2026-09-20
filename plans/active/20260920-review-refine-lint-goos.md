# Review and refine `miru lint --goos` (PR #44) until no findings remain

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` (`/home/ben/miru/workbench2/repos/gotools`) | read-write | Review the diff of PR mirurobotics/gotools#44, fix accepted findings, add tests, validate. |

This plan lives in `gotools/plans/` because all code changes land there. `gotools` is an independent git checkout: run every command from `/home/ben/miru/workbench2/repos/gotools`, never from the workbench root and never from a git worktree. The repo has no `AGENTS.md`/`CLAUDE.md` of its own; conventions are embedded below.

Branching: base branch `main`; working branch `feat/lint-vet-goos`, which already exists, is checked out, is pushed, and backs PR #44. Do not create a new branch or a new PR. The owner asked for the branch to sit on the latest `origin/main`: the orchestrator rebased it onto `ccbfa28` before review (Milestone 0), and Milestone 3 rebases again only if `main` has moved. A push that follows a rebase uses `git push --force-with-lease`; every other change is a new Conventional Commit on top of the branch, with no amends of pushed commits.

## Purpose / Big Picture

PR #44 adds an opt-in flag, `miru lint --goos=<comma-separated targets>`, that runs golangci-lint once more per extra target platform (for example `--goos=windows` lints the `_windows.go` files the Linux host run never sees). After this plan, that change has been reviewed for correctness, test coverage, and regressions; every accepted finding is fixed with a test; a final review pass returns "no findings"; and CI is green on the pushed head of PR #44. Observable result: `LINT_FIX=0 ./scripts/lint.sh` prints a `golangci-lint (windows)` row in its Timings table and exits 0, `./scripts/preflight.sh` prints `=== All checks passed ===`, and `gh pr checks 44` shows `lint`, `test`, and `surface-lint / surface-lint` passing (`[code]smith` reports skipped; ignore it).

## Progress

- [x] Milestone 0: rebase onto `origin/main` (`ccbfa28`), resolving `scripts/lint.sh` in both commits (done by the orchestrator; branch commits are now `60045d3` and `2c69761`).
- [x] Milestone 1: run the `review` skill over the diff; record findings (10 medium findings, R1–R10).
- [x] Milestone 2: run the `refine` skill until no findings remain; every behavior fix has a test (4 iterations; iteration 4 returned "no findings").
- [ ] Milestone 3: local preflight, push, `preflight` skill reports `CLEAN`, PR body resynced, plan moved to `plans/completed/`.

## Surprises & Discoveries

Milestone 1 review findings (2026-09-20; seven lenses, merged and deduplicated; security and performance returned "no findings"). All are severity medium; none is high.

- R1 (correctness, design) — `internal/services/lint/lint.go:105-119`. `runGolangciTargets` runs every non-blank entry as given: it removes neither duplicates nor the host platform. `--goos=windows,windows` runs the identical windows pass twice, reports `golangci-lint (windows)` twice in the failure summary, and prints two identical timing rows. An entry equal to the host GOOS repeats the run `runAnalyzers` already did, so a shared `--goos=linux,darwin,windows` prints every host issue twice on each host. The help text, the `Long` description, and the `LintOpts.GOOS` comment all say "extra" targets, which nothing enforces. Fix direction: an unexported `goosTargets(raw string) []string` that splits, trims, and drops blanks, `runtime.GOOS`, and already-seen entries in first-seen order; `runGolangciTargets` ranges over it; the help text says the host GOOS and duplicates are skipped; tests for the helper and for `runGolangciTargets` with `"notanos,notanos," + runtime.GOOS`.
- R2 (correctness, reliability, documentation) — `internal/services/lint/lint.go:298-309`, env from `internal/services/cmdutil/cmd.go:14`. `hostToolPath` claims to build the tool for the host but runs `go tool -n` with the parent's `GOOS`/`GOARCH`. Reproduced on linux/amd64: `GOOS=windows go tool -n golangci-lint` prints a `golangci-lint.exe` cache path that fails with `exec format error`; `GOARCH=arm64` prints a different (arm64) binary. With either variable exported, every `--goos` target step fails without linting and is reported as a lint failure, and both comments describing a "host binary" are false. Fix direction: in `hostToolPath` only, append `GOOS=runtime.GOOS` and `GOARCH=runtime.GOARCH` to `cmd.Env`; add a test that sets non-host `GOOS`/`GOARCH` with `t.Setenv` and asserts the resolved path equals the clean-environment path.
- R3 (test-coverage) — `internal/services/lint/lint_test.go:275-283`, `scripts/preflight.sh:17-21`. `TestRunGolangciGOOS_Success` needs a real golangci-lint run to succeed. golangci-lint takes a machine-wide lock (`$TMPDIR/golangci-lint.lock`), retries for 5s, then exits 3 with "parallel golangci-lint is running". `./scripts/preflight.sh` runs `lint.sh` (now two consecutive golangci-lint runs) and `covgate.sh` in parallel, so the test fails whenever the lock is held for more than 5s; reproduced with `flock /tmp/golangci-lint.lock sleep 9`. The test also depends on the package being lint-clean for windows. Fix direction: extract an unexported `runGolangciBin(out, errW, bin, newFromRev, goos)` and test success against a fake `#!/bin/sh` binary that echoes `$GOOS`, `$GOWORK`, and its arguments.
- R4 (test-coverage) — `lint.go:113,279`. Nothing asserts that `--new-from-rev` reaches the per-target runs; `golangciArgs("")` at the call site would pass every test. Fix direction: a fake-binary test with `newFromRev="main"` asserting `run --new-from-rev=main`.
- R5 (test-coverage) — `lint.go:95-99`. The `runLintSteps` → `runGolangciTargets` wiring is only tested for the skip case; deleting the block or the `failures` append leaves every test green. Fix direction: `runLintSteps` with `GOOS: "notanos"` asserts the target step is in `failures` and is the last timing.
- R6 (test-coverage) — `lint.go:273-277`. The resolve-failure branch of `RunGolangciGOOS` is untested; `notanos` only reaches the `cmd.Run` failure. Fix direction: `t.Setenv("PATH", <empty dir>)` so `go` cannot be found; assert a non-nil error and the failure line on `errW`.
- R7 (test-coverage) — `lint.go:280`. No test runs with `GOOS` already set in the parent; the appended target `GOOS` overriding an inherited one is unasserted. Fix direction: fake-binary test with `t.Setenv("GOOS", "plan9")` and target `windows`.
- R8 (test-coverage) — `internal/commands/lint.go:60-64`. Flag tests stop at lookup and default; binding `--goos` to the wrong field passes. Fix direction: `bindLintFlags` on a bare command, parse `--goos=windows,darwin --new-from-rev=main`, assert both fields.
- R9 (test-coverage) — `lint.go:106-118`. `runGolangciTargets` is only tested with one non-blank target; stopping after the first passes. Fix direction: table test covering `""`, blanks, and two distinct targets in order.
- R10 (test-coverage) — `lint.go:203-210`. No test asserts the 16-column minimum width of `printTimings`, which applies to every run without `--goos`. Fix direction: assert the exact padded `gofumpt` and `total` lines in `TestPrintTimings`.

Confirmed non-issues: `RunGolangciGOOS`'s export matches `RunGolangci`/`RunGofumpt`/`RunDeadcode`, and surface-lint (yamllint, shellcheck, actionlint) ignores Go symbols; `GOFLAGS` values still yield a single-line path from `go tool -n`; an empty resolved path fails closed at `cmd.Run`; no README or doc lists the sibling flags, so `--goos` is missing nowhere; golangci-lint's lock rules out parallelizing targets.

Refine iteration 2 findings (all medium): S1 no test observes the `tool -n <name>` arguments `hostToolPath` sends (the fake `go` ignored its arguments; dropping `-n` passed every test); S2 no test observes the arguments `RunGolangci` sends after the `golangciArgs` refactor; S3 the `LintOpts.GOOS` comment described other steps and duplicated the `goosTargets` skip wording; S4 the `RunGolangciGOOS` doc overstated coverage and said GOARCH comes "from the host" although an exported `GOARCH` passes through; S5 three comments restated their signatures. Iteration 3 finding (medium, documentation): U1 the reworded `RunGolangciGOOS` doc still said the run "also lints the files the host build excludes", which is false for host-only files and for other targets' files. Iteration 4: "no findings" across all lenses.

golangci-lint's machine-wide lock (`$TMPDIR/golangci-lint.lock`, exit 3 after 5s) makes any test that needs a successful real golangci-lint run flaky under `./scripts/preflight.sh`, which runs lint and covgate in parallel. Tests now use a fake `go` and a fake golangci-lint on a private `PATH`.

## Decision Log

- Decision: accept R1 (dedupe and host-GOOS skip). Rationale: duplicates and host entries repeat whole lint runs and double-report failures; "extra" was unenforced. Fixed in `8f6f36b` with `goosTargets`, `TestGoosTargets`, and `TestRunGolangciTargets/duplicates_and_host` (fails without the fix). 2026-09-20.
- Decision: accept R2 (pin `GOOS`/`GOARCH` in `hostToolPath` only). Rationale: reproduced a `.exe` path and exec format error under `GOOS=windows`; the target run still leaves GOARCH alone. Fixed in `87f0b00` with `TestHostToolPath_IgnoresInheritedTarget` (fail-before took 13s: it cross-compiles golangci-lint; pass state 0.3s). 2026-09-20.
- Decision: accept R3 and replace the real success run with fakes. Rationale: every real success check can fail under lock contention; `scripts/lint.sh --goos=windows` still proves the real binary runs in CI. `4f287f6` extracts `runGolangciBin` and adds `writeScript`, `fakeGolangci`, `fakeGoTool`. 2026-09-20.
- Decision: accept R4–R7 as test-only (`35a076f`), R9 with R1 (`8f6f36b`), R8 and R10 (`aa91250`). Rationale: each named regression was injected and caught only by the new test. 2026-09-20.
- Decision: accept S1 (`d6af37d`) and S2 (`17c7b13`). Rationale: dropping `-n`, or `golangciArgs("")` in `RunGolangci`, passed every test before; the fake `go` now resolves only `tool -n golangci-lint` and echoes other arguments. 2026-09-20.
- Decision: accept S3–S5 (`ffaa4d7`) and U1 (`8975b89`), comment-only. Rationale: no linter here requires doc comments; sibling trivial helpers carry none; the skip wording stays on `goosTargets` and the help text; "GOARCH is left to the environment" is accurate whether or not GOARCH is exported and leaves the settled behavior unchanged. 2026-09-20.
- Decision: no finding was skipped; none contradicted a settled owner decision. Security and performance lenses returned "no findings" in every iteration. 2026-09-20.
- Decision: the orchestrator applied each critique fix plan itself, one commit per fix, instead of a single fix subagent, so that fail-before/pass-after could be checked per commit. 2026-09-20.
- Decision: leave `.covgate` to Milestone 3's ratchet rather than bumping it per commit. 2026-09-20.

## Outcomes & Retrospective

Add entries as work proceeds.

## Context and Orientation

`miru` is the CLI built from `cmd/miru`. `miru lint` runs, in order: the custom linter (`internal/services/lint/linter`, only when `--paths` is set), gofumpt, then golangci-lint and deadcode (concurrently when both are enabled), then the new per-target golangci-lint runs. The diff under review is:

    git diff origin/main...feat/lint-vet-goos

Files in the diff:

- `internal/services/lint/lint.go` — `LintOpts.GOOS` (comma-separated string). `runLintSteps` calls `runGolangciTargets(opts)` after `runAnalyzers(opts)` unless `opts.NoGolangci`. `runGolangciTargets` splits `opts.GOOS` with `strings.Split`, trims, skips blanks, and calls `RunGolangciGOOS(out, errW, newFromRev, goos)` sequentially, recording a step named `golangci-lint (<goos>)` in the failure list and timings. `RunGolangciGOOS` calls `hostToolPath("golangci-lint")`, which runs `go tool -n golangci-lint` through `cmdutil.GoCommand` (in `internal/services/cmdutil/cmd.go`; it sets `cmd.Env = append(os.Environ(), "GOWORK=off")`) and returns the printed binary path; it then runs that binary with `cmd.Env = append(os.Environ(), "GOWORK=off", "GOOS="+goos)`. The host binary is resolved first because `GOOS=x go tool golangci-lint` cross-compiles the tool itself into a binary the host cannot run. `golangciArgs(newFromRev)` builds `run [--new-from-rev=<rev>]` and is shared with `RunGolangci`. `printTimings` widens its name column to the longest step name, minimum 16.
- `internal/commands/lint.go` — `--goos` string flag bound to `opts.GOOS` in `bindLintFlags`, plus the command's `Long` help text.
- `internal/services/lint/lint_test.go` — new tests `TestRunGolangciGOOS_Success`, `TestRunGolangciGOOS_UnsupportedGOOS`, `TestRunGolangciTargets_SkipsBlankEntries`, `TestRunLint_NoGolangciSkipsGOOSTargets`, `TestGolangciArgs`, `TestHostToolPath_UnknownTool`, `TestPrintTimings_WidensForLongStepNames`. Several shell out to the real golangci-lint from the package directory (about 1.3s total with a warm build cache).
- `internal/commands/commands_test.go` — `goos` added to the flag-type and flag-default tables.
- `internal/services/lint/.covgate` (63.6 → 69.3) and `internal/commands/.covgate` (62.3 → 63.0). A `.covgate` file holds a package's minimum coverage percentage. `scripts/covgate.sh` fails a package below its threshold (`FAIL`) and also fails one whose coverage exceeds its threshold by more than 0.5 points (`LOOSE`); `scripts/ratchet-covgates.sh` raises each `.covgate` to the measured coverage (ratchet up only; it never lowers a threshold).
- `scripts/lint.sh` — gotools lints itself with `--goos=windows`.

Settled owner decisions. These are not findings; tell every review and critique subagent to drop them: the flag is about target platforms, not `go vet`; only golangci-lint runs per target; deadcode stays host-only (cross-platform false positives); the custom linter and gofumpt already read every file regardless of build tags; GOARCH is inherited from the host.

House constraints for every fix:

- The custom linter enforces 88-column lines (tab counts as 4), function length 50, nesting depth 4, and at most 6 parameters.
- Code comments are concise and present-tense: they say what the code does now, with no history and no past-decision rationale.
- Match the surrounding idiom, e.g. `strings.Split` in a `range` loop, `_, _ = fmt.Fprintf(...)` for writer output, `//nolint:gosec,noctx // G204: trusted subprocess` on `exec.Command`.
- Tests go in the existing test files, use stdlib `testing` only (no assertion library), follow the existing `TestSubject_Scenario` naming, and put a `//nolint:exhaustruct // <reason>` comment above every partial `LintOpts` literal.
- Commits are Conventional Commits (`fix(lint): …`, `test(lint): …`, `docs(plans): …`), signed per the local git config.

State after Milestone 0 (2026-09-20): the branch is `ccbfa28` plus `60045d3` and `2c69761` (the rebased `613c984` and `734dde5`) plus a `docs(plans)` commit adding this plan, force-pushed with lease; local `main` (`dd4aaf0`) is stale, so always diff against `origin/main`; `origin/main` (`ccbfa28`, PR #43) also edits `scripts/lint.sh`, which conflicted in both commits during the rebase. `gh pr view 44` reports `isDraft: false`; this plan does not change the PR's draft state.

Areas the review must scrutinize, with authoring-time observations to confirm or refute (not pre-accepted findings):

- Error and exit-code handling in `RunGolangciGOOS` and `hostToolPath`: a resolve failure and a lint failure both mark the step failed; the `hostToolPath` error embeds stderr after a newline, which matches `RunGofumpt` in the same file; `go tool -n` runs once per target.
- Duplicate entries (`--goos=windows,windows` runs twice) and entries equal to the host GOOS (repeat the host run). A likely fix is a small parsing helper that trims, drops blanks, duplicates, and `runtime.GOOS`.
- Interaction with `--new-from-rev` (passed to every target), `--fix` (`DoFix` never reaches golangci-lint in either run), and `--no-golangci` (skips all targets).
- Sequential execution and output interleaving: targets run after `runAnalyzers` returns, so nothing interleaves. golangci-lint refuses concurrent instances unless `--allow-parallel-runners` is passed, so confirm this before accepting any "parallelize" suggestion.
- Environment precedence: `os/exec` keeps the last duplicate key, so the appended `GOOS`/`GOWORK` win in `RunGolangciGOOS`. `hostToolPath` inherits the parent's `GOOS`/`GOARCH`, so `GOOS=windows miru lint --goos=…` may resolve an unrunnable tool binary; a likely fix pins `GOOS`/`GOARCH` to `runtime.GOOS`/`runtime.GOARCH` in `hostToolPath` only.
- Whether the tests are hermetic and fast enough, given that they lint the real package directory.
- Help text and flag naming consistency with the other `miru lint` flags; the `printTimings` width logic (`len` counts bytes; step names are ASCII).

## Plan of Work

Milestone 0 is complete: the branch is rebased onto `origin/main`. The only conflict was `scripts/lint.sh`; the resolution is shown in Concrete Steps.

Milestone 1 invokes the `review` skill (via the Skill tool) with scope "branch diff `git diff origin/main...HEAD`", passing the settled decisions, house constraints, and scrutiny list above. Findings must be prioritized by severity with file/line evidence. Give the review subagents two overrides, which also apply to every Milestone 2 review and critique subagent. First, the `--goos` help text, the `Long` description, and flag naming consistency with the other `miru lint` flags are in scope despite the skill's naming exclusion. Second, gotools has no error-code package or assertion library, so drop any finding that asks for `assert.ErrCodeIs` or testify; tests assert with stdlib `testing` on returned errors and writer output. Copy the findings verbatim into Surprises & Discoveries, move this plan to `plans/active/`, and make no code changes.

Milestone 2 invokes the `refine` skill with `scope` = the branch diff, `base` = `origin/main`, `max_iterations` = 5, seeding its first critique step with the Milestone 1 findings and passing the same decisions and constraints to every subagent. Instruct the fix subagent that each behavior fix includes a test, placed in `internal/services/lint/lint_test.go` (service behavior) or `internal/commands/commands_test.go` (flag wiring). Prefer fast inputs: the bogus target `notanos` fails in about 0.3s, and `t.Setenv` covers environment cases. If the two likely fixes above are accepted, the expected tests are `TestRunGolangciTargets_DedupesTargets` (`GOOS: "notanos,notanos"` yields exactly one failure and one timing), `TestRunGolangciTargets_SkipsHostGOOS` (`GOOS: runtime.GOOS` yields none), and `TestHostToolPath_IgnoresParentGOOS` (`t.Setenv("GOOS", "windows")`; the returned path does not end in `.exe`). The fix subagent writes the fix and its test but runs neither; after each iteration the orchestrator verifies fail-before and pass-after for each new test, lints, and commits (see Concrete Steps). The fail-before run of `TestHostToolPath_IgnoresParentGOOS` cross-compiles golangci-lint for Windows once and can take minutes. The milestone ends when a review pass returns "no findings" or the critique accepts none. If the iteration cap is hit with findings remaining, record them in Surprises & Discoveries and run `refine` again; do not proceed to Milestone 3 with open accepted findings.

Milestone 3 validates locally, rebases onto `origin/main` only if it has moved past `ccbfa28`, pushes (plain `git push`, or `git push --force-with-lease` after a rebase), and invokes the `preflight` skill to watch CI. Override that skill's defaults explicitly: it must not rebase or force-push on its own; the PR already exists, so the push itself triggers CI; diff against `origin/main`; pass the settled decisions, house constraints, review overrides, and the Milestone 2 test rule to its refine step and its CI-fix subagent. If the skill commits any change under `internal/` or `scripts/`, re-run `./scripts/preflight.sh` and let CI go green on the new head before reporting.

Then resync the PR #44 body following the `pr` skill's update mode with these overrides: compute history with `git log --no-merges origin/main..HEAD`; keep the existing sections (Summary, per-step table, Why, Follow-ups, Validation) and edit only the Summary bullets and `.covgate` numbers the new commits change; if `gh pr edit 44 --body-file <f>` exits non-zero with a Projects-classic GraphQL error, apply the body with `gh api -X PATCH repos/mirurobotics/gotools/pulls/44 -F body=@<f>` and verify with `gh pr view 44 --json body -q .body`. Finally fill in Outcomes & Retrospective and move the plan to `plans/completed/`.

## Concrete Steps

All commands run from `/home/ben/miru/workbench2/repos/gotools`.

Milestone 0 — sync with main (done):

    git fetch origin
    git rebase origin/main                  # CONFLICT in scripts/lint.sh, once per commit

`scripts/lint.sh` resolves to:

    # bgctx and tempdir require core (mctx, test_dirs), which gotools does not
    # depend on, so its own context and temp-dir calls are legitimate.
    exec go run ./cmd/miru lint \
        --paths=internal \
        --exclude=nofmt,bgctx,tempdir \
        --goos=windows \
        $FIX

The first commit keeps its own `--vet-goos=windows` line in place of `--goos=windows`. After each resolution:

    git add scripts/lint.sh && GIT_EDITOR=true git rebase --continue

Then `go build ./...` and `go test ./internal/services/lint ./internal/commands` pass, and the branch is pushed with `git push --force-with-lease`.

Milestone 1 — review:

    git diff origin/main...HEAD             # the scope handed to the review skill
    mkdir -p plans/active
    mv plans/backlog/20260920-review-refine-lint-goos.md plans/active/
    git add plans/backlog plans/active
    git commit -m "docs(plans): record lint --goos review findings"

Milestone 2 — refine loop. After each iteration's fixes, for each new test (example for a fix in `lint.go`):

    git stash push -- internal/services/lint/lint.go
    go test ./internal/services/lint -run '<TestName>' -count=1   # expect FAIL
    git stash pop
    go test ./internal/services/lint -run '<TestName>' -count=1   # expect ok

Then:

    go test ./internal/services/lint ./internal/commands -count=1   # expect ok, ok
    LINT_FIX=0 ./scripts/lint.sh            # expect "Lint complete"; enforces the house limits
    git add <only the files the fix touched>
    git commit -m "fix(lint): <what the fix does>"

Use one commit per coherent fix (test included in the same commit), `test(lint): …` for test-only changes, and update Progress and the Decision Log (one entry per accepted or skipped finding, with the critique's reason) in a `docs(plans): …` commit at the end of the milestone.

Milestone 3 — validate and publish:

    ./scripts/preflight.sh                  # expect "=== All checks passed ==="

If covgate prints `LOOSE` for `internal/services/lint` or `internal/commands`:

    ./scripts/ratchet-covgates.sh
    git status --short -- '*.covgate'
    git checkout -- <every changed .covgate except internal/services/lint/.covgate and internal/commands/.covgate>
    ./scripts/preflight.sh
    git add internal/services/lint/.covgate internal/commands/.covgate
    git commit -m "test(lint): ratchet covgate thresholds"

A `FAIL` row means a fix lost coverage; the ratchet cannot help. Add tests until the package meets its committed threshold; never hand-lower a `.covgate`. Then:

    git fetch origin
    git merge-base --is-ancestor origin/main HEAD || git rebase origin/main   # only if main moved
    git push                                # --force-with-lease only if the line above rebased
    # invoke the preflight skill with the overrides from Plan of Work
    gh pr checks 44                         # expect lint, test, surface-lint / surface-lint: pass

After `CLEAN`, resync the PR body as described in Plan of Work, fill in Outcomes & Retrospective, then:

    git mv plans/active/20260920-review-refine-lint-goos.md plans/completed/
    git add plans/active plans/completed && git commit -m "docs(plans): complete lint --goos review and refine plan"
    git push
    gh pr checks 44 --watch                 # the final head must also be green

## Validation and Acceptance

- The final `refine` iteration's review step returns an explicit "no findings" (or the critique accepts none), recorded in Outcomes & Retrospective.
- Every behavior fix has a named test in `internal/services/lint/lint_test.go` or `internal/commands/commands_test.go` that fails with the fix reverted and passes with it; `go test ./internal/services/lint ./internal/commands -count=1` prints `ok` for both packages.
- `./scripts/preflight.sh`, run from the normal checkout (not a git worktree), prints `=== All checks passed ===`, with a `golangci-lint (windows)` timing row in the lint output and no `LOOSE` rows.
- `git diff origin/main...HEAD --stat -- '*.covgate'` lists only `internal/services/lint/.covgate` and `internal/commands/.covgate`.
- Preflight reports `CLEAN`, meaning CI (`lint`, `test`, `surface-lint / surface-lint`) is green on the pushed head SHA of PR #44 (`git rev-parse HEAD` equals `gh pr view 44 --json headRefOid -q .headRefOid`). Until then the PR must not leave draft (or be marked ready or merged) and the task must not be reported complete. A `CAPPED` result is a failure: record it and continue fixing.
- `git merge-base --is-ancestor origin/main HEAD` exits 0 on the final head, and `--force-with-lease` was used only for pushes that followed a rebase.

## Idempotence and Recovery

Review, refine, test, and preflight steps are repeatable. An in-progress `git rebase origin/main` can be abandoned with `git rebase --abort` and retried. A bad unpushed fix commit is undone with `git reset --hard HEAD~1`; a bad pushed commit is undone with `git revert <sha>` and a new push, never by amending or dropping it. `./scripts/ratchet-covgates.sh` may raise any package's `.covgate`; `git checkout -- <path>` restores any file that must not change. If `git push --force-with-lease` is rejected, the remote moved: fetch, inspect `origin/feat/lint-vet-goos`, and reconcile before pushing again.
