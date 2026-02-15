# Internal Package Organization

## Principles

- Organize by **domain** (config, registry, git, userconfig) not by layer (handlers, services)
- Each package has a **single clear responsibility**
- Package names: short, lowercase, noun describing what it contains
- No circular dependencies — extract shared types if needed

## Package Responsibilities

| Package       | Responsibility                                                           |
|---------------|--------------------------------------------------------------------------|
| `config/`     | Load `fr8.json` / `conductor.json` with fallback chain                   |
| `git/`        | All git operations via `os/exec` — no go-git                             |
| `port/`       | Sequential port block allocation                                         |
| `names/`      | Adjective-city name generation                                           |
| `filesync/`   | `.worktreeinclude` glob matching and file copy                           |
| `env/`        | Build `FR8_*` and `CONDUCTOR_*` environment variables                    |
| `workspace/`  | Workspace resolution: CWD -> registry -> explicit                        |
| `registry/`   | Unified repo + workspace state CRUD (`~/.local/state/fr8/repos.json`)    |
| `userconfig/` | User preferences: openers, future settings (`~/.config/fr8/config.json`) |
| `opener/`     | Opener execution (`Run()` only)                                          |
| `tmux/`       | Thin wrapper around tmux CLI for background sessions                     |
| `tui/`        | Bubble Tea dashboard TUI                                                 |
| `exitcode/`   | Exit code constants, `ExitError` type, error classification              |
| `jsonout/`    | JSON output mode — `Write()`, `WriteError()`, `Conciser` interface       |
| `mcp/`        | MCP server for tool integrations                                         |

## Key Patterns

- **Git operations**: always go through `internal/git/` — never shell out directly from `cmd/`
- **Registry/userconfig CRUD**: go through their respective packages with file locking
- **File locking**: `.lock` file + `syscall.Flock(LOCK_EX)` + defer unlock + defer remove
- **Config fallback**: `fr8.json` -> `conductor.json` -> defaults
- **Env vars**: `internal/env/` sets both `FR8_*` and `CONDUCTOR_*` (backwards compat)
- **JSON output**: implement `Conciser` interface for `--concise` mode support
