package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(mcpCmd)
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server for AI agent integration",
	Long: `MCP server for AI agent integration.

fr8 includes a built-in Model Context Protocol (MCP) server that lets AI agents
manage workspaces programmatically.

SETUP

  Claude Code (recommended):

    claude mcp add fr8 -- fr8 mcp serve

  Manual .mcp.json (Claude Code, Windsurf, etc.):

    {
      "mcpServers": {
        "fr8": {
          "command": "fr8",
          "args": ["mcp", "serve"]
        }
      }
    }

  Cursor (.cursor/mcp.json):

    {
      "mcpServers": {
        "fr8": {
          "command": "fr8",
          "args": ["mcp", "serve"]
        }
      }
    }

AVAILABLE TOOLS

  workspace_list      List workspaces (filter by repo, running, dirty, merged)
  workspace_status    Get workspace details, env vars, process status
  workspace_create    Create a new workspace (branch, remote, PR, idempotent)
  workspace_archive   Archive a workspace (force, idempotent)
  workspace_run       Start dev server in background tmux session
  workspace_stop      Stop a workspace's background session
  workspace_env       Get FR8_* environment variables for a workspace
  workspace_logs      Get recent output from a background session
  workspace_rename    Rename a workspace
  repo_list           List registered repos
  config_show         Show resolved fr8 configuration for a repo
  config_validate     Validate fr8 configuration and report errors/warnings

All tools accept an optional "repo" parameter to target a specific registered
repo. The server uses the global registry for workspace resolution.

See the README for full documentation.`,
}
