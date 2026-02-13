---
description: fr8 project development guidance — workspace model, common workflows, code patterns
user-invocable: true
---

# fr8 Development Guide

You are working on fr8, a Go CLI for managing git worktrees as isolated dev workspaces. Use this context to make informed decisions about code changes.

## Core Model

**Workspace** = git worktree + allocated port block + state entry + optional tmux session

Creation flow (`createWorkspace()` in `cmd/new.go`):
1. Generate or validate workspace name
2. Create git worktree (branch from HEAD, track remote, or resolve PR)
3. Allocate next available port block (base + N*range)
4. Record in state file (`.git/fr8.json`)
5. Register repo in global registry (`~/.config/fr8/repos.json`)
6. Sync `.worktreeinclude` files
7. Run setup script if configured

**Resolution**: Workspace lookup follows CWD → global registry → explicit path. MCP tools skip CWD detection.

## Development Workflow

### Before any change
```bash
go build ./...              # Catch compile errors
go vet ./...                # Static analysis
go test -race ./...         # Tests with race detector
```

### Adding a CLI command
1. Create `cmd/<name>.go` with command struct and `runXxx` function
2. Register via `init()` with `parentCmd.AddCommand()`
3. Use `RunE` (never `Run`), return errors (never `os.Exit()`)
4. Add tests
5. See `.claude/rules/cobra-commands.md` for full pattern

### Adding an MCP tool
1. Define tool in `registerMCPTools()` in `cmd/mcp_tools.go`
2. Write handler function
3. Update `TestRegisterMCPTools` expected tools list
4. See `.claude/rules/mcp-server.md` for full pattern

### Adding a TUI view
1. Add `viewXxx` to enum in `messages.go`
2. Create `xxx.go` with `renderXxx(m model)`
3. Add `handleXxxKey` method
4. Wire into `View()` and `handleKey()` switches
5. See `.claude/rules/tui-components.md` for full pattern

### Modifying internal packages
- Git ops → `internal/git/`
- State CRUD → `internal/state/`
- Registry CRUD → `internal/registry/`
- Never shell out to git from `cmd/` directly
- See `.claude/rules/package-organization.md`

## Key Files

| What                 | Where                              |
|----------------------|------------------------------------|
| Workspace creation   | `cmd/new.go` → `createWorkspace()` |
| Workspace resolution | `internal/workspace/resolve.go`    |
| State persistence    | `internal/state/state.go`          |
| Git wrapper          | `internal/git/git.go`              |
| MCP tools            | `cmd/mcp_tools.go`                 |
| TUI entry            | `internal/tui/model.go`            |
| Exit codes           | `internal/exitcode/exitcode.go`    |
| JSON output          | `internal/jsonout/jsonout.go`      |
| Error handling       | `cmd/root.go` → `Execute()`        |

## Rules Reference

All project conventions are in `.claude/rules/`:
- `go-style.md`, `cobra-commands.md`, `testing.md`
- `error-handling.md`, `package-organization.md`, `ci.md`
- `mcp-server.md`, `tui-components.md`, `pull-requests.md`
