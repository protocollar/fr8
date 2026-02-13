# MCP Server Development

The fr8 MCP server exposes workspace management tools to AI agents via the Model Context Protocol.

## Architecture

- `internal/mcp/server.go` — Server creation and stdio transport
- `cmd/mcp_tools.go` — Tool definitions and handler implementations
- `cmd/mcp_serve.go` — `fr8 mcp serve` command entry point

Tools are registered in `registerMCPTools()` in `cmd/mcp_tools.go`. The server itself (`internal/mcp/`) is kept thin — tool handlers live in `cmd/` because they reuse CLI logic like `createWorkspace()` and `resolveWorkspace()`.

## Adding a New Tool

1. Define the tool in `registerMCPTools()` using `mcp.NewTool()`:
   - Tool names use `snake_case` with domain prefix: `workspace_list`, `repo_list`, `config_show`
   - Always set `mcp.WithDescription()` — this is what agents see
   - Mark read-only tools with `mcp.WithReadOnlyHintAnnotation(true)`
   - Mark destructive tools with `mcp.WithDestructiveHintAnnotation(true)`
   - Use `mcp.Required()` on parameters that are not optional
   - Parameter names use `snake_case`: `old_name`, `if_not_exists`

2. Write a handler with signature `func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)`:
   - Extract params with `req.GetString()`, `req.GetBool()`, `req.GetInt()`
   - Return success via `mcpResult(v)` — marshals to JSON text content
   - Return errors via `mcpError(msg)` — sets `IsError: true`

3. Update `TestRegisterMCPTools` in `cmd/mcp_tools_test.go` to include the new tool name

## Helpers

- `mcpResult(v any)` — marshals any value as indented JSON, returns as MCP text content
- `mcpError(msg string)` — returns an MCP error result
- `mcpResolveWorkspace(name, repo)` — resolves workspace from global registry (never CWD)
- `mcpResolveRepo(repo)` — resolves repo root path and git common dir

## Key Differences from CLI

- MCP tools never detect from CWD — they always use the global registry for lookup
- MCP tools don't call `os.Exit()` — they return `mcpError()`
- MCP tools don't use `jsonout` — they return structured JSON directly via `mcpResult()`
- Idempotent flags (`if_not_exists`, `if_running`, `if_not_running`) return success with an action describing what happened

## Testing

- Test helpers (`mcpResult`, `mcpError`, `mcpResolveWorkspace`) with unit tests
- Test tool registration with `TestRegisterMCPTools` — verifies all expected tools are present
- Use `initTestRepo(t)` helper for integration tests needing a real git repo
