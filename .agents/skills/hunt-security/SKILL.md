---
name: hunt-security
description: Proactively scan a codebase for security vulnerabilities, prioritize by exploitability and impact, then hand off the most critical one to $patch for remediation. Use when asked to find security issues, audit for vulnerabilities, or harden a codebase.
---

# Security Hunt Workflow

## Inputs
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `scope` (optional): files, package, module, or full repo. Defaults to full repo.
- `severity` (optional): minimum severity to act on — `critical`, `high`, `medium`. Defaults to `high`.
- `base` (optional): target branch for the fix. Defaults to `main`.

## References
Use these skills as authoritative lenses during analysis:
- `$security` for trust boundaries, input validation, auth, secrets, crypto, and dependency analysis.
- `$review` for correctness cross-check (some security bugs are logic bugs).
- `$test` for writing exploit-demonstrating tests.
- `$commit` for commit conventions.
- `$patch` for executing the fix.

## Procedure

### 1. Analyze the codebase for security vulnerabilities
- Read project structure, entrypoints, and attack surface (HTTP handlers, CLI inputs, file parsers, database queries, external API calls).
- Systematically scan using the analysis approach from `$security`:
  - Injection surfaces (SQL, command, template, path traversal, XSS).
  - Authentication and authorization bypass paths.
  - Secret/credential exposure in code, config, logs, or error messages.
  - Unsafe cryptographic primitives or weak randomness.
  - Dependency vulnerabilities and supply-chain risk.
  - Insecure deserialization and unsafe type coercion.
  - Missing or misconfigured security headers/CORS/CSP.
- Produce a ranked vulnerability list sorted by exploitability and impact.

### 2. Select the most severe vulnerability
- Pick the highest-severity vulnerability from the list.
- Present the full vulnerability list to the user, indicate which will be fixed, and proceed immediately.
- If no vulnerabilities are found at or above the minimum severity, report that and stop.

### 3. Write exploit-demonstrating tests (when testable)
- If the vulnerability can be demonstrated via a test (e.g. injection, auth bypass, input validation failure):
  - Create a branch from latest `base`: `security/<short-vulnerability-description>`.
  - Write test cases that exploit the vulnerability — they must fail (i.e. the exploit succeeds) on the current code.
  - Run the tests and **capture the failure output**.
  - Commit the failing tests: `test: add exploit tests for <vulnerability description>`.
- If the vulnerability is not practically testable (e.g. hardcoded secret, dependency CVE, misconfigured header):
  - Skip this step. Prepare descriptive evidence instead (what the vulnerability is, where it is, how it could be exploited).

### 4. Hand off to $patch
- Invoke `$patch` with:
  - `task`: what the vulnerability is, its severity/impact, root cause, file paths/line references, and what the remediation should look like.
  - `repo` and `base`: pass through from inputs.
  - `branch`: the branch created in step 3 (if tests were written), so `$patch` continues on it.
  - `evidence`: the captured test failure output (if tests were written) or the descriptive evidence (if not testable).

## Severity Definitions
- **critical**: Exploitable vulnerability allowing unauthorized access, data exfiltration, remote code execution, or credential exposure.
- **high**: Vulnerability exploitable under realistic conditions — auth bypass, injection with reachable input, missing access control on sensitive endpoint.
- **medium**: Weakness exploitable under unusual conditions, or defense-in-depth gap (e.g. missing rate limiting, overly permissive CORS in non-production context).

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Never commit unrelated changes alongside the fix.
- Stop and report if no vulnerabilities meet the severity threshold.

## Output Contract
1. **Vulnerability inventory**: ranked list with exploitability, impact, and attack path for each.
2. **Selected vulnerability**: which one is being fixed and why it ranks highest.
3. **Test evidence**: exploit test output (when testable) or descriptive evidence (when not).
4. **Patch result**: output from `$patch` (task summary, changes, verification, PR link).
