# Fix `isErrorSentinel` to Exempt Custom Error Constructors

**File:** `plans/backlog/20260407-mutableglobal-custom-error-constructors.md`
**Created:** 2026-04-07
**Status:** backlog

---

## 1. Scope

This plan covers a targeted bug fix in one Go source file and one test file inside the `gotools` repository:

- **Source:** `internal/services/lint/linter/mutableglobal/mutableglobal.go`
- **Tests:** `internal/services/lint/linter/mutableglobal/mutableglobal_test.go`

No other files are changed.

---

## 2. Purpose / Big Picture

The `mutableglobal` linter reports a diagnostic whenever a package-level `var` declaration is found, because mutable global state is generally a source of bugs (race conditions, hidden dependencies, etc.). There are explicit exemptions for:

- Interface compliance checks (`var _ SomeInterface = (*Impl)(nil)`)
- Declared-but-suppressed lint via `//nolint:mutableglobal` comments
- Error sentinel values assigned from `errors.New(...)` or `fmt.Errorf(...)`

The problem is that the third exemption is too narrow. In real codebases, teams create their own error constructor functions — for example:

```go
var ErrNotFound   = myerrors.NewNotFound("resource not found")
var ErrUnavailable = grpc.Errorf(codes.Unavailable, "svc unavailable")
var errInternal    = pkg.New("internal error")
```

None of these are exempted by the current code. The linter fires a false positive for every one.

The fix adds two pieces of logic:

1. **Variable-name guard:** Only skip the diagnostic when the variable name starts with `Err` or `err` (conventional Go error naming). A var named `counter` calling `pkg.New(...)` should still be flagged.
2. **Generalised constructor recognition:** Treat any qualified call (`pkg.Fn(...)`) whose function name starts with `"New"` or ends with `"Errorf"` as an error constructor — not just the hard-coded `errors.New` / `fmt.Errorf` pair.

---

## 3. Progress

- [ ] Milestone 1 — Add `isErrVarName` helper and update `isErrorSentinel` signature
- [ ] Milestone 2 — Expand `isErrorConstructor` to handle generic constructors
- [ ] Milestone 3 — Update call site in `Check`
- [ ] Milestone 4 — Add new test cases and verify all tests pass
- [ ] Milestone 5 — Final commit

---

## 4. Surprises & Discoveries

_Nothing discovered yet. Update this section as you work._

---

## 5. Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-07 | Gate exemption on var name starting with `Err`/`err` | Prevents false negatives: `var cache = pkg.New(...)` must still be flagged |
| 2026-04-07 | Match function names starting with `"New"` or ending with `"Errorf"` | Covers `NewNotFound`, `NewError`, `grpc.Errorf`, etc. without an allowlist |
| 2026-04-07 | Do not match unqualified calls (bare `New(...)`) | Unqualified calls are unusual for sentinel errors and too broad to exempt |

---

## 6. Outcomes & Retrospective

_Fill in after the work is merged._

---

## 7. Context and Orientation

### Repository layout

```
gotools/
  cmd/                        # CLI entry points
  internal/
    services/
      lint/
        linter/
          mutableglobal/
            mutableglobal.go       ← source file to change
            mutableglobal_test.go  ← test file to change
          analysis/
            analysis.go            ← Diagnostic type, IsTestFile helper
  go.mod
  go.sum
  scripts/
```

### Key types

- `ast.ValueSpec` — one line inside a `var (...)` block or a bare `var` statement. Fields relevant here:
  - `Names []*ast.Ident` — the variable names on the left-hand side
  - `Values []ast.Expr` — the right-hand side expressions (may be shorter than `Names` if some vars are uninitialized)
- `ast.Ident` — an identifier node; `.Name` is the plain string (e.g. `"ErrNotFound"`)
- `ast.CallExpr` — a function call node; `.Fun` holds the callee expression
- `ast.SelectorExpr` — a qualified expression like `pkg.Fn`; `.X` is the left side (`pkg`), `.Sel` is the right side (`Fn`)

### Current `isErrorSentinel` (lines 73-79)

```go
func isErrorSentinel(expr ast.Expr) bool {
    call, ok := expr.(*ast.CallExpr)
    if !ok {
        return false
    }
    return isErrorConstructor(call)
}
```

It does **not** receive the variable name, so it cannot apply the name guard.

### Current `isErrorConstructor` (lines 81-100)

```go
func isErrorConstructor(call *ast.CallExpr) bool {
    sel, ok := call.Fun.(*ast.SelectorExpr)
    if !ok { return false }
    ident, ok := sel.X.(*ast.Ident)
    if !ok { return false }
    pkg, fn := ident.Name, sel.Sel.Name
    if pkg == "errors" && fn == "New" { return true }
    if pkg == "fmt"    && fn == "Errorf" { return true }
    return false
}
```

Only two hard-coded cases are accepted.

### Call site (lines 45-53 in `Check`)

```go
for i, name := range vs.Names {
    if isInterfaceCheck(name) { continue }
    if i < len(vs.Values) && isErrorSentinel(vs.Values[i]) { continue }
    // ... report diagnostic
}
```

`name` (`*ast.Ident`) is already available here — it just needs to be threaded through to `isErrorSentinel`.

### Running tests

All commands are run from the `gotools/` directory:

```
cd /home/ben/miru/workbench2/gotools

go test ./internal/services/lint/linter/mutableglobal/...
go test ./internal/services/lint/...
go test ./...
```

---

## 8. Plan of Work

### Milestone 1 — Add `isErrVarName` and update `isErrorSentinel` signature

Add a small helper that encodes the naming convention, then extend `isErrorSentinel` to accept the variable name and short-circuit when the name does not look like an error sentinel.

### Milestone 2 — Expand `isErrorConstructor`

Broaden the constructor recognition from two hard-coded pairs to:
- Any qualified call (`pkg.Fn`) where `fn` starts with `"New"`, **or**
- Any qualified call where `fn` ends with `"Errorf"`

The two original cases (`errors.New`, `fmt.Errorf`) are still covered by these new rules.

### Milestone 3 — Update call site

Pass `name` as the first argument to `isErrorSentinel` at the single call site in `Check`.

### Milestone 4 — New test cases

Add four table-driven tests to `mutableglobal_test.go` covering:
1. Custom `NewFoo` constructor with `Err`-prefixed var name → 0 diagnostics
2. `grpc.Errorf` with `Err`-prefixed var name → 0 diagnostics
3. Lowercase `err`-prefixed var name with `pkg.New` → 0 diagnostics
4. Non-`Err`/`err` var name with `pkg.New` → 1 diagnostic (still flagged)

### Milestone 5 — Verify and commit

Run the full test suite, confirm it is green, commit.

---

## 9. Concrete Steps

> **Working directory for all commands:** `/home/ben/miru/workbench2/gotools`

---

### Step 1 — Baseline: confirm tests pass before touching anything

    cd /home/ben/miru/workbench2/gotools
    go test ./internal/services/lint/linter/mutableglobal/...

Expected output:

    ok  	github.com/mirurobotics/gotools/internal/services/lint/linter/mutableglobal	0.XXXs

If this fails, do not proceed — investigate the pre-existing failure first.

---

### Step 2 — Add `isErrVarName` helper to `mutableglobal.go`

Open `internal/services/lint/linter/mutableglobal/mutableglobal.go`.

After the closing brace of `isInterfaceCheck` (line 68), add the following function. Insert it before `isErrorSentinel`:

```go
// isErrVarName returns true when name follows the Go convention for error
// sentinels: it must start with "Err" (exported) or "err" (unexported) and
// be at least 3 characters long (so the bare string "er" is not matched).
func isErrVarName(name string) bool {
    if len(name) < 3 {
        return false
    }
    return strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "err")
}
```

Note: `strings` is already imported in this file.

---

### Step 3 — Update `isErrorSentinel` signature and body

Replace the existing `isErrorSentinel` function (lines 70-79) with:

```go
// isErrorSentinel returns true when varName follows the error-sentinel naming
// convention (starts with "Err" or "err") AND expr is a call to a recognised
// error constructor: errors.New, fmt.Errorf, any pkg.NewXxx, or any pkg.Xxxerrorf.
func isErrorSentinel(varName *ast.Ident, expr ast.Expr) bool {
    if !isErrVarName(varName.Name) {
        return false
    }
    call, ok := expr.(*ast.CallExpr)
    if !ok {
        return false
    }
    return isErrorConstructor(call)
}
```

---

### Step 4 — Expand `isErrorConstructor`

Replace the existing `isErrorConstructor` function (lines 81-100) with:

```go
// isErrorConstructor returns true for qualified calls that are conventional
// error constructors:
//   - pkg.New(...)       — exact name "New"
//   - pkg.NewFoo(...)    — name starts with "New"
//   - pkg.Errorf(...)    — name ends with "Errorf"
//
// This covers errors.New, fmt.Errorf, grpc.Errorf, myerrors.NewNotFound, etc.
func isErrorConstructor(call *ast.CallExpr) bool {
    sel, ok := call.Fun.(*ast.SelectorExpr)
    if !ok {
        return false
    }
    _, ok = sel.X.(*ast.Ident)
    if !ok {
        return false
    }
    fn := sel.Sel.Name
    if strings.HasPrefix(fn, "New") {
        return true
    }
    if strings.HasSuffix(fn, "Errorf") {
        return true
    }
    return false
}
```

---

### Step 5 — Update the call site in `Check`

Find this line (around line 50):

```go
if i < len(vs.Values) && isErrorSentinel(vs.Values[i]) {
```

Change it to:

```go
if i < len(vs.Values) && isErrorSentinel(name, vs.Values[i]) {
```

---

### Step 6 — Verify the file compiles

    cd /home/ben/miru/workbench2/gotools
    go build ./internal/services/lint/linter/mutableglobal/...

Expected output: no output (success). If there is an error, fix it before continuing.

---

### Step 7 — Add new test cases to `mutableglobal_test.go`

Open `internal/services/lint/linter/mutableglobal/mutableglobal_test.go`.

Locate the last entry in the `tests` slice (the `"non-error call expression flagged"` case, which ends with a `},` around line 229). Insert the following four new cases immediately after it, before the closing `}` of the slice:

```go
{
    name:     "custom NewFoo sentinel allowed",
    filename: "foo.go",
    src: `package foo

var ErrNotFound = myerrors.NewNotFound("x")
`,
    wantN:   0,
    wantMsg: "",
},
{
    name:     "grpc Errorf sentinel allowed",
    filename: "foo.go",
    src: `package foo

var ErrUnavailable = grpc.Errorf("svc")
`,
    wantN:   0,
    wantMsg: "",
},
{
    name:     "lowercase err sentinel with New allowed",
    filename: "foo.go",
    src: `package foo

var errInternal = pkg.New("x")
`,
    wantN:   0,
    wantMsg: "",
},
{
    name:     "non-Err var with New function still flagged",
    filename: "foo.go",
    src: `package foo

var counter = pkg.New("x")
`,
    wantN:   1,
    wantMsg: "",
},
```

---

### Step 8 — Run the mutableglobal tests

    cd /home/ben/miru/workbench2/gotools
    go test -v ./internal/services/lint/linter/mutableglobal/...

Expected output (all subtests listed, all PASS):

    === RUN   TestCheck
    === RUN   TestCheck/const_is_allowed
    === RUN   TestCheck/function_is_allowed
    === RUN   TestCheck/mutable_var_flagged
    ...
    === RUN   TestCheck/custom_NewFoo_sentinel_allowed
    === RUN   TestCheck/grpc_Errorf_sentinel_allowed
    === RUN   TestCheck/lowercase_err_sentinel_with_New_allowed
    === RUN   TestCheck/non-Err_var_with_New_function_still_flagged
    --- PASS: TestCheck (0.XXs)
    PASS
    ok  	github.com/mirurobotics/gotools/internal/services/lint/linter/mutableglobal	0.XXXs

---

### Step 9 — Run the broader lint package tests

    cd /home/ben/miru/workbench2/gotools
    go test ./internal/services/lint/...

Expected: all packages `ok`.

---

### Step 10 — Run the full test suite

    cd /home/ben/miru/workbench2/gotools
    go test ./...

Expected: all packages `ok`, no failures.

---

### Step 11 — Commit (Milestone 5)

From within the `gotools` git checkout (it is an independent git repo at `/home/ben/miru/workbench2/gotools`):

    cd /home/ben/miru/workbench2/gotools
    git add internal/services/lint/linter/mutableglobal/mutableglobal.go \
            internal/services/lint/linter/mutableglobal/mutableglobal_test.go
    git commit -m "fix(mutableglobal): exempt custom error constructors from mutable-global lint

isErrorSentinel now requires the variable name to start with Err or err, and
isErrorConstructor now matches any pkg.NewXxx or pkg.Xxxerrorf call, not just
the hard-coded errors.New / fmt.Errorf pair."

---

## 10. Validation and Acceptance

The change is complete when **all** of the following are true:

1. `go test ./internal/services/lint/linter/mutableglobal/...` reports `ok` with zero failures.
2. All four new test cases pass individually when run with `-run TestCheck/<name>`.
3. `go test ./...` reports `ok` for every package — no regressions.
4. The following specific behaviours hold:
   - `var ErrNotFound = myerrors.NewNotFound("x")` → 0 diagnostics
   - `var ErrUnavailable = grpc.Errorf("svc")` → 0 diagnostics
   - `var errInternal = pkg.New("x")` → 0 diagnostics
   - `var counter = pkg.New("x")` → 1 diagnostic
   - `var Err = errors.New("x")` → 0 diagnostics (exactly "Err", len=3, passes the `>= 3` guard)
   - `var errFoo = errors.New("x")` → 0 diagnostics
   - `var error = errors.New("x")` → 1 diagnostic (does not start with "Err"/"err")
   - `var myErr = errors.New("x")` → 1 diagnostic (does not start with "Err"/"err")
   - Pre-existing test cases `"error sentinel allowed"` and `"fmt.Errorf sentinel allowed"` still pass.

Preflight must report `clean` before a PR is opened.

---

## 11. Idempotence and Recovery

**If something goes wrong mid-edit**, the safest recovery is:

    cd /home/ben/miru/workbench2/gotools
    git diff internal/services/lint/linter/mutableglobal/

Review the diff. If it is badly broken, restore from git:

    git checkout -- internal/services/lint/linter/mutableglobal/mutableglobal.go
    git checkout -- internal/services/lint/linter/mutableglobal/mutableglobal_test.go

Then start the Concrete Steps from Step 2 again.

**Re-running steps is safe** — all edits are replacing specific named functions, so applying the same replacement twice will produce the same result (Go compilation will catch any duplication immediately).

**Each step is independently verifiable** via `go build` (Step 6) before the tests are added, so compilation errors are caught early.

**The commit in Step 11** is the only git mutation. If you need to undo it:

    git revert HEAD

This creates a new revert commit rather than rewriting history, which is safe even if the branch has been pushed.
