---
description: "Promote a learning to permanent documentation"
argument-hint: "LEARNING_ID [--to FILE] [--keep]"
---

# Promote Learning

```!
tk learning promote $ARGUMENTS
```

Move a learning from Tasuku's context to permanent project documentation.

## Usage

```bash
tk learnings                        # List learnings with indices
tk promote <index>                  # Promote to auto-detected context file
tk promote <index> --to CLAUDE.md   # Promote to specific file
tk promote <index> --keep           # Keep in learnings after promoting
```

## Auto-Detected Context Files

Tasuku looks for these files (in priority order):
1. `CLAUDE.md` - Claude Code
2. `.cursorrules` - Cursor
3. `.github/copilot-instructions.md` - GitHub Copilot
4. `AGENTS.md` - Generic AI agents

If none exist, defaults to creating `CLAUDE.md`.

## When to Promote

Promote learnings when they:
- Have proven valuable multiple times
- Are "never/always" rules worth enforcing
- Should persist beyond the current task
- Would help future developers or agents

## What Gets Promoted

- The learning text is appended to the context file
- Adds under a "## Learnings" section if not present
- Removes from Tasuku learnings unless `--keep` is used

## Best Practices

1. Let learnings prove their value before promoting
2. Use `--to` to choose the appropriate file
3. Review promoted content for clarity
4. Periodically audit context files for outdated learnings
