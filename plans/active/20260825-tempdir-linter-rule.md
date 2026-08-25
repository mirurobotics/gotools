# tempdir linter rule

## Goal

Add a custom linter rule, `tempdir`, that forbids creating temporary test
directories with the `testing` package's `TempDir` method. Miru Go code must
use `test_dirs.CreateTemp(t)` from `github.com/mirurobotics/core/tests/dirs`
instead, so every temp dir is a `dirs.Dir` rather than a bare string path.

## Background

`core/tests/dirs` wraps `t.TempDir()` once:

    func CreateTemp(t *testing.T) dirs.Dir { return dirs.New(t.TempDir()) }

Call sites that reach for `t.TempDir()` directly get a `string` and bypass the
`dirs.Dir` abstraction the rest of the codebase is written against. The rule
pushes call sites onto the wrapper and leaves exactly one sanctioned
`t.TempDir()` — inside the wrapper itself, marked `//nolint:tempdir`.

`bgctx` is the closest existing precedent: it bans `context.Background()` in
favour of the `mctx` helpers, gates on the relevant import being present, and
offers a same-line `//nolint` escape hatch. `tempdir` follows that shape.

## Design

New package `internal/services/lint/linter/tempdir` exporting:

    func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic

### Detection

The check is AST-only (no type information), so it identifies testing handles
structurally:

1. Resolve the local name of the `"testing"` import. If the file does not
   import `testing`, no identifier in it can be a `*testing.T`, so return
   early with no diagnostics.
2. Walk every `*ast.Field` in the file and collect the names declared with a
   testing handle type: `*testing.T`, `*testing.B`, `*testing.F`, or
   `testing.TB`. `*ast.Field` covers function parameters, method receivers,
   and struct fields in one pass, so fixture structs holding a `t` field are
   caught alongside plain `func TestX(t *testing.T)`.
3. Flag every call `<base>.TempDir()` whose base resolves to a collected name.
   The base is the identifier in `t.TempDir()` and the trailing selector in
   `s.t.TempDir()`, so both direct and fixture-held handles are reported.

Gating on the `testing` import and on a declared handle name keeps the rule
from flagging unrelated types that happen to expose a `TempDir()` method.

### Exemptions

- A call with a `//nolint:tempdir` comment on the same line. This is how the
  wrapper in `core/tests/dirs` will exempt its single sanctioned call.

Test files are deliberately *not* exempt — unlike `bgctx`, `_test.go` files
are the rule's primary target.

### Message

    t.TempDir() is not allowed; use test_dirs.CreateTemp(t) to create temp dirs

### Registration

`internal/services/lint/linter/run.go`:
- import the new package
- add `RuleTempDir Rule = "tempdir"`
- add it to `AllRules()`
- add a non-fixable entry to the `ruleCheckers` dispatch table

### Self-exclusion

`gotools` does not depend on `core`, so `test_dirs.CreateTemp` is not
available to its own tests and its 20+ existing `t.TempDir()` calls are
legitimate. `scripts/lint.sh` already excludes `bgctx` for the same reason
(no `mctx` dependency); add `tempdir` to that exclusion list and note why.

## Steps

1. Create `internal/services/lint/linter/tempdir/tempdir.go` with `Check`.
2. Create `internal/services/lint/linter/tempdir/tempdir_test.go`.
3. Register the rule in `run.go`.
4. Add `tempdir` to the `--exclude` list in `scripts/lint.sh`.
5. Add `internal/services/lint/linter/tempdir/.covgate` with a threshold
   within the 0.5pp tightness tolerance of measured coverage.

## Test steps

Table-driven `TestCheck` in the package, mirroring `bgctx_test.go`:

- `t.TempDir()` in a `func TestX(t *testing.T)` → 1 diagnostic, exact message
- `b.TempDir()` on a `*testing.B` → flagged
- `f.TempDir()` on a `*testing.F` → flagged
- `tb.TempDir()` on a `testing.TB` → flagged
- handle held in a fixture struct field, called as `s.t.TempDir()` → flagged
- file that does not import `testing` → not flagged
- `TempDir()` called on a non-testing receiver (a local type with its own
  `TempDir` method) → not flagged
- `t.TempDir()` with `//nolint:tempdir` on the same line → not flagged
- `//nolint:tempdir` on the preceding line → still flagged
- `//nolint:bgctx` on the same line → still flagged
- aliased `testing` import (`import tst "testing"`) → flagged
- multiple calls in one file → each flagged
- other `t` methods (`t.Helper()`, `t.Cleanup()`) → not flagged

Verification:

- `./scripts/test.sh` (or `go test ./...`) passes
- `./scripts/covgate.sh` passes, including the tightness check
- `LINT_FIX=0 ./scripts/lint.sh` passes

## Validation

Preflight must report `CLEAN` — CI green on the pushed branch head — before
the PR leaves draft or the task is reported complete.

## Follow-up (out of scope, separate repo)

Once the rule ships, `core` must add `//nolint:tempdir` to the sanctioned
calls in `tests/dirs/os.go` and `tests/files/os.go`, and migrate its
remaining direct `t.TempDir()` call sites to `test_dirs.CreateTemp`.
