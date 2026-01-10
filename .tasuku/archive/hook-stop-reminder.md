---
status: done
priority: 1
created_at: 2026-01-06T04:22:13.256412Z
updated_at: 2026-01-06T04:27:52.818928Z
---

# Add Stop hook to remind about running timers and in-progress tasks


## Notes

### 2026-01-06T04:22:00Z [6f232e]
Hook should check for: (1) Running timers - warn to stop, (2) In-progress tasks - prompt to pause with notes, (3) Unclaimed tasks that were being worked on. Should NOT block, just remind.

