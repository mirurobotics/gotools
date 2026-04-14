# Plan: Run deadcode in parallel with golangci-lint

## Goal
Save ~45s by running deadcode concurrently with golangci-lint in `runLintSteps`.
Both are independent read-only checks with no shared state.

## Changes

### `internal/services/lint/lint.go`
- When both golangci-lint and deadcode are enabled, launch deadcode in a
  goroutine alongside golangci-lint.
- Buffer deadcode's output so it doesn't interleave with golangci-lint's
  streaming output. Print deadcode output after both complete.
- Collect timing and failure results from both via a channel or struct.
- When only one is enabled, behavior is unchanged (no goroutine).

### `internal/services/lint/lint_test.go`
- Add a test that enables both golangci-lint and deadcode (both skipped via
  NoGolangci/Deadcode flags in existing tests) to verify the parallel path
  produces correct timings and output.

## Validation
- `go test ./...` passes.
- `go tool miru lint` runs successfully.
- Preflight must report `clean` before changes are published.
