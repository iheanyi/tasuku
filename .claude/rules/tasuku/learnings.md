# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

## Rules

- Always force UTC timezone in golden tests (os.Setenv("TZ", "UTC") in init()) when testing time-dependent output. Local timezone varies between CI and developer machines, causing spurious test failures.

## Insights

- CLI command is 'tk' to avoid collisions
- Using flock for file locking (works on macOS/Linux)
- JSON chosen over YAML for faster parsing and less ambiguity
- MCP server will use stdio mode for Claude Code integration
- taskResult type in MCP server is local - use JSON marshal/unmarshal for testing
- When adding items to TodoWrite that look like project-level tasks (features, bugs, milestones), also add them to tk to ensure persistence across sessions
- JSON Schema in 'tk context schema' is outdated - missing V3 fields: parent_id, claimed_at, tags, fields, timer_start, duration, notes (now on task not context). Learnings should be objects with {id, text, is_rule, created_at}, not strings.
- HTTP server has no handling for stale processes or port conflicts. Consider PID file or port check on startup. For dev hot-reload, use 'air' or 'watchexec'.
- When updating storage formats, check ALL user-facing strings: ErrNotInitialized, help text, error messages, hook scripts. Shared errors can leak format-specific messages.

