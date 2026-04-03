---
name: review
description: Perform structured code review focused on correctness, test coverage, and regressions, with findings prioritized by severity and grounded in file/line evidence. Use when asked to review, audit, assess risk, or evaluate change quality before merge or commit.
---

# Review Workflow

## Core Principle

Every finding must be **actionable** — it must require a code change or a deliberate decision from the author. If the fix direction is "leave as-is", "no change needed", or "worth confirming", it is not a finding. Drop it.

**Excluded from review findings:**
- Style, formatting, and naming — that's the linter's job.
- "Verify this compiles" — if it compiles, it's fine. Check it yourself.
- Observations about things that are "probably fine" — if you think it's fine, don't mention it.
- Things that look intentional — skip them.

## Inputs
- `scope` (optional): files, package, module, branch, or full repo.
- `mode` (optional): `single-pass` (default) or `iterative`.
- `write_review_file` (optional): default `false`; persist artifacts only if explicitly requested.

## References
Use these skills as authoritative review lenses:
- `$design` for design/correctness quality checks.
- `$reliability` for error handling/lifecycle/failure-mode review.
- `$security` for security-sensitive change review.
- `$performance` for performance hotspot and regression review.
- `$test` for test quality and coverage checks.

## Scope and Order
1. Correctness — logic bugs, broken invariants, state corruption, data loss.
2. Test coverage — every changed behavior path has a test that would fail if the change regressed.
3. Security/reliability/performance where applicable.

## Procedure

### 1. Resolve scope
Default scope is **uncommitted changes** (staged + unstaged working tree diff). Only review previous commits if the user explicitly asks for it (e.g., "review the last 3 commits", "review branch X").

Gather the diff and list of changed files — this is the shared context every review lens needs.

### 2. Determine execution mode

**If subagents are available**, launch review lenses in parallel (see step 3a).
**If subagents are not available**, run all lenses sequentially yourself (see step 3b).

Subagent-based review is preferred because each lens gets full context without competing for attention, and the reviews run concurrently. The results are the same either way — the difference is speed and thoroughness.

### 3a. Parallel review (subagents available)

Launch the following subagents **in a single turn** so they run concurrently. Each subagent receives the same scope (diff, changed files, and relevant surrounding code) and must return findings in the format defined in the **Findings Format** section below.

**Correctness subagent:**
> Review the following changes for correctness. For each changed code path, verify:
> - **Logic**: Does the code do what the author intended? Trace inputs through branches, loops, and early returns to confirm the output is correct for all reachable states.
> - **Invariants**: Are pre/post-conditions and type contracts preserved? If a function's callers depend on a guarantee (non-null return, sorted output, unique keys), confirm the change still upholds it.
> - **State transitions**: For stateful code, confirm every transition is reachable, no state is skipped, and no impossible state is introduced.
> - **Boundary values**: Check off-by-one, empty collections, zero/negative/max values, and nil/null/None at every boundary the change touches.
> - **Error paths**: Confirm errors are propagated correctly — not swallowed, not double-wrapped, not silently converted to a default value that hides the failure.
>
> Category: `correctness`

**Test coverage subagent:**
> Review the following changes for test coverage. Load the `$test` skill for authoritative guidance. For each changed behavior, locate the corresponding test. If no test exists, that is a finding. Specifically check:
> - **Happy path**: Is the primary success case tested with a meaningful assertion on the output or side effect?
> - **Error/failure path**: If the change adds or modifies error handling, is there a test that triggers the error and asserts the correct behavior (error type, message, recovery)?
> - **Boundary/edge cases**: Are boundary values covered by tests? If a function now handles empty input differently, there must be a test for empty input.
> - **Regression protection**: If this change fixes a bug, is there a test that reproduces the original bug and would fail if the fix were reverted?
> - **Assertion quality**: Tests that assert on implementation details (mock call counts, internal state) instead of observable behavior are weak — flag them if they would not catch a real regression.
>
> Category: `test-coverage`

**Design subagent:**
> Review the following changes for design quality. Load the `$design` skill for authoritative guidance. Focus on structural issues that will cause real problems — not subjective preferences.
>
> Category: `design`

**Security subagent:**
> Review the following changes for security issues. Load the `$security` skill for authoritative guidance. Focus on input handling, auth boundaries, secret management, and injection vectors in the changed code.
>
> Category: `security`

**Reliability subagent:**
> Review the following changes for reliability issues. Load the `$reliability` skill for authoritative guidance. Focus on error handling, resource lifecycle, cancellation/timeout behavior, and failure recovery in the changed code.
>
> Category: `reliability`

**Performance subagent:**
> Review the following changes for performance issues. Load the `$performance` skill for authoritative guidance. Focus on hot paths, unnecessary allocations, I/O patterns, and algorithmic complexity in the changed code.
>
> Category: `performance`

Each subagent prompt must include:
1. The full diff and list of changed files.
2. Instructions to read the referenced skill's SKILL.md and relevant checklists for the detected language.
3. The findings format and severity definitions from this skill.
4. An instruction to return findings as a structured list — no preamble, no summary, just findings (or an explicit "no findings" statement).

### 3b. Sequential review (no subagents)

If subagents are not available, perform each review lens yourself in sequence:

1. **Correctness analysis** — apply the correctness checks described in the correctness subagent prompt above.
2. **Test coverage analysis** — apply the test coverage checks described in the test coverage subagent prompt above.
3. **Sensitive path review** — apply `$security`, `$reliability`, and `$performance` lenses to the changed code where applicable.

### 4. Merge and deduplicate findings

After all lenses complete (whether from subagents or sequential execution):

1. **Collect** all findings from every lens.
2. **Deduplicate** — if two lenses flag the same line for the same underlying issue (e.g., correctness and reliability both flag an unhandled error), keep the finding with the more specific fix direction and drop the other.
3. **Self-resolve open questions** — before raising an open question, search the codebase to answer it yourself. Only raise questions you genuinely cannot resolve from the diff and surrounding code.
4. **Sort** by severity (high first), then by file path.

### 5. Present findings

Produce the unified output per the **Output Contract** below.

## Severity Definitions
- **high**: Will cause a bug, data loss, security issue, or regression in production. Also: changed behavior with **no test coverage at all**.
- **medium**: Likely to cause problems under realistic conditions. Also: existing tests that are **too weak to catch a regression** in the changed behavior (e.g., missing error-path test, assertion on implementation detail instead of behavior).

Style, formatting, and naming concerns are never findings at any severity. If something won't break anything and isn't a meaningful design concern, leave it out.

## Findings Format
For each finding include:
- Severity: `high` or `medium`
- Category: `correctness`, `test-coverage`, `design`, `security`, `reliability`, or `performance`
- What will go wrong (not what might go wrong)
- Evidence path with line reference
- Concrete fix direction (a specific change, not "consider" or "worth looking into")

If no findings exist, state that explicitly.

## Artifact Rule
- Default to chat output only.
- Do not write `REVIEWS.md` or other files unless explicitly requested.

## Output Contract
1. **Findings** (highest severity first). Fewer findings, more decisive.
2. **Test coverage summary** — for each changed behavior, state whether it is tested, untested, or weakly tested. Keep this concise: a table or short list, not a paragraph per item.
3. **Open questions** only where the reviewer cannot determine correctness from the diff and codebase.

No hedging language. No "worth confirming", "presumably", "likely intentional". Be direct.
