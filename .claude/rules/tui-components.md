# TUI Component Patterns

The dashboard TUI (`internal/tui/`) uses Bubble Tea with a single top-level model.

## File Organization

| File | Purpose |
|------|---------|
| `model.go` | Main model, `Init`, `Update`, `View`, key handlers, async commands |
| `messages.go` | View state enum, item types (`repoItem`, `workspaceItem`), message types |
| `keys.go` | Key bindings (`keyMap` struct, global `keys` var) |
| `styles.go` | Adaptive color palette, lipgloss styles |
| `panel.go` | Reusable panel rendering (`renderTitledPanel`, `renderBreadcrumb`, `renderHelpBar`) |
| `help.go` | Help overlay view |
| `repo_list.go` | Repo list view rendering |
| `workspace_list.go` | Workspace list view rendering |
| `opener_picker.go` | Opener picker view rendering |
| `create_workspace.go` | Create workspace text input view |

## Model Pattern

Single `model` struct holds all state. No sub-models except Bubble Tea components (`spinner.Model`, `textinput.Model`).

- `view` field (`viewState` enum) controls which view is rendered
- `previousView` preserves state for overlay navigation (help screen)
- `cursor` is shared across views and reset to 0 on view transitions
- `loading` flag shows spinner and blocks key input

## Message Pattern

All async operations return typed messages:

```go
type xxxResultMsg struct {
    name string  // identifier for the affected resource
    err  error   // nil on success
}
```

- Messages are defined in `messages.go`
- Result messages follow `xxxResultMsg` naming
- Load messages follow `xxxLoadedMsg` naming
- Request messages follow `xxxRequestMsg` naming (for deferred actions after `tea.Quit`)

## Key Handling

- Global keys (`Quit`, `Help`) handled first in `handleKey()`
- View-specific keys dispatched via switch on `m.view`
- Each view has its own `handleXxxKey(msg tea.KeyMsg)` method
- Key bindings defined in `keys.go` using `key.NewBinding()` with `WithKeys` and `WithHelp`

## View Rendering

- Each view has a `renderXxx(m model) string` function in its own file
- Views use `renderBreadcrumb()` for navigation context
- List views use `renderTitledPanel()` for bordered sections
- `renderHelpBar()` shows contextual key hints at the bottom
- `padToHeight()` ensures consistent terminal height

## Styles

- All colors use `lipgloss.AdaptiveColor{Light, Dark}` for theme support
- Style variables are package-level in `styles.go`
- Semantic naming: `statusCleanStyle`, `statusDirtyStyle`, `errorStyle`, `confirmStyle`
- `panelBorder()` returns a bordered style for panels

## Adding a New View

1. Add a `viewXxx` constant to the `viewState` enum in `messages.go`
2. Add any new message types to `messages.go`
3. Create `xxx.go` with `renderXxx(m model) string`
4. Add a `handleXxxKey` method to `model.go`
5. Add the view case to `View()` and `handleKey()` switch statements
6. Add key bindings to `keys.go` if needed
