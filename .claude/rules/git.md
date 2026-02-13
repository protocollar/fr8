# Git Etiquette

## Commit Messages

- Always create atomic commits that do not leave the app in a broken state

## Conventional Commits Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## Commit Types

| Label       | Purpose                                                          |
|-------------|------------------------------------------------------------------|
| `feat:`     | New feature for the user (MINOR version bump)                    |
| `fix:`      | Bug fix for the user (PATCH version bump)                        |
| `docs:`     | Documentation changes                                            |
| `style:`    | Formatting, missing semicolons, etc. (no production code change) |
| `refactor:` | Refactoring production code (e.g., renaming a variable)          |
| `perf:`     | Performance improvements                                         |
| `test:`     | Adding/refactoring tests (no production code change)             |
| `build:`    | Build system or external dependency changes                      |
| `ci:`       | CI configuration changes                                         |
| `chore:`    | Other tasks (no production code change)                          |

## Scope (Optional)

Use scope to specify what area of the codebase is affected:

```bash
feat(conversations): add search filtering
fix(parser): handle missing message timestamps
refactor(models): extract sourceable concern
```

## Breaking Changes

Indicate breaking changes using either method:

```bash
# Method 1: Add ! before the colon
feat!: change API response format
feat(api)!: rename endpoints for v2

# Method 2: BREAKING CHANGE footer
feat: change API response format

BREAKING CHANGE: response now returns array instead of object
```

## Multi-line Commits

For complex changes, add a body and/or footer:

```bash
fix(parser): handle conversations with empty messages

The parser was failing when a conversation contained messages
with nil content. Added guard clause to skip empty messages.

Fixes #123
```

## Examples

```bash
feat: add conversation search
feat(projects): add folder grouping support
fix: resolve nil error in message parsing
fix(parser)!: change conversation data structure
docs: update development setup guide
style: fix indentation in conversations controller
refactor: extract sourceable concern from models
perf: optimize conversation list queries
test: add tests for message model
build: upgrade Rails to 8.1
ci: add parallel test runners
chore: update rubocop configuration
```