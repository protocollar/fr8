# fr8

Git worktree workspace manager. Creates isolated development environments with per-workspace port ranges, database prefixes, and environment variables.

## Install

```bash
go install github.com/thomascarr/fr8@latest
```

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

| Command                               | Description                                              |
|---------------------------------------|----------------------------------------------------------|
| `fr8 ws new [name] [-b branch]`       | Create a workspace (worktree + port + file sync + setup) |
| `fr8 ws list`                         | List all workspaces                                      |
| `fr8 ws status [name]`                | Show workspace details and environment variables         |
| `fr8 ws run [name] [-A/--all]`        | Run the dev server in a background tmux session          |
| `fr8 ws stop [name] [-A/--all]`       | Stop a workspace's background tmux session               |
| `fr8 ws attach [name]`                | Attach to a running background session                   |
| `fr8 ws logs [name] [-n lines]`       | Show recent output from a background session             |
| `fr8 ws ps`                           | List all running fr8 workspace sessions                  |
| `fr8 ws exec [name] -- <cmd>`         | Run a command with workspace environment                 |
| `fr8 ws shell [name]`                 | Open a subshell with workspace environment               |
| `fr8 ws cd [name]`                    | Print workspace path                                     |
| `fr8 ws browser [name]`               | Open workspace dev server in the browser                 |
| `fr8 ws archive [name]`               | Tear down workspace (archive script + remove worktree)   |
| `fr8 dashboard`                       | Interactive TUI for browsing repos and workspaces        |
| `fr8 repo add\|list\|remove`          | Manage the global repo registry                          |
| `fr8 completion [bash\|zsh\|fish]`    | Generate shell completions                               |

When `[name]` is omitted, fr8 auto-detects the current workspace from your working directory. When a name is provided, it also works from outside a git repo by searching all registered repos.

## Configuration

Create `fr8.json` in your repo root:

```json
{
  "scripts": {
    "setup": "bin/setup-workspace",
    "run": "bin/run-workspace",
    "archive": "bin/archive-workspace"
  },
  "portRange": 10,
  "basePort": 5000,
  "worktreePath": "../fr8"
}
```

| Field             | Default        | Description                                       |
|-------------------|----------------|---------------------------------------------------|
| `scripts.setup`   |                | Command to run after creating a workspace         |
| `scripts.run`     |                | Command to start the dev server                   |
| `scripts.archive` |                | Command to run before removing a workspace        |
| `portRange`       | `10`           | Number of consecutive ports per workspace         |
| `basePort`        | `5000`         | Starting port for allocation                      |
| `worktreePath`    | `../fr8`       | Where to create worktrees (relative to repo root) |

Falls back to `conductor.json` if `fr8.json` doesn't exist, so projects using [Conductor](https://conductor.build) work without changes.

## How It Works

Each workspace is a git worktree with an allocated port range and injected environment variables. The lifecycle is:

1. **`fr8 ws new`** creates a git worktree, allocates a port block, syncs gitignored files (via `.worktreeinclude`), then runs your setup script.
2. **`fr8 ws run`** starts your run script in a background tmux session, freeing up your terminal.
3. **`fr8 ws archive`** auto-stops any running background session, runs your archive script (e.g. drop databases), removes the git worktree, and frees the port.

### Background Process Management

fr8 uses tmux to run workspaces in the background. This lets you start multiple workspaces without dedicating a terminal to each one.

```bash
# Start a workspace in the background
fr8 ws run my-feature

# See what's running
fr8 ws ps

# Check recent output
fr8 ws logs my-feature

# Attach for interactive debugging (detach with Ctrl-B d)
fr8 ws attach my-feature

# Stop when done
fr8 ws stop my-feature
```

Sessions are named `fr8/<repo>/<workspace>` (e.g. `fr8/myapp/bright-berlin`). The `fr8 ws list` and `fr8 ws status` commands show running state, and `fr8 ws archive` auto-stops sessions before tearing down.

The TUI dashboard (`fr8 dashboard`) also supports background management: `r` to run, `x` to stop, `t` to attach.

Requires tmux to be installed (`brew install tmux` / `apt install tmux`). All commands that use tmux gracefully degrade when it's not available.

### Environment Variables

fr8 sets these before running any script:

| Variable             | Example                                    |
|----------------------|--------------------------------------------|
| `FR8_WORKSPACE_NAME` | `bright-berlin`                            |
| `FR8_WORKSPACE_PATH` | `/Users/you/fr8/myapp/bright-berlin`       |
| `FR8_ROOT_PATH`      | `/Users/you/Code/myapp`                    |
| `FR8_DEFAULT_BRANCH` | `main`                                     |
| `FR8_PORT`           | `5000`                                     |

`CONDUCTOR_*` equivalents are also set for backwards compatibility with Conductor.

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

Ports are allocated sequentially in blocks of `portRange` (default 10) starting from `basePort`. Each workspace gets exclusive use of its block. Your scripts can use the base port (`FR8_PORT`) and offset from it for additional services (e.g. Redis on `FR8_PORT + 1`).

When allocating ports, fr8 checks all registered repos (see `fr8 repo list`) to avoid conflicts across projects that share the same `basePort`. If the global registry is unavailable, allocation falls back to the current repo's ports only.

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

## License

MIT
