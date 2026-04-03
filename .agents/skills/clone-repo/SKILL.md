---
name: clone-repo
description: Clone and access external git repositories the agent needs to read, reference, or work with. Use this skill whenever you need code, configs, or documentation from another repository — whether to inspect an implementation, check a dependency, read a shared library, cross-reference API contracts, or pull in any external codebase. Also use when a task mentions a repo you don't have locally, when you need to look something up in another project, or when the user asks you to check, clone, or fetch a repository.
---

# Clone Repository

Clone external repositories to a canonical local location so they can be reused across tasks without redundant clones.

## Inputs

- `repo` (required): repository identifier — one of:
  - `owner/repo` (GitHub shorthand, e.g. `mirurobotics/backend`)
  - `git@github.com:owner/repo.git` (SSH URL)
  - `https://github.com/owner/repo.git` (HTTPS URL)

## Canonical Clone Location

All cloned repositories go in `.agents/modules/` relative to the project root (the directory containing AGENTS.md). This directory is gitignored — never commit its contents.

Use the repository name as the subdirectory (e.g. `.agents/modules/backend/` for `mirurobotics/backend`). If two repos from different owners share a name, namespace them: `.agents/modules/owner-repo/`.

## Procedure

### 1. Check for existing clone

Before cloning, check if the repo is already available locally:

```bash
ls .agents/modules/<repo-name>/.git 2>/dev/null
```

If it exists, update it instead of re-cloning:

```bash
git -C .agents/modules/<repo-name> pull --ff-only
```

If the pull fails (e.g. detached HEAD, conflicts), delete and re-clone rather than debugging.

### 2. Check accessibility

Verify the repo is reachable before attempting a full clone. Try these in order — use the first that works:

**GitHub CLI** (preferred — respects existing auth session):
```bash
gh repo view owner/repo --json name -q .name
```

**git ls-remote** (works with any auth method):
```bash
git ls-remote --exit-code <repo-url> HEAD
```

**SSH connectivity** (verify SSH keys work for GitHub):
```bash
ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"
```

If none succeed, stop and tell the user the repo is not accessible. Suggest they check:
- Do they have access to this repo?
- Is `gh auth status` showing a valid session?
- Are SSH keys configured (`ssh -T git@github.com`)?
- Is a `GH_TOKEN` or `GITHUB_TOKEN` environment variable set?

### 3. Clone

Use shallow clones by default to save time and disk. Choose the auth method based on what's available:

**GitHub CLI** — simplest, uses existing `gh auth` session:
```bash
gh repo clone owner/repo .agents/modules/<repo-name> -- --depth 1
```

**SSH** — when SSH keys are configured:
```bash
git clone --depth 1 git@github.com:owner/repo.git .agents/modules/<repo-name>
```

**HTTPS with token** — for CI or automated environments where `gh` and SSH aren't available:
```bash
git clone --depth 1 https://x-access-token:${GH_TOKEN}@github.com/owner/repo.git .agents/modules/<repo-name>
```

If you need full history (e.g. for `git log` or `git blame`), omit `--depth 1`. Prefer shallow clones unless the task specifically requires history.

### Auth method selection

Try methods in this order:
1. **`gh` CLI** if `gh auth status` succeeds — this is the most portable method and handles token management automatically.
2. **SSH** if `ssh -T git@github.com` authenticates — common for developer machines.
3. **HTTPS with token** if `GH_TOKEN` or `GITHUB_TOKEN` is set in the environment — typical in CI.

## Safety Rules

- Never commit the contents of `.agents/modules/` — it is gitignored for a reason.
- Never store tokens or credentials in files. Use environment variables or `gh` auth.
- Never modify files in a cloned repo unless explicitly asked. Treat clones as read-only references by default.
- Never clone into a location outside `.agents/modules/` — one canonical location ensures reuse and clean gitignore coverage.

## Checking for stale agent context

Consumer repos include agent context (skills, AGENTS.md) via git subtree at `.agents/`. This context can become stale. To check, compare the local version against the remote:

```bash
# Local version (in the consumer repo)
cat .agents/version.txt

# Remote version (source of truth)
gh api repos/mirurobotics/ai/contents/.agents/version.txt --jq '.content' | base64 -d
```

If the remote version is higher, the local agent context is out of date. Inform the user and suggest they sync:

```bash
git subtree pull --prefix=.agents <agents-repo-url> main --squash
```

Only check staleness when the task would benefit from it (e.g., a skill is missing or behaving unexpectedly). Don't check on every invocation.

## Output Contract

1. Path to the cloned (or already-existing) repository.
2. If the repo was inaccessible, a clear explanation of what failed and what the user can do about it.
