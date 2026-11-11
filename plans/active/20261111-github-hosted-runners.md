# Run gotools CI on GitHub-hosted `ubuntu-latest` runners

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` (mirurobotics/gotools) | read-write | Change the six Blacksmith runner labels in `.github/workflows/` to `ubuntu-latest`, and drop the three Blacksmith labels from `.github/actionlint.yml`. No other files change. |
| `docs/` (mirurobotics/docs) | read-only | Exemplar: mirurobotics/docs#320 made the same change and settled the label and the inline comment. |
| `cli/`, `ansible-collection-agent/` | read-only | Precedent: both already run their jobs, including the same pinned reusable AI review workflow, on `ubuntu-latest`. |

This plan lives in `gotools/plans/` because the only edits are in the gotools repo. Work on the checked-out branch `claude/youthful-lamport-yn0sf2` (base: `main`).

Out of scope: the same move in agent and setup-cli, which follow in their own PRs.

## Purpose / Big Picture

mirurobotics/gotools is a public repository. GitHub-hosted standard runners are free for public repositories (Linux: 4 vCPU, 16 GB) and do not draw from the GitHub Enterprise Cloud included compute allowance, while every Blacksmith minute spent here is billed. After this change every job in the gotools workflows runs on GitHub's `ubuntu-latest` runner, so gotools CI costs nothing. The jobs keep their names and steps, so branch protection's required checks are unchanged. A reader sees it working when the job logs on the PR show "Runner name: GitHub Actions …" and an image of `ubuntu-24.04` instead of a Blacksmith runner, and all checks are green.

This PR follows the exemplar mirurobotics/docs#320 and cites it in its body.

## Decisions

### Precedent

| New file | Mirrors |
|----------|---------|
| `plans/active/20261111-github-hosted-runners.md` | `docs/plans/active/20261111-github-hosted-runners.md` (the exemplar's plan) and `plans/completed/20260920-review-refine-lint-goos.md` |

No source file is added. The edited `runs-on` lines mirror the exemplar docs#320, each written as `runs-on: ubuntu-latest # standard GitHub-hosted runners are free in public repos`. The `with: runs-on:` input to the AI review workflow mirrors `cli/.github/workflows/ai-review.yml`, which passes `ubuntu-latest` to `mirurobotics/.github/.github/workflows/ai-review.yml` at the same SHA gotools pins (`bf09161…`).

### Left out

- The agent and setup-cli workflows: proposed follow-up PRs that cite docs#320, at most three open at a time. In agent only the Linux jobs are candidates; its Windows jobs (`blacksmith-4vcpu-windows-2025` in `ci.yml` and `release.yml`) stay on Blacksmith, and `builder.yml` uses `useblacksmith/*` Docker actions that need their own replacement.
- The `ubuntu-latest-m` entry in `.github/actionlint.yml`: it predates the Blacksmith move and no gotools workflow uses it. Proposed follow-up: remove it in a separate PR.
- Larger or ARM runners, timeout or concurrency changes, and bumping the pinned reusable-workflow SHAs: not needed to move off Blacksmith.

### Open questions

- Add `# yamllint disable-line rule:line-length` above each changed line? The new lines are 84 and 86 characters, over yamllint's default 80, and the repo puts a disable comment before its other long lines. Default: no. `.yamllint.yml` sets `line-length` to `level: warning`, and yamllint without `--strict` exits 0 on warnings, so `surface-lint` stays green; the line then matches the exemplar exactly. If a reviewer wants the warnings silenced, add the five disable comments.

## Progress

- [x] Source written
- [x] Replace the six Blacksmith labels with `ubuntu-latest` plus the inline comment, and remove the three Blacksmith labels from `.github/actionlint.yml`.
- [x] Commit `ci: run gotools workflows on GitHub-hosted ubuntu-latest runners`.
- [ ] Push and open the draft PR; CI and a dispatched CodeQL run pass on the branch head.
- [ ] Preflight reports `CLEAN`.
- [ ] After merge: AI Review on the next PR, the `ci.yml` push run on `main` and the next nightly CodeQL run pass on `ubuntu-latest`.

## Surprises & Discoveries

(Add entries as work proceeds.)

## Decision Log

- 2026-11-11: Label, inline comment and core-count reasoning are taken unchanged from the exemplar docs#320, whose review settled them.

## Outcomes & Retrospective

(Summarize at completion.)

## Context and Orientation

A runner is the virtual machine a GitHub Actions job runs on, chosen by the job's `runs-on` key. `blacksmith-2vcpu-ubuntu-2404` and `blacksmith-4vcpu-ubuntu-2404` are labels served by Blacksmith, a paid third-party runner provider; `ubuntu-latest` is GitHub's own hosted Ubuntu runner, which resolves to `ubuntu-24.04` today, the same release. In public repos `ubuntu-latest` has 4 vCPU and 16 GB, so the 4-vCPU jobs keep their core count and the 2-vCPU jobs gain cores. Blacksmith CPUs may be faster per core, so watch the `timeout-minutes: 10` on `lint` and `test` (and inside the reusable `surface-lint` job).

The labels appear six times, all in `.github/workflows/` (line numbers as of `main` at 7c17a25):

- `ai-review.yml:30`, job `ai-review`, `blacksmith-2vcpu-ubuntu-2404`. A `with: runs-on:` input to the reusable workflow `mirurobotics/.github/.github/workflows/ai-review.yml`, which uses it as its job's runner.
- `ci.yml:17` (`lint`) and `:33` (`test`), `blacksmith-4vcpu-ubuntu-2404`.
- `ci.yml:50`, job `surface-lint`, `blacksmith-4vcpu-ubuntu-2404`. A `with: runs-on:` input to the reusable workflow `mirurobotics/.github/.github/workflows/surface-lint.yml@6472b6d…`, whose input defaults to `ubuntu-latest`.
- `codeql-analysis.yml:17` (`changes`) and `:33` (`codeql`), `blacksmith-2vcpu-ubuntu-2404`.

`.github/actionlint.yml` registers custom runner labels for actionlint, a GitHub Actions workflow linter that rejects a `runs-on` label it does not know. It lists `ubuntu-latest-m` and `blacksmith-2vcpu-ubuntu-2404`, `blacksmith-4vcpu-ubuntu-2404`, `blacksmith-8vcpu-ubuntu-2404`. `ubuntu-latest` is a standard label and needs no entry. actionlint runs in CI in the `surface-lint` job: the pinned reusable workflow runs yamllint with `.yamllint.yml`, shellcheck on `scripts/`, and `rhysd/actionlint`, which reads `.github/actionlint.yml`. Locally `scripts/lint-surface.sh` runs the same three tools; it is not run here.

Reading every workflow found no Blacksmith-only behavior: no `useblacksmith/*` actions, no sticky disks, no Docker layer caching. Go caching comes from `actions/setup-go`, which uses the standard GitHub Actions cache on any runner. The other actions are `actions/checkout`, `github/codeql-action` and the composite action `mirurobotics/.github/actions/codeql-gate`, none tied to a runner provider.

When each workflow runs, which decides what the PR itself can prove:

- `ci.yml`: on `pull_request` to `main`, on push to `main`, and `workflow_dispatch`. All three jobs (`lint`, `test`, `surface-lint / surface-lint`) run on the PR.
- `codeql-analysis.yml`: nightly (`cron: '31 7 * * *'`) and `workflow_dispatch` only; it does not run on PRs or pushes. `codeql-gate` skips the scan only on scheduled runs where `main` has not moved, so a dispatched run always scans.
- `ai-review.yml`: on `pull_request_target`, which runs the file from the base branch. The PR's own AI review still runs on Blacksmith; the new runner takes effect on the first PR after merge.

## Plan of Work

In the three files under `.github/workflows/`, replace each Blacksmith label with `ubuntu-latest # standard GitHub-hosted runners are free in public repos`, leaving indentation and every other line untouched. In `.github/actionlint.yml`, delete the three `blacksmith-*` lines and keep `ubuntu-latest-m`. One commit, one milestone.

## Concrete Steps

All commands run from the gotools repo root, `/home/user/gotools`.

1. Confirm the branch and the starting count:

       git branch --show-current          # expect: claude/youthful-lamport-yn0sf2
       grep -rc 'blacksmith-' .github/
       # expect: .github/actionlint.yml:3, .github/workflows/ai-review.yml:1,
       #         .github/workflows/ci.yml:3, .github/workflows/codeql-analysis.yml:2
       #         (dependabot.yml and CODEOWNERS: 0)

   If every count is 0, the change is already applied; skip to step 4.

2. Replace the labels and drop the registrations:

       sed -i 's/runs-on: blacksmith-[24]vcpu-ubuntu-2404$/runs-on: ubuntu-latest # standard GitHub-hosted runners are free in public repos/' .github/workflows/*.yml
       sed -i '/blacksmith-/d' .github/actionlint.yml

3. Check the result (read-only inspection, not validation tooling):

       grep -rn 'blacksmith' .github/          # expect: no output
       grep -rc 'runs-on: ubuntu-latest # standard GitHub-hosted runners are free in public repos' .github/workflows/
       # expect: .github/workflows/ai-review.yml:1, ci.yml:3, codeql-analysis.yml:2
       cat .github/actionlint.yml              # expect: only ubuntu-latest-m under labels
       git diff --stat                         # expect: 4 files, 6 insertions, 9 deletions

4. Commit the milestone:

       git add .github plans/
       git commit -m "ci: run gotools workflows on GitHub-hosted ubuntu-latest runners"

5. Push and open the draft PR (via `$pr`). In the PR body, link docs#320, name the follow-ups for agent and setup-cli, and state which workflows can only be verified after merge.

6. Dispatch CodeQL against the branch, which runs on GitHub, not locally:

       gh workflow run codeql-analysis.yml --ref claude/youthful-lamport-yn0sf2
       gh run list --workflow codeql-analysis.yml --branch claude/youthful-lamport-yn0sf2 --limit 1

7. Run `$preflight` until it reports `CLEAN`. Fix any failure from `gh run view --log-failed`.

No local actionlint, yamllint, `scripts/*.sh`, builds or tests are run.

## Validation and Acceptance

The repo has no test suite for workflow configuration, and this change adds no test. The test is that every touched workflow still runs and passes on GitHub's runner:

- `ci.yml` on the PR: `lint`, `test` and `surface-lint / surface-lint` pass, and each job log's "Set up job" section shows a GitHub-hosted runner with image `ubuntu-24.04`, not Blacksmith. The `surface-lint` actionlint step proves the edited workflows and the trimmed `.github/actionlint.yml` are valid: a workflow still using a removed Blacksmith label would fail it. If `lint` or `test` hit `timeout-minutes: 10`, record the durations in Surprises and raise the timeout in the same PR.
- `codeql-analysis.yml`: the dispatched run from step 6 passes `changes` and runs `codeql` on `ubuntu-latest`.
- Gate: `$preflight` must report `CLEAN`, meaning CI is green on the pushed branch head, before the PR leaves draft.

Verified only after merge, and recorded in Outcomes:

- `ai-review.yml`: the AI review on the next PR after merge runs on `ubuntu-latest` and posts its `ai-review` status. The cli repo already passes `runs-on: ubuntu-latest` to the same reusable workflow at the same SHA.
- `ci.yml` push to `main` and the next nightly CodeQL run pass on `ubuntu-latest`.

## Idempotence and Recovery

Step 2 is safe to rerun; once the labels are gone neither `sed` matches anything. Before committing, `git checkout -- .github` restores the files. After merge, a failure that only `ubuntu-latest` causes (for example a timeout) is fixed forward in a new PR, or reverted with `git revert <sha>` in a PR. Never force-push `main`.
