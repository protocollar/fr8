#!/bin/bash
# Stop hook: remind about staged changes that haven't been committed.

cd "$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

if ! git diff --cached --quiet 2>/dev/null; then
  echo "Reminder: staged changes detected. Verify go test -race ./... passes before committing."
fi
