---
status: done
tags: [hooks, ux]
time_spent: 202202899000
created_at: 2026-02-21T17:05:20.671991Z
updated_at: 2026-02-21T17:13:29.571443Z
---

# Improve hook error visibility in Claude Code sessions


## Notes

### 2026-02-21T17:05:35Z [aeb44e]
Problem: When 'tk hooks session' fails (e.g. legacy V3 format, outdated hooks), Claude Code startup banner shows 'SessionStart:startup hook error' but the system context passed to the LLM says success. The AI agent literally cannot see the failure. Suggestions: (1) Output actionable fix on stdout before exiting non-zero. (2) Version mismatch should warn not fail - still run the hook. (3) Consider project-local hooks as default instead of global. (4) Separate hard errors from soft errors - soft errors should degrade gracefully.

