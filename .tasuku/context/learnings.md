# Learnings

## e254f9 - 2026-01-05T00:00:00Z
CLI command is 'tk' to avoid collisions

## 4b7d8d - 2026-01-05T00:00:00Z
Using flock for file locking (works on macOS/Linux)

## 2b245a - 2026-01-05T00:00:00Z
JSON chosen over YAML for faster parsing and less ambiguity

## 8d6dde - 2026-01-05T00:00:00Z
MCP server will use stdio mode for Claude Code integration

## 4e75ce - 2026-01-05T00:00:00Z
taskResult type in MCP server is local - use JSON marshal/unmarshal for testing

## 43de6c - 2026-01-05T00:00:00Z
When adding items to TodoWrite that look like project-level tasks (features, bugs, milestones), also add them to tk to ensure persistence across sessions

## 9b1e40 - 2026-01-05T00:00:00Z
JSON Schema in 'tk context schema' is outdated - missing V3 fields: parent_id, claimed_at, tags, fields, timer_start, duration, notes (now on task not context). Learnings should be objects with {id, text, is_rule, created_at}, not strings.

## bfe274 - 2026-01-05T00:00:00Z
HTTP server has no handling for stale processes or port conflicts. Consider PID file or port check on startup. For dev hot-reload, use 'air' or 'watchexec'.

## aebb8c - 2026-01-05T00:00:00Z
When updating storage formats, check ALL user-facing strings: ErrNotInitialized, help text, error messages, hook scripts. Shared errors can leak format-specific messages.

## 37aa8f - 2026-01-06T00:00:00Z
Always force UTC timezone in golden tests (os.Setenv("TZ", "UTC") in init()) when testing time-dependent output. Local timezone varies between CI and developer machines, causing spurious test failures.

## be327a - 2026-01-07T18:49:04Z
Never manually manipulate ANSI-styled strings with rune/character operations. ANSI escape codes (e.g., \x1b[31m) are counted as characters, corrupting position calculations. Always use lipgloss.Place(), lipgloss.Width(), lipgloss.Height() which properly handle escape sequences.

## 788070 - 2026-01-07T18:49:12Z
Never iterate over a map while modifying it (e.g., archiving tasks while looping over m.file.Tasks). Always collect IDs/keys first into a slice, then iterate over the slice to perform modifications.

## a4a226 - 2026-01-07T18:49:18Z
In BubbleTea TUIs, always save both the selected item ID AND index before refresh/reinitializing lists. After refresh, restore by ID first (item may have moved), fall back to index position (item may be deleted), or clamp to list bounds.

## ebc542 - 2026-01-07T18:49:25Z
For terminal UI overlays/modals, use lipgloss.Place() to center content on a clean screen rather than trying to composite foreground over background. Manual overlay compositing breaks with styled text.

## 68d5cd - 2026-01-07T23:45:30Z
scope: *.md
Always audit CLI command documentation (README.md, CLAUDE.md) when adding new commands or changing command signatures. Shortcut commands like `tk learn` should be documented alongside their full forms (`tk learning add`).

## 2543ae - 2026-01-08T01:12:45Z
syscall.Flock (file locking) is Unix-only. Windows builds will fail if using it directly. Either use build tags with platform-specific implementations or exclude Windows from build targets.

## f11dfa - 2026-01-09T03:36:44Z
Always use `tk list` to verify task IDs before calling `tk done`, as the ID might differ from the description slug or be truncated.

