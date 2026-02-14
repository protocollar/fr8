# CI Pipeline

CI config: `.github/workflows/ci.yml`

## Triggers

- Push to `main`
- Pull requests targeting `main`

## Jobs

### test
- **Matrix**: `[stable, oldstable]` Go versions x `[ubuntu, macos, windows]`
- Runs: `go build ./...`, `go vet ./...`, `go test -race -count=1 ./...`
- Tests must be race-safe and deterministic

### lint
- **Runs on**: ubuntu only
- `golangci-lint` v2 — don't introduce lint violations

### tidy
- **Runs on**: ubuntu only
- Checks `go mod tidy` produces no diff — run it after dependency changes

## Cross-Platform Awareness

- Tests run on Windows — `syscall.Flock` is Unix-only
- File-locking code (`state/`, `registry/`) needs build tags or conditional compilation if tests fail on Windows
- Use `filepath.Join()` not string concatenation for paths
- Use `os.UserConfigDir()` not hardcoded `~/.config`
