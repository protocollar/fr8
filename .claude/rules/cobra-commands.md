# Cobra Command Conventions

## File Organization

- One file per command in `cmd/`
- Package-level `var` for the command struct
- Standalone `func runXxx(cmd *cobra.Command, args []string) error` for implementation logic
- Register commands via `init()` with `parentCmd.AddCommand(childCmd)`

## Command Definition

- **Always use `RunE`** — never `Run`. Errors propagate to central handler in `root.go`
- **Never call `os.Exit()`** in command implementations — return errors instead
- Use `syscall.Exec` for process-replacing commands (`shell`, `attach`, `run`, `exec`)

## Flags

- Define flags in `init()`, after command definition
- Use `MarkFlagsMutuallyExclusive()` for conflicting flags (see `new.go` for example)
- Persistent flags on parent commands for shared concerns (`--json`, `--concise` on root)

## Argument Validation

- Use `cobra.MaximumNArgs(n)`, `cobra.ExactArgs(n)`, `cobra.NoArgs` for arg validation
- Add `ValidArgsFunction` for shell completion on workspace/repo names

## Root Command

- `SilenceUsage: true` and `SilenceErrors: true` are set on root — never change these
- Central error handling in `Execute()` handles exit codes and JSON error output

## Example Pattern

```go
var exampleCmd = &cobra.Command{
    Use:   "example [name]",
    Short: "One-line description",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runExample,
}

func init() {
    exampleCmd.Flags().BoolVar(&someFlag, "flag", false, "description")
    parentCmd.AddCommand(exampleCmd)
}

func runExample(cmd *cobra.Command, args []string) error {
    // implementation — return errors, don't os.Exit()
}
```
