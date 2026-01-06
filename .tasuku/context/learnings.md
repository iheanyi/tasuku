# Learnings

## e254f9 - 2026-01-05
CLI command is 'tk' to avoid collisions

## 4b7d8d - 2026-01-05
Using flock for file locking (works on macOS/Linux)

## 2b245a - 2026-01-05
JSON chosen over YAML for faster parsing and less ambiguity

## 8d6dde - 2026-01-05
MCP server will use stdio mode for Claude Code integration

## 4e75ce - 2026-01-05
taskResult type in MCP server is local - use JSON marshal/unmarshal for testing

## 43de6c - 2026-01-05
When adding items to TodoWrite that look like project-level tasks (features, bugs, milestones), also add them to tk to ensure persistence across sessions

## 9b1e40 - 2026-01-05
JSON Schema in 'tk context schema' is outdated - missing V3 fields: parent_id, claimed_at, tags, fields, timer_start, duration, notes (now on task not context). Learnings should be objects with {id, text, is_rule, created_at}, not strings.

## bfe274 - 2026-01-05
HTTP server has no handling for stale processes or port conflicts. Consider PID file or port check on startup. For dev hot-reload, use 'air' or 'watchexec'.

## aebb8c - 2026-01-05
When updating storage formats, check ALL user-facing strings: ErrNotInitialized, help text, error messages, hook scripts. Shared errors can leak format-specific messages.

## 37aa8f - 2026-01-06
Always force UTC timezone in golden tests (os.Setenv("TZ", "UTC") in init()) when testing time-dependent output. Local timezone varies between CI and developer machines, causing spurious test failures.

