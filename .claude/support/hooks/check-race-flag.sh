#!/bin/bash
# PreToolUse hook: warn when `go test` is run without -race flag.
# CI requires -race, so catch this early.

CMD="$CLAUDE_TOOL_INPUT"
if echo "$CMD" | grep -qE 'go test[^|]*\./' && ! echo "$CMD" | grep -q '\-race'; then
  echo "Consider adding -race flag — CI requires it (go test -race ./...)."
fi
