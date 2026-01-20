---
status: done
priority: 2
tags: [feature, skills, abstraction]
created_at: 2026-01-20T16:56:02.920279Z
updated_at: 2026-01-20T17:13:37.756284Z
---

# - Claude Code: plugins with `commands/*.md`

- Copilot CLI: skills in `.github/skills/` or `~/.copilot/skills/`
- Codex: skills in `.codex/skills/` or `~/.codex/skills/`
- Cursor: skills format TBD

Tasks:
- Define canonical Tasuku skill format (SKILL.md)
- Create converter for Claude Code plugin format
- Implement `tk plugin install --tool <name>`
- Support both global and project-local installation
