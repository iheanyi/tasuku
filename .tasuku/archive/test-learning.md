---
status: done
priority: 3
tags: [testing, coverage]
created_at: 2026-01-14T22:30:56.413553Z
updated_at: 2026-01-15T01:54:21.371482Z
---

# Add tests for internal/cmd/learning package (currently 58.5% coverage)

## Summary
Coverage improved from 59.4% to 80.3% (+20.9 pp)

## Notes

### 2026-01-15T01:54:21Z [bd318e]
Coverage improved from 59.4% to 80.3% (+20.9 pp). Added tests for: runPromote (by ID, by text, with keep flag, not found), formatAge (all time ranges), detectContextFile (CLAUDE.md, GEMINI.md, .cursorrules, default), appendToContextFile (existing section, new section, new file).

