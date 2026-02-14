# Error Handling Patterns

## Wrapping Errors

- Always use `%w` (not `%v`) unless intentionally hiding internals
- Keep context succinct: `"reading config: %w"` not `"failed to read config file: %w"`
- Include relevant local variables: `fmt.Errorf("reading config %s: %w", path, err)`

## Sentinel Errors

- Use `var ErrXxx = errors.New(...)` sparingly — only when callers need `errors.Is` branching
- Most errors just need wrapping with `fmt.Errorf`

## Exit Codes

Defined in `internal/exitcode/exitcode.go`:

| Code | Constant          | Meaning                                           |
|------|-------------------|---------------------------------------------------|
| 0    | `Success`         | OK                                                |
| 1    | `GeneralError`    | General failure                                   |
| 2    | `NotFound`        | Workspace/repo/opener not found                   |
| 3    | `AlreadyExists`   | Resource already exists                           |
| 4    | `NotInRepo`       | Not inside a git repository                       |
| 5    | `DirtyWorkspace`  | Uncommitted changes                               |
| 6    | `InteractiveOnly` | Feature requires TTY (incompatible with `--json`) |
| 7    | `TmuxUnavailable` | tmux not installed                                |
| 8    | `ConfigError`     | Configuration error                               |

## ExitError

Use `exitcode.ExitError` for errors that need specific exit codes:

```go
return exitcode.New("workspace_not_found", exitcode.NotFound, "workspace not found: "+name)
return exitcode.Wrap("not_in_repo", exitcode.NotInRepo, err)
```

## Error Classification

- `exitcode.ClassifyError()` maps error strings to codes for untyped errors
- Add new patterns there when introducing new error categories
- Central handling in `cmd/root.go` `Execute()` — never handle exit codes in individual commands

## JSON Mode Errors

- Errors output via `jsonout.WriteError(code, message, exitCode)`
- Always include a machine-readable `code` string (e.g. `"workspace_not_found"`)
