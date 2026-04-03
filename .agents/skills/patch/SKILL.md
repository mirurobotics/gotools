---
name: patch
description: Execute a small, scoped task end-to-end — branch, implement, test, verify, commit, and open a PR. Use when you have a well-defined task (bug fix, docs improvement, small feature, refactor) and need to land it as a complete unit of work.
---

# Patch Workflow

## Inputs
- `task`: what to do — a clear description of the change (e.g. a bug finding, a docs gap, a small feature request).
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `base` (optional): target branch for the PR. Defaults to `main`.
- `branch` (optional): branch name. Auto-generated from task if omitted.
- `draft` (optional): open as draft PR when `true`.
- `evidence` (optional): pre-collected evidence to include in the PR body (e.g. test failure output that proves a bug exists before the fix).

## References
Delegate to these skills for their respective concerns:
- `$design` for implementation approach and language-specific checks.
- `$test` for writing and verifying tests.
- `$review` for self-review before pushing.
- `$commit` for commit conventions.
- `$pr` for branch naming and pull request creation.

## Procedure

### 1. Understand the task
- Parse the task description into: what is wrong or missing, where it lives, and what the fix looks like.
- Read the relevant code to confirm the task is valid and scoped.
- If the task is ambiguous or too broad, ask the user to narrow it before proceeding.

### 2. Create a branch
- From the latest `base`, create and checkout a new branch.
- Derive the branch name from the task: `<prefix>/<short-description>` using kebab-case.
- Infer prefix from task type: `fix/` for bugs, `docs/` for documentation, `feat/` for features, `refactor/` for cleanup, `perf/` for performance.
- Follow branch naming conventions from `$pr`.

### 3. Implement the change
- Make the minimal change required to complete the task.
- Follow implementation principles from `$design`:
  - Prefer simple, explicit control flow.
  - Keep changes single-purpose and cohesive.
  - Do not refactor surrounding code or add unrelated improvements.

### 4. Write or update tests
- If the change affects behavior, write tests that verify the new behavior.
- For bug fixes: write tests that would have caught the bug.
- For documentation-only changes: skip this step.
- Follow test conventions from `$test`.

### 5. Verify locally
- Run the full local CI suite (lint, typecheck, test, build) as defined by the repo.
- If any check fails, fix the issue before proceeding.
- Do not skip or bypass any checks.

### 6. Commit
- Organize changes into atomic commits following `$commit` conventions.
- Separate test commits from implementation commits when both exist.
- Use Conventional Commit format: `type(scope): summary`.

### 7. Self-review
- Before pushing, review your own changes using `$review`.
- Scope the review to the diff between `base` and the current branch (`git diff <base>...HEAD` plus any uncommitted changes).
- If the review surfaces findings, fix them:
  1. Apply the fix.
  2. Re-run verification (step 5).
  3. Commit the fix (step 6).
  4. Re-review — but only the new changes, not the full diff again.
- Repeat until no findings remain.
- Cap the loop at **5 iterations**. If findings persist after 5 rounds, stop and ask the user — the task may be larger than expected.

### 8. Push and open a PR
- Push the branch with `git push -u origin <branch>`.
- Open a PR using `$pr` conventions.
- PR body must include:
  - **What**: what was changed and why.
  - **How**: approach taken.
  - **Tests**: what test cases were added or updated (if any).
  - **Evidence**: if `evidence` was provided, include it verbatim in a collapsible `<details>` block so reviewers can see pre-fix state without cluttering the PR.

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Never commit unrelated changes alongside the task.
- Always verify CI passes before opening the PR.
- Stop and ask if the task turns out to be larger than expected.

## Output Contract
1. **Task summary**: what was done and why.
2. **Changes made**: files changed with rationale.
3. **Verification**: CI check results (all passing).
4. **PR link**: URL of the opened pull request.
