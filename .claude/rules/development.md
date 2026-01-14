# Development Guidelines

## Code Style

- `go fmt` on save
- `golint` clean
- Errors are wrapped with context: `fmt.Errorf("store: failed to read: %w", err)`
- No magic - explicit is better than implicit

### CLI Architecture (PlanetScale Pattern)

Commands use the constructor pattern instead of `init()` functions:

```go
// Good: Constructor pattern
func newTaskCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "task",
        Short: "Manage tasks",
    }
    cmd.AddCommand(newListCmd())
    cmd.AddCommand(newAddCmd())
    return cmd
}

var Cmd = newTaskCmd()

func newAddCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "add [description]",
        RunE:  runAdd,
    }
    cmd.Flags().String("id", "", "Optional task ID")
    cmd.Flags().Int("priority", 2, "Task priority (0-4)")
    return cmd
}

// Bad: init() pattern (don't use)
var addCmd = &cobra.Command{...}
func init() {
    Cmd.AddCommand(addCmd)
    addCmd.Flags().String("id", "", "Optional task ID")
}
```

**Why constructors:**
- Explicit initialization order (predictable)
- No global side effects
- Easier to test commands in isolation
- Flags stay local to functions (no package-level vars)
- Industry standard (PlanetScale CLI, GitHub CLI)

## Development Workflow

### We dogfood tasuku while building it

1. Tasks are tracked in `.tasuku/` at repo root (V4 Markdown format)
2. Use `tk` commands or edit Markdown files directly
3. Every PR should update task status

### Branching

- `main` - stable, tested
- `feature/*` - new features
- `fix/*` - bug fixes

### Commits

Reference task IDs in commits:
```
feat: Add file locking to store (#store-locking)
fix: Handle empty task list (#empty-list-bug)
```

## Adding New Functionality Checklist

When adding new MCP tools, CLI commands, or features, follow this audit checklist:

### 1. TUI/CLI/MCP/Plugin Parity
- [ ] New CLI command → Add corresponding MCP tool
- [ ] New MCP tool → Consider CLI equivalent
- [ ] Core operation → Add to TUI if visual interaction helps
- [ ] Frequent operation → Consider adding a Skill (slash command)
- [ ] Same capabilities, same behavior across all interfaces

### 2. Tool Descriptions (Nudges)
Every MCP tool description should include:
- **WHAT it does** (basic description)
- **WHEN to use it** ("Use PROACTIVELY when...")
- **Examples** of trigger scenarios (numbered list)
- **Follow-up hints** (what to do after using)

### 3. Response Enhancements
Consider adding smart responses that:
- **Warn** about potential issues (e.g., multiple in_progress tasks)
- **Suggest** next actions (e.g., "Consider archiving" after tk_done)
- **List** affected items (e.g., "These tasks are now unblocked")
- **Prompt** for follow-up (e.g., "Add a note explaining why paused")

### 4. Hook Integration
Check if the new feature should trigger or be triggered by:
- **SessionStart**: Should it be included in session context?
- **Stop**: Should it be reminded about at session end?
- **PostToolUse**: Should it prompt for related actions?

### 5. Documentation
- [ ] Update CLAUDE.md if it affects agent workflow
- [ ] Update README.md MCP tools table
- [ ] Add to CLI help text

### 6. Plugin Commands (Optional)
If the feature is frequently used, consider adding a skill (slash command).

## Mandatory Learning Documentation

**CRITICAL: Document learnings IMMEDIATELY when they occur, not at session end.**

### When to Record Learnings (MANDATORY)

Record a learning using `tk_learn` or `tk learn` **IMMEDIATELY** after ANY of these events:

1. **Bug Fix Completed**: After fixing ANY bug, record:
   - What the bug was
   - Why it happened (root cause)
   - The rule to prevent it ("Never X" or "Always Y")

2. **Gotcha Discovered**: When you discover unexpected behavior:
   - API that doesn't work as documented
   - Edge case that causes failures
   - Implicit assumptions in code

3. **Pattern Identified**: When you notice a recurring issue:
   - Same type of bug appearing multiple times
   - Code smell that leads to problems
   - Anti-pattern in the codebase

4. **Workaround Required**: When standard approach doesn't work:
   - Library limitation requiring different approach
   - Framework quirk needing special handling

### Learning Format

Use "Never" or "Always" prefixes for rules that should prevent future bugs:

```bash
# Good - actionable rule
tk learn "Never manually manipulate ANSI-styled strings with rune operations. Use lipgloss.Width/Height/Place which handle escape sequences correctly."

# Good - specific gotcha
tk learn "Always collect map keys into a slice before iterating if you'll modify the map during iteration."

# Bad - too vague
tk learn "Be careful with strings"
```

## Agent Task Management

When working on this codebase, follow these guidelines for task tracking:

### When to use `tk` (Tasuku)
- New features or enhancements (e.g., "Add dark mode support")
- Bug reports (e.g., "Fix race condition in auth")
- Project milestones (e.g., "V3 migration")
- Tasks that should persist across sessions
- Tasks that other agents might need to see

### When to use TodoWrite only
- Implementation steps within a session (e.g., "Update file X", "Fix type error in Y")
- Temporary tracking of sub-steps
- Progress tracking that doesn't need to persist

### Nudge Rule

**Before adding items to TodoWrite**, use `tk_suggest` (MCP) or `tk suggest` (CLI) to check if it should also be tracked in tk.
