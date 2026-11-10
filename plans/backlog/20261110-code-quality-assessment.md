# gotools code quality assessment

## Goal

Find the main code quality gaps in `gotools` and rank them, so follow-up
work can be planned as small PRs. This document changes no code. Each item
lists the problem, where it is, a suggested fix, and a rough effort.

Effort scale: **S** is under half a day, **M** is one to two days, **L** is
more than two days.

Priority scale:

- **P0**: wrong results or checks that silently pass. Fix first.
- **P1**: maintainability or reliability gaps that will cost time soon.
- **P2**: polish and nice-to-haves.

## How this was assessed

Reviewed at `main` @ `7c17a25`:

- Read every non-test Go file under `cmd/` and `internal/`, the scripts, the
  CI workflows, and the lint config.
- Ran `go build`, `go vet`, and `go test -race ./...` with Go 1.25.3.
- Ran `LINT_FIX=0 ./scripts/lint.sh`. It passes, with one warning (see P1-3).
- Tried `govulncheck`. It could not reach `vuln.go.dev` from the review
  sandbox, so vulnerability status is not covered here.

## What is already good

This is a tidy, well-tested repo, and the items below are on top of a
strong base:

- Clear layout: `cmd/miru` is a thin entry point, `internal/commands` holds
  Cobra wiring, and `internal/services/*` holds the logic.
- Per-package coverage gates (`.covgate`) with a tightness check. Most
  packages sit at 90 to 100 percent.
- Strict linting: golangci-lint with `exhaustruct`, `gosec`, `gocritic`,
  `unparam` and more, plus gofumpt, deadcode, an extra Windows pass, and the
  repo's own custom rules.
- CI actions pinned to commit SHAs, Dependabot for Go modules and Actions,
  nightly CodeQL, and a required AI review.
- Past design work is written up under `plans/completed/`.

## Summary

| ID | Priority | Item | Effort |
|----|----------|------|--------|
| P0-1 | P0 | `covgate`/`covratchet` resolve `.covgate` files from the current directory, not the module root | S |
| P0-2 | P0 | A malformed `.covgate` file silently falls back to the default threshold | S |
| P0-3 | P0 | Custom linter turns parse and write errors into warnings and still exits 0 | S |
| P0-4 | P0 | `miru tag latest` ranks a pre-release above its final release | S |
| P0-5 | P0 | Two covratchet tests fail when run as root | S |
| P1-1 | P1 | No README, LICENSE, CHANGELOG or `--version` on a public, versioned tool | S |
| P1-2 | P1 | Unformatted files stop `miru lint` before golangci-lint and deadcode run | S |
| P1-3 | P1 | Custom rules share the `//nolint:` namespace with golangci-lint | M |
| P1-4 | P1 | Each rule handles suppression and path exemptions itself, with substring matching | M |
| P1-5 | P1 | Rule metadata lives in four places in `linter/run.go` | M |
| P1-6 | P1 | Subprocess calls have no shared seam, so some packages are barely tested | M |
| P1-7 | P1 | covgate and covratchet duplicate logic and have drifted apart | M |
| P1-8 | P1 | `covratchet` drops test output when a package fails | S |
| P1-9 | P1 | `exhaustruct` needs 31 `//nolint` lines in non-test code | S |
| P2-1 | P2 | No `context.Context`, signal handling or timeouts for subprocesses | M |
| P2-2 | P2 | `miru lint` writes files by default | S |
| P2-3 | P2 | CI gaps: no race detector, no govulncheck, Linux only | S |
| P2-4 | P2 | Command tests check metadata, not behavior | M |
| P2-5 | P2 | Small correctness and clarity nits | S |
| P2-6 | P2 | Go toolchain pinned at 1.25.3 | S |
| P2-7 | P2 | `install-lint-tools.sh` installs `actionlint@latest` | S |

Suggested order: all P0 items in one or two PRs (they are small and
independent), then P1-1, P1-2 and P1-8 as quick wins, then the linter
refactor (P1-3, P1-4, P1-5) as one planned piece of work.

---

## P0: wrong results or silent passes

### P0-1: `.covgate` lookup depends on the current directory

**Where:** `internal/services/covgate/covgate.go:401`,
`internal/services/covratchet/covratchet.go:209`,
`internal/services/gocover/gocover.go:63`.

**Problem:** Both commands build the threshold path as `"./" + relPkg`,
which is relative to the current directory, not the module root. Run from
any subdirectory, every `.covgate` file is "missing". covgate then applies
the default threshold, and covratchet would write new `.covgate` files in
the wrong place.

Reproduced on this repo:

    $ cd internal && go run ../cmd/miru covgate \
        --packages=./services/coverage/... --src-prefix=internal -p 1
    FAIL   17.9%   80.0%   internal/services/coverage   # .covgate (17.9) ignored

    $ go run ./cmd/miru covgate \
        --packages=./internal/services/coverage/... --src-prefix=internal -p 1
    PASS   17.9%   17.9%   internal/services/coverage

The same assumption forces every test that touches `.covgate` files to call
`t.Chdir`, which is why those tests cannot run in parallel.

**Fix:** Resolve the module root once (`go list -m -f '{{.Dir}}'` or
`go env GOMOD`) and join paths to it. Pass the root through the runner so
tests can point at a temp dir without `t.Chdir`.

**Effort:** S

### P0-2: Malformed `.covgate` silently uses the default threshold

**Where:** `internal/services/gocover/gocover.go:63-76` (`LookupThreshold`).

**Problem:** If the file exists but does not parse (a typo, a stray `%`, an
empty file), `LookupThreshold` returns `(0, false)`, the same as "no file".
covgate then uses `--default-threshold` and skips the tightness check, with
no warning. covratchet treats the same file as a hard failure, so the two
commands disagree. Values outside 0 to 100 are also accepted.

**Fix:** Return an error for "exists but invalid" and fail that package
with a clear message. Validate the range. Share one reader between covgate
and covratchet (see P1-7).

**Effort:** S

### P0-3: Custom linter errors do not fail the run

**Where:** `internal/services/lint/linter/run.go:204-210`.

**Problem:** `linter.Run` prints `warning: <file>: <err>` and continues for
any per-file error: a stat or read error, a parse error, and in fix mode a
failed `os.WriteFile`. The run returns no error and the file adds no
diagnostics, so `miru check` exits 0 for a file it could not lint or could
not fix.

**Fix:** Collect per-file errors and return them joined (`errors.Join`)
after processing every file, so one bad file does not hide others but the
run still fails.

**Effort:** S

### P0-4: `miru tag latest` picks a pre-release over the final release

**Where:** `internal/services/tag/tag.go:32` (`VersionLess`) and `:146`
(`latest`).

**Problem:** Sorting copies `sort -V`, where a longer string sorts later.
That makes `v1.0.0-beta` newer than `v1.0.0`, and a test locks it in
(`tag_test.go:22`, "prerelease shorter"). Semantic versioning says the
opposite. With tags `v1.0.0-rc.1` and `v1.0.0`, `latest` returns
`v1.0.0-rc.1`. The `latest` pattern also has no end anchor, so
`v1.2.3anything` matches.

**Fix:** Use `golang.org/x/mod/semver` (already in the module graph) for
comparison, anchor the pattern to a semver suffix, and update the test.
First check how consumers call `miru tag latest`, in case one relies on the
current order.

**Effort:** S

### P0-5: Two covratchet tests fail when run as root

**Where:** `internal/services/covratchet/covratchet_test.go:142`
(`TestRatchetPackage_New_WriteError`) and `TestRatchetPackage_Up_WriteError`.

**Problem:** The tests make a directory read-only with `chmod 0555` to force
a write error. Root ignores directory permissions, so the write succeeds and
both tests fail. This affects containers and some devcontainer setups. CI
passes only because the runner is not root.

**Fix:** Inject the file writer into the runner (like `measure`), and have
the test supply one that returns an error. That removes the filesystem trick.

**Effort:** S

---

## P1: maintainability and reliability

### P1-1: Missing repo docs and version info

**Problem:** The repo is public and tagged (23 tags up to `v0.2.6`), but
has:

- No `README.md`: nothing explains what `miru` does, how other repos install
  or pin it, or what the `.covgate` file format is.
- No `LICENSE`. GitHub reports `license: null`, so others have no clear
  right to use the public code.
- No `CHANGELOG.md` or GitHub releases. Consumers bumping versions (for
  example `core/plans/completed/20260408-update-gotools-v016.md`) have to
  read commits.
- No `miru --version`, so a consumer cannot tell which version ran in CI.
- No `.gitignore`. `miru coverage` writes `coverage.out` to the current
  directory.
- No `CLAUDE.md` or `AGENTS.md`, which other Miru repos use to brief agents.

**Fix:** Add a README (install, commands, `.covgate` format, rule list), a
LICENSE (owner's choice), a CHANGELOG, and a `.gitignore`. Set
`cmd.Version` from `runtime/debug.ReadBuildInfo()`, which works with
`go run …@vX`. Optionally add a tag-triggered release workflow that creates
release notes.

**Effort:** S

### P1-2: Unformatted files stop the lint run early

**Where:** `internal/services/lint/lint.go:80-88`.

**Problem:** In check mode, a gofumpt failure returns as a fatal error.
golangci-lint, deadcode and the extra GOOS passes then never run, so a CI
run shows one problem at a time. Other steps add to `failures` and keep
going.

**Fix:** Treat "files need formatting" as a normal step failure. Keep fatal
returns for cases where gofumpt itself could not run.

**Effort:** S

### P1-3: Custom rules collide with golangci-lint's `//nolint`

**Where:** `nolint:` checks in `funclen`, `bgctx`, `tempdir`, `nofmt`,
`mutableglobal`.

**Problem:** Custom rules read `//nolint:<rule>`, which golangci-lint also
owns. Every lint run prints:

    level=warning msg="[runner/nolint_filter] Found unknown linters in //nolint directives: funclen"

That stops the repo from enabling `nolintlint` to catch stale suppressions,
and a name clash with a future golangci linter would silently suppress the
wrong thing.

**Fix (choose one):**

1. Short term: use a separate directive, such as `//miru:ignore funclen`,
   and accept the old form during a deprecation period.
2. Longer term: port the rules to `golang.org/x/tools/go/analysis` and ship
   them as a golangci-lint module plugin. golangci-lint then owns `//nolint`,
   output format, caching and editor integration, and the custom walker and
   fixer could mostly go away. This is a larger project (**L**) and deserves
   its own plan.

**Effort:** M for option 1, L for option 2.

### P1-4: Suppression and path exemptions are reimplemented per rule

**Where:** `internal/services/lint/linter/analysis/exempt.go:29-37`, and
`hasNolint` copies in five rule packages.

**Problem:**

- Five rules carry their own `hasNolint` helper, with slightly different
  rules about which line the comment must be on. Other rules (`linelen`,
  `paramcount`, `nestdepth` and more) cannot be suppressed at all.
- Exemptions use substring matching on the path. `IsCmdSource` matches any
  path containing `cmd/` (so `internal/subcmd/x.go`), `IsTestHelper` matches
  `unittests/`, and the paths would not match on Windows separators.
- `IsLintSource` exempts any path containing `services/lint/`. `paramcount`
  and `nofmt` use it, so the linter skips its own source. In a consumer repo
  it would skip any package that happens to have that path.
- `FindGoFiles` always skips any directory named `codegen`
  (`run.go:365`), which is a Miru-specific rule hidden in a generic tool.

**Fix:** Move suppression into the runner: collect suppression comments
once per file and drop matching diagnostics centrally, so every rule
supports it the same way. Match exemptions on path segments
(`filepath.ToSlash` then split), and make skip directories a flag or config
value with today's values as the default. Drop the `IsLintSource` self
exemption and fix or explicitly suppress the code it hides.

**Effort:** M

### P1-5: Rule metadata is spread across four places

**Where:** `internal/services/lint/linter/run.go`: the `Rule` constants
(`:37`), `AllRules()` (`:59`), `ruleCheckers()` (`:115`, which needs
`//nolint:funclen`), `applyCandidateFixers()`, and the special case for
`imports`.

**Problem:** A new rule must be added in up to four places. Missing one
fails quietly; for example a rule left out of `AllRules()` cannot be
excluded with `--exclude`. There is also no way to list rules with
descriptions from the CLI.

**Fix:** One registry: a slice of `{Name, Doc, Fixable, Check, FindCandidates}`
built in one place. Derive `AllRules`, validation, check dispatch and fix
dispatch from it. Add `miru check --list-rules` and generate the README rule
table from it.

**Effort:** M (fits well with P1-4)

### P1-6: No shared seam for running subprocesses

**Where:** `cmdutil.GoCommand`, `gocover.ExecOutput` (a second copy of the
same `GOWORK=off` logic at `gocover.go:202`), direct `exec.Command` in
`lint.go` and `tag.go`.

**Problem:** Some packages inject function fields for tests (`covgate`,
`deps`, `tag`); others call `exec` directly (`coverage`, `gotest`, most of
`lint`). The direct-call packages have the lowest coverage gates:
`coverage` 17.9, `gotest` 35.7, `commands` 63.0, `deps` 64.7, `tag` 70.8,
`lint` 72.2. The `lint` service tests take about 47 seconds because they run
the real tools.

**Fix:** Add a small `Runner` interface in `cmdutil` (run, with
stdout/stderr, env and dir) with a real implementation and a fake for tests.
Route all subprocess calls through it, remove `ExecOutput`, then raise the
low gates with `covratchet`.

**Effort:** M

### P1-7: covgate and covratchet duplicate logic and have drifted

**Where:** `covgate.go` and `covratchet.go`: `effectiveParallelism` (with a
"keep in sync" comment), `runPackages`, `progressColWidth`, `firstLine`,
header printing, `.covgate` reading.

**Problem:** About 150 lines are copied, and the behavior has already
drifted:

- covratchet has no `--exclude` and no orphan test handling under
  `--test-dir`.
- Invalid `.covgate` handling differs (see P0-2).

**Fix:** Move the shared worker pool, progress output and `.covgate`
read/write into `gocover` (or a new `covgatefile` package). Give covratchet
the same `--exclude` behavior. Inside each package, also share the "tests
failed" row formatting that `checkPackage` and `checkTestOnly` duplicate.

**Effort:** M

### P1-8: covratchet hides why a package failed

**Where:** `internal/services/covratchet/covratchet.go:214`
(`actual, _, err := r.measure(...)`).

**Problem:** When tests fail, the test output is thrown away and the row
just says `FAIL — —`. The user has to rerun `go test` by hand to see why.
covgate prints the output in the same situation.

**Fix:** Keep the output and print it under the FAIL row, as covgate does.

**Effort:** S

### P1-9: `exhaustruct` causes a lot of `//nolint` noise

**Problem:** There are 52 `//nolint` lines in non-test code, and 31 are for
`exhaustruct`. They are mostly `cobra.Command` literals and
`return analysis.Candidate{}, false`. Counting tests, `exhaustruct` appears
149 times. That hides the suppressions that matter.

**Fix:** Configure the linter instead: exclude `cobra.Command` with
`exhaustruct.exclude`, and allow empty literals in returns
(`allow-empty-returns`). Remove the now-redundant comments. Enable
`nolintlint` once P1-3 is done so unused suppressions are flagged.

**Effort:** S

---

## P2: polish

### P2-1: No cancellation, signal handling or timeouts

**Problem:** Every subprocess uses `exec.Command` without a context, with
`//nolint:noctx`. A stuck `go test` hangs `covgate` until the CI job
timeout. On Ctrl-C, deferred cleanup of temp coverage files
(`gocover.MeasureWithEnv`) does not run, so files pile up in `$TMPDIR`.

**Fix:** Create a context with `signal.NotifyContext` in `main`, pass it
through `cmd.Context()`, use `exec.CommandContext`, and add optional
`--timeout` flags for covgate and covratchet.

**Effort:** M

### P2-2: `miru lint` writes files by default

**Where:** `internal/commands/lint.go:45` (`--fix` defaults to `true`).

**Problem:** Any caller that forgets `--fix=false` (or `LINT_FIX=0` in the
script) changes files, and in CI the result is a confusing pass on a dirty
tree. `miru check` already defaults to `false`, so the two commands are
inconsistent.

**Fix:** Default `--fix` to `false` and have `scripts/lint.sh` opt in.
Consumers that rely on the default need a heads-up in the CHANGELOG.

**Effort:** S

### P2-3: CI gaps

- `covgate` runs `go test` without `-race`, though `covgate`, `covratchet`
  and `lint` use goroutines. Add `-race` as an option to covgate, or a
  separate `go test -race ./...` job.
- No `govulncheck` job. Add one (nightly, next to CodeQL, is enough).
- Tests run on Linux only, while the tool runs on developer machines too.
  The Windows lint pass checks that the code compiles there, not that the
  path handling works. A small macOS job is cheap insurance.

**Effort:** S

### P2-4: Command tests check wiring, not behavior

**Where:** `internal/commands/commands_test.go` (847 lines).

**Problem:** Almost every test checks `Use`, `Short`, flag names and
defaults, or help output. None runs a command's `RunE`. The `tag`
subcommands print with `fmt.Println` to `os.Stdout` directly instead of
`cmd.OutOrStdout()`, so their output cannot be captured in tests.

**Fix:** Write to `cmd.OutOrStdout()` and `cmd.ErrOrStderr()` everywhere,
inject services through a small dependency struct, and replace most
metadata tests with a few table tests that run commands against fakes
(this builds on P1-6).

**Effort:** M

### P2-5: Small nits

- `--deadcode-exclude` help says "grep-v pattern", but it is a plain
  substring match and accepts only one value (`lint.go:396`). Fix the help
  text or accept a comma-separated list.
- `FilterDeadcodeOutput` drops lines containing `/go/pkg/mod/`. That breaks
  with a custom `GOMODCACHE`. Compare against `go env GOMODCACHE` instead.
- `imports` hardcodes `github.com/mirurobotics/` as the internal prefix
  (`imports.go:19`). Fine for Miru, but make it a flag with that default.
- Variables of type `checkPackageCtx` and `ratchetCtx` are named `ctx`,
  which reads like a `context.Context`. Rename them to `cfg` or `run`.
- `fmt.Errorf` with no format arguments (`"coverage gate failed"`,
  `"deadcode found issues"`); use `errors.New`.
- `coverage` writes a fixed `coverage.out` in the current directory; add an
  `--out` flag.
- `RunExternal` takes a `name` argument, but every caller passes `"go"`.
- `linter.Run` lints files one by one; a worker pool would speed up large
  repos.
- `FindGoFiles` uses `filepath.Walk`; `filepath.WalkDir` is cheaper.

**Effort:** S each

### P2-6: Go toolchain version

**Where:** `go.mod` (`go 1.25.3`), `.devcontainer/devcontainer.json`
(`go:1.25-bookworm`).

**Problem:** CI builds with the exact version in `go.mod`, so it misses
later 1.25.x patch releases, which include standard library security fixes.
Dependabot does not bump the `go` directive.

**Fix:** Bump to the latest supported patch (or add a `toolchain` line),
and check that 1.25 is still a supported Go release. Consider a Renovate
rule or a monthly reminder to keep it current.

**Effort:** S

### P2-7: `install-lint-tools.sh` is unpinned

**Where:** `scripts/install-lint-tools.sh`.

**Problem:** `go install …/actionlint@latest` means local runs can differ
from CI. yamllint and shellcheck are also unpinned, and the script uses
`sudo apt-get` without `apt-get update`.

**Fix:** Pin actionlint, either through a version variable or as a `tool`
line in `go.mod` like golangci-lint, and pin yamllint.

**Effort:** S

---

## Out of scope

- Rewriting the custom linter on `go/analysis` (mentioned in P1-3). It is
  worth its own design doc if the team wants it.
- Vulnerability findings: `govulncheck` could not run in the review sandbox.
