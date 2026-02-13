# fr8

Go CLI for managing git worktrees as isolated dev workspaces. See `README.md` for user-facing docs.

## Stack

- Go 1.25+
- Cobra (CLI framework)
- doublestar (glob matching for `.worktreeinclude`)
- No other external dependencies — standard library for everything else

## Build & Test

```bash
go build ./...          # Build all packages
go build -o fr8 .       # Build binary
go vet ./...            # Static analysis
go test ./...           # Run tests
go install .            # Install to GOPATH/bin
```

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

- All git operations shell out via `os/exec` — no go-git dependency
- Workspace commands live under `fr8 workspace` (alias `fr8 ws`): new, list, status, env, open, archive, run, start, stop, attach, logs, ps, shell, cd, exec, browser
- Workspace resolution: local (CWD git repo) first, falls back to global registry search when a name is given
- `run`, `exec`, and `attach` commands use `syscall.Exec` to replace the process (clean signal handling)
- `fr8 ws new` drops into a subshell after creation (`--no-shell` to skip); `-r`/`--remote` tracks a remote branch, `-p`/`--pull-request` resolves a GitHub PR (requires `gh`)
- `createWorkspace()` in `cmd/new.go` is the shared creation function used by both CLI and TUI dashboard
- Background process management uses tmux sessions named `fr8/<repo>/<workspace>`; graceful degradation when tmux is not installed
- Workspace openers are stored at `~/.config/fr8/openers.json`; TUI picker shown when multiple are configured
- State is JSON in `.git/fr8.json` with advisory file locking (`syscall.Flock`)
- Sets both `FR8_*` and `CONDUCTOR_*` env vars for backwards compatibility
- Config falls back from `fr8.json` → `conductor.json` in the repo root
