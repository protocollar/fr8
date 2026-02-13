# Contributing

## Development

```bash
go build ./...          # Compile all packages
go vet ./...            # Static analysis
go test -race ./...     # Tests with race detector
golangci-lint run       # Lint (optional locally, enforced in CI)
```

## CI

CI runs automatically on pushes to `main` and all pull requests. It runs three parallel jobs:

- **test** — build, vet, and test with the race detector across two Go versions (stable + oldstable) and three OS (Linux, macOS, Windows)
- **lint** — golangci-lint
- **tidy** — verifies `go.mod` and `go.sum` are clean

All checks must pass before merging.
