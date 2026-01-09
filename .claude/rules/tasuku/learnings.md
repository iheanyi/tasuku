# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

## Rules

- Always force UTC timezone in golden tests (os.Setenv("TZ", "UTC") in init()) when testing time-dependent output. Local timezone varies between CI and developer machines, causing spurious test failures.
- Never manually manipulate ANSI-styled strings with rune/character operations. ANSI escape codes (e.g., \x1b[31m) are counted as characters, corrupting position calculations. Always use lipgloss.Place(), lipgloss.Width(), lipgloss.Height() which properly handle escape sequences.
- Never iterate over a map while modifying it (e.g., archiving tasks while looping over m.file.Tasks). Always collect IDs/keys first into a slice, then iterate over the slice to perform modifications.
- In BubbleTea TUIs, always save both the selected item ID AND index before refresh/reinitializing lists. After refresh, restore by ID first (item may have moved), fall back to index position (item may be deleted), or clamp to list bounds.
- Always use `tk list` to verify task IDs before calling `tk done`, as the ID might differ from the description slug or be truncated.

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
- For terminal UI overlays/modals, use lipgloss.Place() to center content on a clean screen rather than trying to composite foreground over background. Manual overlay compositing breaks with styled text.
- syscall.Flock (file locking) is Unix-only. Windows builds will fail if using it directly. Either use build tags with platform-specific implementations or exclude Windows from build targets.

