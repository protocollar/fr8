# Testing Conventions

## File Placement

- Test files live next to source: `state.go` + `state_test.go`
- Every `internal/` package has a `_test.go` file

## Test Structure

- **Table-driven tests** for parameterized cases:
  - Name the slice `tests`
  - Loop variable `tt`
  - Always use `t.Run(tt.name, ...)`
- **Simple standalone tests** are fine for single-case scenarios — not everything needs a table

## Assertions

- `t.Fatal()` / `t.Fatalf()` for setup failures (stops the test immediately)
- `t.Error()` / `t.Errorf()` for assertion failures (continues to report all failures)

## Filesystem Tests

- Use `t.TempDir()` for any test that touches the filesystem — automatically cleaned up

## External Commands

- For `os/exec`-dependent code (git, tmux): test wrapper functions with real binaries when feasible
- Use the test helper process pattern (`TestHelperProcess` + `GO_TEST_HELPER` env var) when mocking external commands

## No Mocking Frameworks

- Use real implementations with temp dirs
- Standard library is sufficient — no testify, gomock, etc.

## Running Tests

```bash
go test ./...              # Quick local run
go test -race ./...        # Race detector (CI uses this)
go test -race -count=1 ./... # Exactly what CI runs (no caching)
go vet ./...               # Static analysis
go build ./...             # Catch compile errors
```

## Requirements

- All new code must have test coverage
- Run `go test ./...` before considering any change complete
- Tests must be race-safe and deterministic (CI runs with `-race`)
