# Tasuku Learnings

_Auto-synced from .tasuku/context/learnings.md_

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
- Charmbracelet ecosystem essentials for TUIs: (1) lipgloss for styling - use Place() for centering modals on ANSI-aware backgrounds; (2) bubbles/list for filterable lists with custom delegates; (3) bubbles/progress for progress bars; (4) bubbles/textarea for multi-line input; (5) bubbles/key for keybinding definitions; (6) glamour for Markdown rendering in views; (7) charmbracelet/x/ansi.Truncate for ANSI-aware string truncation (handles wide chars, graphemes); (8) x/exp/teatest for golden file testing.
- BubbleTea async I/O pattern: (1) Define message types for results (TasksLoadedMsg, ActionResultMsg); (2) Create command constructors that wrap I/O in tea.Cmd closures; (3) New() starts with empty state and loading=true; (4) Init() returns the load command; (5) Update() handles result messages and chains commands (action → reload); (6) In tests, process Init() command before checking state: cmd := m.Init(); msg := cmd(); m.Update(msg).
- Claude Code hooks (UserPromptSubmit, PostToolUse, etc.) can only analyze user messages and tool calls, not agent responses. For agent self-awareness (detecting "got it", "you're right", "turns out"), embed guidance directly in MCP tool descriptions under "AGENT SELF-AWARENESS" sections rather than trying to hook agent responses.

