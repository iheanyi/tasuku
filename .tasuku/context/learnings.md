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

## ebc542 - 2026-01-07T18:49:25Z
For terminal UI overlays/modals, use lipgloss.Place() to center content on a clean screen rather than trying to composite foreground over background. Manual overlay compositing breaks with styled text.

## 2543ae - 2026-01-08T01:12:45Z
syscall.Flock (file locking) is Unix-only. Windows builds will fail if using it directly. Either use build tags with platform-specific implementations or exclude Windows from build targets.

## 1f9d2a - 2026-01-09T15:50:20Z
Charmbracelet ecosystem essentials for TUIs: (1) lipgloss for styling - use Place() for centering modals on ANSI-aware backgrounds; (2) bubbles/list for filterable lists with custom delegates; (3) bubbles/progress for progress bars; (4) bubbles/textarea for multi-line input; (5) bubbles/key for keybinding definitions; (6) glamour for Markdown rendering in views; (7) charmbracelet/x/ansi.Truncate for ANSI-aware string truncation (handles wide chars, graphemes); (8) x/exp/teatest for golden file testing.

## a64c8c - 2026-01-09T15:50:27Z
BubbleTea async I/O pattern: (1) Define message types for results (TasksLoadedMsg, ActionResultMsg); (2) Create command constructors that wrap I/O in tea.Cmd closures; (3) New() starts with empty state and loading=true; (4) Init() returns the load command; (5) Update() handles result messages and chains commands (action → reload); (6) In tests, process Init() command before checking state: cmd := m.Init(); msg := cmd(); m.Update(msg).

## 30b116 - 2026-01-15T22:48:33Z
Claude Code hooks (UserPromptSubmit, PostToolUse, etc.) can only analyze user messages and tool calls, not agent responses. For agent self-awareness (detecting "got it", "you're right", "turns out"), embed guidance directly in MCP tool descriptions under "AGENT SELF-AWARENESS" sections rather than trying to hook agent responses.

## 9fd1b2 - 2026-01-20T17:19:32Z
Cursor commands use plain Markdown files in .cursor/commands/ without YAML frontmatter. Format: H1 title, description paragraph, then ## Instructions section. Different from Claude Code (YAML frontmatter with description/argument-hint) and Copilot/Codex (SKILL.md with name/description frontmatter).

