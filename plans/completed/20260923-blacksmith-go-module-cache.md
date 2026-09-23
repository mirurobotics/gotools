# Disable setup-go caching and keep an explicit Go module cache

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` | read-write | Edit `.github/workflows/ci.yml` and `.github/workflows/codeql-analysis.yml`. |

Branch: `claude/vigilant-fermi-pojn1o` from `origin/main`.

## Purpose / Big Picture

Blacksmith now provides native Go build caching (GOCACHEPROG) on its runners, so the build cache that `actions/setup-go` saves and restores (`~/.cache/go-build`) is redundant. After this change, every `actions/setup-go` step on a Blacksmith runner sets `cache: false`, and the Go module cache (`~/go/pkg/mod`) is kept by an explicit `actions/cache` step placed right after Setup Go, matching `mirurobotics/backend`.

## Plan of Work

1. In `ci.yml` (`lint` and `test` jobs) and `codeql-analysis.yml` (`codeql` job), add `cache: false` under the Setup Go `with:` block.
2. Directly after each Setup Go step, add:

        - name: Cache Go modules
          uses: actions/cache@55cc8345863c7cc4c66a329aec7e433d2d1c52a9 # v6.1.0
          with:
            path: ~/go/pkg/mod
            key: gomod-${{ runner.os }}-${{ hashFiles('**/go.sum') }}
            restore-keys: |
              gomod-${{ runner.os }}-

3. Change nothing else: no runner, test-flag, or action-pin changes.

## Validation and Acceptance

- `yamllint .github/workflows` and `actionlint` report no errors (where available locally).
- Tests: no Go code changes; the existing `test` job (`./scripts/covgate.sh`) must still pass.
- Preflight must report `CLEAN` (CI green on the pushed branch head) before the PR leaves draft or the task is reported complete.

## Progress

- [x] Edit workflows (ci.yml lint + test, codeql-analysis.yml codeql).
- [x] Local YAML lint (yamllint and actionlint clean).
- [x] Push, open draft PR #45; CI must be green on the head (`CLEAN`).
- [x] Move plan to `plans/completed/`; PR marked ready once CI is green.

## Surprises & Discoveries

None yet.

## Decision Log

- The CodeQL workflow also gets the module cache step, per the task, so Autobuild reuses downloaded modules.

## Outcomes & Retrospective

Three `actions/setup-go` steps now set `cache: false`, each followed by an explicit Go module cache step. No other workflow changes.
