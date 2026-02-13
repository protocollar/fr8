# Go Code Style

## Naming

- **Receivers**: 1-2 letter abbreviation of the type, consistent across all methods (`s` for `*State`, `r` for `*Registry`)
- **Functions**: name describes return value, not implementation. Omit input/output types from the name
- **Acronyms**: consistent casing — `ID` not `Id`, `URL` not `Url`. Unexported: `userID`, `httpClient`
- **Short scopes**: single-letter vars are fine (`i`, `n`, `f`). Longer-lived vars get descriptive names
- **No stutter**: `config.Config` is fine, `config.ConfigManager` is not. If pkg name == type name, use `New()` not `NewConfig()`

## Struct Tags

- JSON tags use `snake_case`: `json:"created_at"`, `json:"workspace_name"`, `json:"exit_code"`

## Design

- Accept interfaces, return concrete types
- Define interfaces at the consumer site, not the implementation site
- Avoid premature interfaces — only when 2+ implementations exist or needed for testing
- Zero value should be useful where possible (nil slices work as empty)
- Use `os.UserConfigDir()` / `os.UserCacheDir()` for platform-appropriate paths

## Error Messages

- Lowercase, no punctuation at end
- No "failed to" prefix — it's noise that compounds as errors wrap
- Include relevant context variables: `fmt.Errorf("reading config %s: %w", path, err)`
- See `error-handling.md` for full error patterns
