# fr8

Go CLI for managing git worktrees as isolated dev workspaces. See `README.md` for user-facing docs.

## Stack

- Go 1.25+
- Cobra (CLI framework)
- doublestar (glob matching for `.worktreeinclude`)
- No other external dependencies — standard library for everything else

## Build & Test

```bash
go build ./...              # Build all packages
go build -o fr8 .           # Build binary
go vet ./...                # Static analysis
go test ./...               # Run tests
go test -race -count=1 ./...  # Exactly what CI runs
go install .                # Install to GOPATH/bin
```

CI also runs `golangci-lint` v2 — see `.claude/rules/ci.md`.

## Project Layout

```
main.go                          # Entry point
cmd/                             # Cobra command definitions (one file per command)
  workspace.go                   # `fr8 workspace` (alias `ws`) parent group
  new.go, list.go, status.go …  # Subcommands under `workspace`
  env.go                         # `fr8 ws env` — export workspace vars
  ws_open.go                     # `fr8 ws open` — open workspace with opener
  start.go, stop.go, attach.go  # Background process management (tmux)
  logs.go, ps.go                 # Background session inspection
  browser.go, dashboard.go       # Browser + interactive TUI
  repo.go                        # `fr8 repo` group (registry management)
  opener.go                      # `fr8 opener` group (opener management)
  resolve.go                     # Shared resolveWorkspace() helper (local → global fallback)
internal/
  config/config.go               # Load fr8.json / conductor.json
  state/state.go                 # Workspace state CRUD (.git/fr8.json)
  git/git.go                     # Shell out to git (worktree, branch, status)
  port/port.go                   # Sequential port block allocation
  names/{names,words}.go         # Adjective-city name generation
  filesync/filesync.go           # .worktreeinclude glob + copy
  env/env.go                     # Build FR8_* and CONDUCTOR_* env vars
  workspace/resolve.go           # Resolve workspace by name, CWD, or global registry
  registry/registry.go           # Global repo registry (~/.config/fr8/repos.json)
  opener/opener.go               # Workspace opener config (~/.config/fr8/openers.json)
  tmux/tmux.go                   # Thin wrapper around tmux CLI for background sessions
  tui/                           # Bubble Tea dashboard TUI
    create_workspace.go          # TUI view for creating workspaces
```

## Testing

- All changes must be covered by existing tests or new tests
- Run `go test ./...` before considering any change complete
- Run `go build ./...` and `go vet ./...` to catch compile errors and static analysis issues

## Conventions

Detailed conventions are in `.claude/rules/`:

- `go-style.md` — Naming, struct tags, design principles
- `cobra-commands.md` — How to add/modify CLI commands
- `testing.md` — Test structure and running tests
- `error-handling.md` — Error wrapping, exit codes, JSON errors
- `package-organization.md` — Internal package responsibilities
- `ci.md` — CI pipeline and cross-platform awareness
- `git.md` — Commit message format and git etiquette
- `mcp-server.md` — MCP tool definitions and handler patterns
- `tui-components.md` — Bubble Tea model, messages, views, styles
- `pull-requests.md` — PR description format and pre-flight checks

Key architectural notes:

- `createWorkspace()` in `cmd/new.go` is the shared creation function used by both CLI and TUI dashboard
- Background process management uses tmux sessions named `fr8/<repo>/<workspace>`; graceful degradation when tmux is not installed
- Workspace openers are stored at `~/.config/fr8/openers.json`; TUI picker shown when multiple are configured
