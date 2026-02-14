---
name: {{.Name}}
description: >-
  Manage git worktree workspaces via CLI. Create, run, stop, inspect,
  and archive isolated dev environments with per-workspace port ranges
  and environment variables.
---

# {{.Name}} — Workspace Management

fr8 manages git worktrees as isolated dev workspaces. Each workspace gets its
own branch, port range, database prefix, and environment variables.

Source: https://github.com/protocollar/fr8

## CLI Usage

Always use `--json` for structured output. Add `--concise` for minimal fields.
Human messages are suppressed in JSON mode. Errors appear on stderr as JSON:

```json
{"error": "workspace \"my-feature\" not found", "code": "workspace_not_found", "exit_code": 2}
```

## Operations

| Operation         | Command                               | Key Flags                                                                             |
|-------------------|---------------------------------------|---------------------------------------------------------------------------------------|
| List workspaces   | `fr8 ws list --json`                  | `--running`, `--dirty`, `--merged`, `--repo <name>`                                   |
| Get status        | `fr8 ws status <name> --json`         | `--repo <name>`                                                                       |
| Create workspace  | `fr8 ws new <name> --json --no-shell` | `-b <branch>`, `-r <remote>`, `-p <pr>`, `--no-setup`, `--if-not-exists`, `--dry-run` |
| Archive workspace | `fr8 ws archive <name> --json`        | `--force`, `--if-exists`, `--dry-run`                                                 |
| Run dev server    | `fr8 ws run <name> --json`            | `--if-not-running`, `-A` (all)                                                        |
| Stop dev server   | `fr8 ws stop <name> --json`           | `--if-running`, `-A` (all)                                                            |
| Get env vars      | `fr8 ws env <name> --json`            |                                                                                       |
| Get logs          | `fr8 ws logs <name> --json`           | `-n <lines>`                                                                          |
| Rename workspace  | `fr8 ws rename <old> <new> --json`    |                                                                                       |
| List repos        | `fr8 repo list --json`                | `-w` (include workspaces)                                                             |
| Show config       | `fr8 config show --json`              | `--repo <name>`                                                                       |
| Check config      | `fr8 config doctor --json`            | `--fix`, `--repo <name>`                                                              |

## Exit Codes

| Code | Meaning            | Example                                        |
|------|--------------------|------------------------------------------------|
| 0    | Success            |                                                |
| 1    | General error      | Config parse error, script failure             |
| 2    | Not found          | Workspace, repo, or opener doesn't exist       |
| 3    | Already exists     | Workspace or repo name collision               |
| 4    | Not in repo        | Command run outside a git repository           |
| 5    | Dirty workspace    | Uncommitted changes block archive              |
| 6    | Interactive only   | `--json` used with attach/shell/exec/dashboard |
| 7    | tmux not available | tmux required but not installed                |
| 8    | Config error       | Invalid or missing configuration               |

## Idempotency Flags

Use these flags to make commands safe to call repeatedly:

| Flag               | Command                | Behavior                                     |
|--------------------|------------------------|----------------------------------------------|
| `--if-not-exists`  | `ws new`               | Succeed silently if workspace already exists |
| `--if-exists`      | `ws archive`           | Succeed silently if workspace not found      |
| `--if-not-running` | `ws run`               | Succeed silently if already running          |
| `--if-running`     | `ws stop`              | Succeed silently if not running              |
| `--dry-run`        | `ws new`, `ws archive` | Preview what would happen without doing it   |

## Common Workflows

Create and start a workspace:

```bash
fr8 ws new my-feature -b feature/auth --json --no-shell
fr8 ws run my-feature --json
```

Check status and view logs:

```bash
fr8 ws status my-feature --json
fr8 ws logs my-feature --json -n 100
```

Archive when done:

```bash
fr8 ws archive my-feature --json --force
```

## Key Notes

- Always pass `--no-shell` with `fr8 ws new` to prevent an interactive subshell
- Use `--json --concise` for minimal output to reduce token usage
- Use `--repo <name>` when workspace names may overlap across repos
- When `<name>` is omitted, fr8 auto-detects from the current working directory
