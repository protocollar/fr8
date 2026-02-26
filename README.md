# fr8

Git worktree workspace manager. Creates isolated development environments with per-workspace port ranges, database prefixes, and environment variables.

## Install

```bash
go install github.com/protocollar/fr8@latest
```

**Supported platforms:** macOS, Linux. Windows is not currently supported.

## Quick Start

```bash
# In your project root, create a config file
cat > fr8.json <<'EOF'
{
  "scripts": {
    "setup": "bin/setup-workspace",
    "run": "bin/run-workspace",
    "archive": "bin/archive-workspace"
  }
}
EOF

# Create a workspace
fr8 ws new my-feature -b feature/auth

# Start the dev server in the background
fr8 ws run my-feature

# Run a command in the workspace
fr8 ws exec my-feature -- npm test

# Tear it down when done
fr8 ws archive my-feature
```

## Commands

All workspace commands live under `fr8 ws` (alias `fr8 workspace`).

| Command                                                       | Description                                            |
|---------------------------------------------------------------|--------------------------------------------------------|
| `fr8 ws new [name] [-b branch] [-r branch] [-p PR]`           | Create a workspace and drop into a shell               |
| `fr8 ws list [--running] [--dirty] [--merged]`                | List all workspaces (with optional filters)            |
| `fr8 ws rename <old> <new>`                                   | Rename a workspace                                     |
| `fr8 ws status [name]`                                        | Show workspace details and environment variables       |
| `fr8 ws env [name]`                                           | Print FR8_* env vars as `export` statements            |
| `fr8 ws open [name] [--opener name]`                          | Open workspace with a configured opener                |
| `fr8 ws run [name] [-A/--all]`                                | Run the dev server in a background tmux session        |
| `fr8 ws stop [name] [-A/--all]`                               | Stop a workspace's background tmux session             |
| `fr8 ws attach [name]`                                        | Attach to a running background session                 |
| `fr8 ws logs [name] [-n lines] [-f]`                          | Show recent output from a background session           |
| `fr8 ws ps`                                                   | List all running fr8 workspace sessions                |
| `fr8 ws exec [name] -- <cmd>`                                 | Run a command with workspace environment               |
| `fr8 ws shell [name]`                                         | Open a subshell with workspace environment             |
| `fr8 ws cd [name]`                                            | Print workspace path                                   |
| `fr8 ws browser [name]`                                       | Open workspace dev server in the browser               |
| `fr8 ws archive [name] [--force]`                             | Tear down workspace (archive script + remove worktree) |
| `fr8 dashboard`                                               | Interactive TUI for browsing repos and workspaces      |
| `fr8 config show\|doctor [--fix]`                             | View config or check health (fix issues with --fix)    |
| `fr8 repo add\|list\|remove`                                  | Manage the global repo registry                        |
| `fr8 opener add\|list\|remove\|set-default`                   | Manage workspace openers (e.g. VSCode, Cursor)         |
| `fr8 completion [bash\|zsh\|fish]`                            | Generate shell completions                             |
| `fr8 mcp serve`                                               | Start MCP server on stdio (for AI agent integration)   |
| `fr8 skill install [--claude\|--codex] [--global\|--project]` | Install agent skill for CLI-based AI integration       |

All `fr8 ws` subcommands accept a `--repo <name>` flag to target a specific registered repo, which is useful when workspace names overlap across repos.

When `[name]` is omitted, fr8 auto-detects the current workspace from your working directory. When a name is provided, it also works from outside a git repo by searching all registered repos. If multiple repos contain a workspace with the same name, fr8 lists the matches and asks you to disambiguate with `--repo`.

## Configuration

Create `fr8.json` in your repo root:

```json
{
  "scripts": {
    "setup": "bin/setup-workspace",
    "run": "bin/run-workspace",
    "archive": "bin/archive-workspace"
  },
  "port_range": 10,
  "base_port": 60000,
  "worktree_path": "~/fr8"
}
```

| Field             | Default | Description                                                           |
|-------------------|---------|-----------------------------------------------------------------------|
| `scripts.setup`   |         | Command to run after creating a workspace                             |
| `scripts.run`     |         | Command to start the dev server                                       |
| `scripts.archive` |         | Command to run before removing a workspace                            |
| `port_range`      | `10`    | Number of consecutive ports per workspace                             |
| `base_port`       | `60000` | Starting port for allocation                                          |
| `worktree_path`   | `~/fr8` | Where to create worktrees (supports `~`, relative, or absolute paths) |

Falls back to `conductor.json` if `fr8.json` doesn't exist, so projects using [Conductor](https://conductor.build) work without changes.

Legacy camelCase keys (`portRange`, `basePort`, `worktreePath`) are still accepted but deprecated. Run `fr8 config doctor --fix` to migrate automatically.

Use `fr8 config show` to see the resolved configuration (with defaults applied) and `fr8 config doctor` to check for issues.

## How It Works

Each workspace is a git worktree with an allocated port range and injected environment variables. The lifecycle is:

1. **`fr8 ws new`** creates a git worktree, allocates a port block, syncs gitignored files (via `.worktreeinclude`), runs your setup script, then drops you into a subshell in the new workspace. Use `--no-shell` to skip the shell (useful for scripting). Use `-r`/`--remote` to track an existing remote branch, or `-p`/`--pull-request` to create a workspace from a GitHub PR (requires `gh` CLI).
2. **`fr8 ws run`** starts your run script in a background tmux session, freeing up your terminal.
3. **`fr8 ws archive`** auto-stops any running background session, runs your archive script (e.g. drop databases), removes the git worktree, and frees the port.

### Background Process Management

fr8 uses tmux to run workspaces in the background. This lets you start multiple workspaces without dedicating a terminal to each one.

```bash
# Start a workspace in the background
fr8 ws run my-feature

# See what's running
fr8 ws ps

# Check recent output (use -f to follow)
fr8 ws logs my-feature

# Attach for interactive debugging (detach with Ctrl-B d)
fr8 ws attach my-feature

# Stop when done
fr8 ws stop my-feature
```

Sessions are named `fr8/<repo>/<workspace>` (e.g. `fr8/myapp/bright-berlin`). The `fr8 ws list` and `fr8 ws status` commands show running state, and `fr8 ws archive` auto-stops sessions before tearing down.

The TUI dashboard (`fr8 dashboard`) provides a full interactive interface. Press `?` in the dashboard for a keybinding reference. Key highlights:

**Repo list:** `enter` to view workspaces, `r`/`x` to run/stop all in a repo, `R`/`X` for global run/stop across all repos.

**Workspace list:** `n` to create, `r` to run, `x` to stop, `t` to attach, `s` to shell, `o` to open, `b` to open browser, `a` to archive, `A` to batch-archive all merged+clean workspaces.

Requires tmux to be installed (`brew install tmux` / `apt install tmux`). All commands that use tmux gracefully degrade when it's not available.

### Workspace Openers

Configure external tools for opening workspaces directly from the TUI dashboard:

```bash
# Add openers (executable must be in $PATH)
fr8 opener add rubymine
fr8 opener add vscode code              # name differs from executable
fr8 opener add vscode-nw code --new-window  # command with arguments
fr8 opener add cursor

# Set a default opener (used when multiple are configured)
fr8 opener set-default vscode

# List configured openers
fr8 opener list

# Remove an opener
fr8 opener remove cursor
```

Open workspaces from the CLI with `fr8 ws open [name]` (auto-selects if one opener or a default is set, use `--opener <name>` to override). In the dashboard, press `o` on a workspace to open it. If you have one opener or a default configured, it's used directly. Otherwise, a picker lets you choose.

Opener configuration is stored at `~/.config/fr8/openers.json`.

### Environment Variables

fr8 sets these before running any script:

| Variable             | Example                                    |
|----------------------|--------------------------------------------|
| `FR8_WORKSPACE_NAME` | `bright-berlin`                            |
| `FR8_WORKSPACE_PATH` | `/Users/you/fr8/myapp/bright-berlin`       |
| `FR8_ROOT_PATH`      | `/Users/you/Code/myapp`                    |
| `FR8_DEFAULT_BRANCH` | `main`                                     |
| `FR8_PORT`           | `60000`                                    |

`CONDUCTOR_*` equivalents are also set for backwards compatibility with Conductor.

To load workspace environment variables into your current shell: `eval "$(fr8 ws env)"`.

### File Syncing

Create a `.worktreeinclude` file in your repo root listing gitignored files that should be copied to new worktrees:

```gitignore
# Environment files
.env*

# Credentials
config/master.key
config/credentials/*.key

# Local config
.mcp.json
```

Supports glob patterns including `**`. Files are only copied when their content differs.

### Port Allocation

Ports are allocated sequentially in blocks of `port_range` (default 10) starting from `base_port`. Each workspace gets exclusive use of its block. Your scripts can use the base port (`FR8_PORT`) and offset from it for additional services (e.g. Redis on `FR8_PORT + 1`).

When allocating ports, fr8 checks all registered repos (see `fr8 repo list`) to avoid conflicts across projects that share the same `base_port`. If the global registry is unavailable, allocation falls back to the current repo's ports only.

### State

Workspace state is stored in `.git/fr8.json` inside the repository's git directory. This is automatically shared across all worktrees.

## Shell Setup

Add a helper function to jump into workspaces:

```bash
# ~/.zshrc or ~/.bashrc
fr8cd() { cd "$(fr8 ws cd "$@")"; }
```

Enable completions:

```bash
# Bash
source <(fr8 completion bash)

# Zsh
fr8 completion zsh > "${fpath[1]}/_fr8"

# Fish
fr8 completion fish | source
```

## JSON Output

All commands support `--json` for structured machine-readable output. Add `--concise` for minimal fields (useful in pipelines).

```bash
# Structured workspace list
fr8 ws list --json

# Minimal output for scripting
fr8 ws list --json --concise

# Create a workspace without entering a shell
fr8 ws new my-feature -b feature/auth --json

# Check status as JSON
fr8 ws status my-feature --json
```

When `--json` is active:
- Stdout contains only the JSON result object (one per command)
- Human progress messages (e.g. "Fetching latest from origin...") are suppressed
- Errors are written to stderr as JSON: `{"error": "...", "code": "...", "exit_code": N}`
- Interactive commands (`attach`, `shell`, `exec`, `dashboard`) return an error with code `interactive_only`
- When stdout is not a TTY (even without `--json`), human messages are routed to stderr so piped stdout stays clean

### Exit Codes

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

### Idempotency Flags

These flags make commands safe to call repeatedly in scripts and automation:

| Flag               | Command      | Behavior                                     |
|--------------------|--------------|----------------------------------------------|
| `--if-not-exists`  | `ws new`     | Succeed silently if workspace already exists |
| `--if-exists`      | `ws archive` | Succeed silently if workspace not found      |
| `--if-not-running` | `ws run`     | Succeed silently if already running          |
| `--if-running`     | `ws stop`    | Succeed silently if not running              |
| `--dry-run`        | `ws new`     | Show what would be created without doing it  |
| `--dry-run`        | `ws archive` | Show what would be done without doing it     |

## MCP Server (AI Agent Integration)

fr8 includes a built-in [Model Context Protocol](https://modelcontextprotocol.io) server, allowing AI agents (Claude, Cursor, etc.) to manage workspaces programmatically.

### Setup

Run `fr8 mcp help` for setup instructions, or read on.

Add fr8 to your MCP client configuration. For Claude Code, add it to `.mcp.json` or via the CLI:

```bash
claude mcp add fr8 -- fr8 mcp serve
```

Or manually in your MCP config file (e.g. `.mcp.json`, `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "fr8": {
      "command": "fr8",
      "args": ["mcp", "serve"]
    }
  }
}
```

For Cursor, add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "fr8": {
      "command": "fr8",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Available Tools

The MCP server exposes 12 tools:

| Tool                 | Description                                                    |
|----------------------|----------------------------------------------------------------|
| `workspace_list`     | List workspaces (filter by repo, running, dirty, merged)       |
| `workspace_status`   | Get workspace details, env vars, process status, dirty state   |
| `workspace_create`   | Create a new workspace (branch, remote, PR, idempotent)        |
| `workspace_archive`  | Archive a workspace (force, idempotent)                        |
| `workspace_run`      | Start dev server in background tmux session                    |
| `workspace_stop`     | Stop a workspace's background session                          |
| `workspace_env`      | Get FR8_* environment variables for a workspace                |
| `workspace_logs`     | Get recent output from a background session                    |
| `workspace_rename`   | Rename a workspace                                             |
| `repo_list`          | List registered repos (optionally include workspace details)   |
| `config_show`        | Show resolved fr8 configuration for a repo                     |
| `config_doctor`      | Check fr8 configuration health and report errors/warnings      |

All tools accept an optional `repo` parameter to target a specific registered repo. The MCP server uses the global registry for workspace resolution (it does not auto-detect from CWD since it runs as a long-lived process).

## Agent Skills (CLI Integration)

fr8 can also be used by AI agents through direct CLI invocation instead of MCP. The `fr8 skill install` command generates a SKILL.md file that teaches agents how to manage workspaces using `fr8` commands with `--json` output.

Use CLI mode (skills) when your agent supports [Agent Skills](https://agentskills.io), or MCP mode when it supports the Model Context Protocol. Both expose the same operations.

### Install

```bash
# Claude Code (default — installs to ~/.claude/skills/fr8/)
fr8 skill install

# OpenAI Codex (installs to ~/.agents/skills/fr8/)
fr8 skill install --codex

# Project-scoped instead of global
fr8 skill install --claude --project
fr8 skill install --codex --project
```

Use `--name <name>` to customize the skill directory name (default: `fr8`), and `--force` to overwrite an existing installation. Run `fr8 skill --help` for the full explanation of CLI mode vs MCP.

## Example: Rails Project

A typical Rails setup script might handle dependencies, databases, and config:

```bash
#!/usr/bin/env bash
# bin/setup-workspace
set -e

REDIS_PORT=$((FR8_PORT + 1))

# Write workspace env file
cat > .env.workspace <<EOF
FR8_WORKSPACE_NAME=${FR8_WORKSPACE_NAME}
FR8_PORT=${FR8_PORT}
DB_PREFIX=${FR8_WORKSPACE_NAME}_
PORT=${FR8_PORT}
REDIS_URL=redis://localhost:${REDIS_PORT}
EOF

# Install dependencies
bundle install
npm install

# Create per-workspace databases
redis-server --port "$REDIS_PORT" --daemonize yes --save "" --appendonly no
bin/rails db:prepare
RAILS_ENV=test bin/rails db:prepare
redis-cli -p "$REDIS_PORT" shutdown nosave
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
