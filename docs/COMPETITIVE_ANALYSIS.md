# Competitive Analysis: Tasuku vs Beads vs Dots

## Executive Summary

Three agent-first task management tools have emerged for AI coding assistants:

| Tool | Author | Language | Storage | Philosophy |
|------|--------|----------|---------|------------|
| **Tasuku** | iheanyi | Go | JSON file | Pull-based, minimal context |
| **Beads** | Steve Yegge | TypeScript | SQLite + JSONL + Git | Comprehensive memory system |
| **Dots** | Joel Reymont | Zig | SQLite (beads-compatible) | Minimal beads replacement |

---

## Detailed Comparison

### Architecture

#### Tasuku
```
.tasuku.json (single file)
├── version: 1
├── tasks: { id -> Task }
└── context: { learnings, decisions, notes }
```

- **Single JSON file** - Human-readable, git-friendly, trivially editable
- **File locking** via `flock` - Safe for parallel agents
- **MCP server** - Native Claude Code integration via stdio
- **No daemon** - Stateless CLI, fast startup

#### Beads
```
.beads/
├── beads.db (SQLite)
├── issues.jsonl (Git-synced log)
└── cache/
```

- **SQLite database** for local queries
- **JSONL files** for git collaboration
- **Background daemon** syncs DB ↔ JSONL
- **Two-way sync** complexity

#### Dots
```
.beads/
├── beads.db (SQLite, compatible with beads)
└── todo-mapping.json (TodoWrite sync)
```

- **SQLite only** - No JSONL, no daemon
- **Beads-compatible** database schema
- **Hook-based** Claude Code integration
- **Statically linked** - Zero runtime dependencies

---

### Feature Matrix

| Feature | Tasuku | Beads | Dots |
|---------|--------|-------|------|
| **Binary size** | ~8 MB | ~19 MB | ~0.9 MB |
| **Startup time** | ~5 ms | ~7 ms | ~3 ms |
| **Background daemon** | No | Yes | No |
| **File locking** | Yes (flock) | Via daemon | No |
| **Git sync** | Manual | Automatic | Manual |
| **MCP server** | Built-in | Plugin | No |
| **Claude Code hooks** | Yes (V2.0) | No | Built-in |
| **Dependencies** | None | SQLite runtime | None |
| **Human-readable data** | JSON | JSONL | SQLite |
| **Parallel agent safety** | Yes | Via daemon | Not addressed |
| **Decision tracking** | Yes | No | No |
| **Learning capture** | Yes | No | No |
| **Task notes** | Yes | Via comments | Via description |
| **Priority levels** | Yes (0-4) | Yes (0-4) | Yes (0-4) |
| **Time tracking** | Yes (V2.0) | No | No |
| **Custom fields** | Yes (V2.0) | No | No |
| **Epic hierarchy** | Via blocking | Native | Via parent |
| **Ready queue** | `tk task ready` | `bd ready` | `dot ready` |

---

## Strengths & Weaknesses

### Tasuku

**Strengths:**
1. **Pull-based philosophy** - Agent queries when needed, no constant context injection
2. **Context capture** - First-class `learnings`, `decisions`, and `notes` - unique to Tasuku
3. **Human-editable** - JSON is readable/editable without tools
4. **MCP-native** - Built-in MCP server, no plugins needed
5. **Parallel-safe** - File locking prevents corruption with multiple agents
6. **Stateless** - No daemon, no sync issues, predictable behavior
7. **Beads migration** - `tk migrate` converts existing beads projects

**Weaknesses:**
1. ~~**No priority system**~~ ✅ Fixed - Now has 0-4 priority levels
2. ~~**No TodoWrite sync**~~ ✅ Fixed - `tk hooks sync` integrates with TodoWrite
3. ~~**No hook integration**~~ ✅ Fixed - `tk hooks session` for startup context
4. **JSON scaling** - Large projects may see slower reads (though rare in practice)
5. ~~**No search**~~ ✅ Fixed - `tk task find` searches all content
6. **New project** - Less battle-tested than beads

### Beads

**Strengths:**
1. **Comprehensive** - Epics, priorities, dependencies, comments, all built-in
2. **Git-native** - Two-way sync with JSONL enables team collaboration
3. **Semantic compaction** - Old issues summarized to save context
4. **Large community** - Most widely adopted, active development
5. **Claude plugin** - Available in Claude Code marketplace
6. **Hierarchical IDs** - `bd-a1b2.1.1` sub-task notation

**Weaknesses:**
1. **Daemon dependency** - Background process can mangle changes, sync issues
2. **Complexity** - 188K lines of code, many moving parts
3. **Slow startup** - ~7ms (2x slower than dots)
4. **Large binary** - 19 MB
5. **Push-based** - Injects full context, wastes tokens
6. **Sync failures** - "Several times a week" per community reports
7. **No native MCP** - Requires plugin installation
8. **Over-engineered** - SQLite + JSONL + daemon for what could be simpler

### Dots

**Strengths:**
1. **Minimal** - 800 lines of Zig, 0.9 MB binary
2. **Fast** - 3ms startup, 2x faster than beads
3. **Beads-compatible** - Can read existing beads.db files
4. **TodoWrite sync** - Hooks integrate with Claude's built-in todo system
5. **Session hooks** - Auto-displays context on session start
6. **No daemon** - Stateless like Tasuku
7. **Priority system** - 0-4 priority levels

**Weaknesses:**
1. **No MCP server** - CLI only, no tool-based integration
2. **SQLite opaque** - Binary database, not human-readable
3. **No parallel safety** - No file locking for multiple agents
4. **No context capture** - No learnings, decisions, or notes
5. **No git sync** - SQLite doesn't merge well
6. **Zig ecosystem** - Less accessible for contributors
7. **Young project** - Limited documentation and community

---

## Use Case Fit

| Use Case | Best Tool | Why |
|----------|-----------|-----|
| **Single agent, simple project** | Dots | Minimal, fast, TodoWrite sync |
| **Multi-agent parallel work** | Tasuku | File locking, stateless |
| **Team collaboration (git)** | Beads | Two-way JSONL sync |
| **Decision documentation** | Tasuku | First-class decisions/learnings |
| **Claude Code native** | Tasuku | Built-in MCP server |
| **Beads migration** | Dots or Tasuku | Both have migration paths |
| **Maximum performance** | Dots | 0.9 MB, 3ms startup |
| **Human supervision** | Tasuku | Readable JSON, easy auditing |
| **Enterprise/complex projects** | Beads | Full feature set |

---

## Potential Improvements for Tasuku

### High Priority

1. **Add priority levels**
   - Add 0-4 priority scale like beads/dots
   - `tk add "task" --priority 1`
   - Sort by priority in `tk list`

2. **TodoWrite hook integration**
   - Add `tk hook sync` command like dots
   - Auto-sync with Claude's built-in TodoWrite
   - Bidirectional: TodoWrite → Tasuku → TodoWrite

3. **Session startup hook**
   - Add `tk hook session` command
   - Display active tasks and ready queue on session start
   - Show recent learnings and decisions

4. **Search command**
   - Add `tk find "query"` for full-text search
   - Search across descriptions, notes, learnings

### Medium Priority

5. **Ready queue command**
   - Add `tk ready` command (alias for `tk list --status ready`)
   - Show unblocked tasks sorted by priority
   - Match beads/dots UX

6. **Tree view**
   - Add `tk tree` for dependency visualization
   - Show blocking relationships graphically

7. **JSON output mode**
   - Add `--json` flag to all commands
   - Enable jq piping for power users

8. **Compact/archive command**
   - Add `tk compact` to archive old done tasks
   - Optional: summarize with AI like beads

### Lower Priority

9. **Web viewer**
   - Simple localhost web UI for visualization
   - Like beads_viewer but built-in

10. **Task templates**
    - Pre-defined task structures for common patterns
    - `tk add --template feature`

11. **Time tracking**
    - Optional duration on tasks
    - `tk done task-id --duration 2h`

12. **Export formats**
    - `tk export --format markdown|csv|github`
    - Integration with external trackers

---

## Competitive Positioning

### Tasuku's Unique Value Proposition

**"The thinking agent's task manager"**

Unlike beads (comprehensive but complex) or dots (minimal but sparse), Tasuku occupies the middle ground:

1. **Context-aware** - Captures not just tasks, but learnings and decisions
2. **Pull-based** - Agent queries when needed, no context pollution
3. **Human-readable** - JSON file you can actually edit
4. **Parallel-safe** - Multiple agents can work without corruption
5. **MCP-native** - First-class Claude Code integration
6. **Stateless** - No daemon, no sync, no surprises

### Target Users

- Developers using Claude Code for substantial projects
- Teams wanting readable, auditable task history
- Projects requiring decision documentation
- Multi-agent workflows needing parallel safety

### Differentiator Summary

| vs Beads | vs Dots |
|----------|---------|
| Simpler (no daemon) | More features (context capture) |
| Faster startup | MCP server built-in |
| Human-readable | Parallel-safe |
| Pull-based | Human-readable |
| Context capture | Decision tracking |

---

## Recommendations

### Immediate (v1.1) - ✅ ALL COMPLETE
1. ✅ Add priority levels (0-4) - `tk task priority`
2. ✅ Add `tk ready` command - `tk task ready`
3. ✅ Add `--json` flag to commands - `-f json` on all commands

### Short-term (v1.2) - ✅ ALL COMPLETE
4. ✅ TodoWrite hook integration - `tk hooks sync`
5. ✅ Session startup hook - `tk hooks session`
6. ✅ Search command - `tk task find`

### Medium-term (v2.0) - MOSTLY COMPLETE
7. ✅ Tree visualization - `tk task deps`
8. ❌ Web viewer - Not yet implemented
9. ❌ Archive/compact - Not yet implemented
10. ✅ Time tracking - `tk task timer` (V2.0)
11. ✅ Custom fields - `tk task field` (V2.0)
12. ✅ GitHub PR integration - `tk pr` (V2.0)

This positions Tasuku as the "thoughtful middle ground" - more capable than dots, simpler than beads, with unique context-capture features neither competitor offers.

---

## Cross-Agent Compatibility

### The Multi-Agent Landscape

The AI coding assistant market is fragmented across multiple platforms:

| Agent | Integration Method | Current Tasuku Support |
|-------|-------------------|----------------------|
| **Claude Code** | MCP (stdio) | Native |
| **Cursor** | MCP (stdio) | Native |
| **Windsurf** | MCP (stdio) | Native |
| **OpenAI Codex** | CLI / API | CLI only |
| **Aider** | CLI | CLI only |
| **Continue** | MCP (stdio) | Native |
| **GitHub Copilot** | None (IDE-embedded) | CLI only |
| **Cline** | MCP (stdio) | Native |
| **Roo Code** | MCP (stdio) | Native |

### Current Integration Approaches

#### 1. MCP Protocol (Primary)
- **Supports:** Claude Code, Cursor, Windsurf, Continue, Cline, Roo Code
- **Status:** Fully implemented via `tk serve`
- **Gap:** None for MCP-compatible agents

#### 2. CLI Interface (Universal)
- **Supports:** All agents that can execute shell commands
- **Status:** Fully implemented
- **Gap:** No machine-readable output (need `--json` flag)

#### 3. Direct File Access
- **Supports:** Any agent that can read/write files
- **Status:** Works (agents can read `.tasuku.json` directly)
- **Gap:** No schema documentation for agents

### Integration Gaps Analysis

#### Gap 1: No JSON Output Mode
**Problem:** Non-MCP agents (Codex, Aider) must parse human-readable CLI output.

**Solution:** Add `--json` flag to all commands:
```bash
tk list --json | jq '.[] | select(.status == "ready")'
```

#### Gap 2: No OpenAI-style Function Calling
**Problem:** OpenAI's Codex uses function calling, not MCP.

**Solution:** Add OpenAPI/JSON Schema spec for Tasuku tools:
```yaml
# openapi.yaml
paths:
  /tasks:
    get:
      operationId: list_tasks
      parameters:
        - name: status
          in: query
          schema:
            enum: [ready, in_progress, blocked, done]
```

This enables Codex to use Tasuku via its native function calling.

#### Gap 3: No HTTP API
**Problem:** Some agents prefer HTTP over stdio/CLI.

**Solution:** Add optional HTTP mode:
```bash
tk serve --http :3000
```

#### Gap 4: No Agent-Agnostic Schema
**Problem:** Agents need to understand `.tasuku.json` structure.

**Solution:** Publish JSON Schema:
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Tasuku File",
  "type": "object",
  "properties": {
    "version": { "type": "integer" },
    "tasks": { "$ref": "#/definitions/TaskMap" },
    "context": { "$ref": "#/definitions/Context" }
  }
}
```

Include in repo as `schema.json`, reference in CLAUDE.md/README.

#### Gap 5: No Aider Integration
**Problem:** Aider has its own conventions and expects specific file patterns.

**Solution:** Add Aider-compatible commands:
```bash
tk aider-status  # Output in Aider's expected format
```

Or document how to use `tk context | aider --message-file -`

### Recommended Integration Roadmap

#### Phase 1: Universal CLI (v1.1) - ✅ COMPLETE
- [x] Add `--json` flag to all commands - `-f json`
- [x] Add `--quiet` flag (exit codes only) - `-q` flag
- [x] Publish JSON Schema for `.tasuku.json` - `tk server schema`
- [x] Add `tk schema` command to output schema - `tk server schema`

#### Phase 2: HTTP API (v1.2) - ✅ COMPLETE
- [x] Add `tk serve --http :3000` - `tk server http`
- [x] OpenAPI 3.0 specification - `/schema` endpoint
- [ ] Swagger UI at `/docs` - Not yet
- [x] CORS headers for web clients - Implemented

#### Phase 3: Agent-Specific Adapters (v2.0) - PARTIAL
- [ ] OpenAI function calling spec
- [ ] Aider integration guide
- [ ] VS Code extension (for Copilot context)
- [ ] LSP server for IDE integration

### Design Principles for Multi-Agent Support

1. **Protocol Agnostic Core**
   - Core logic in `internal/store` and `internal/task`
   - Multiple frontends: CLI, MCP, HTTP, LSP
   - Same operations, different transports

2. **Schema-First Design**
   - JSON Schema as source of truth
   - Generate API specs from schema
   - Validate agent inputs against schema

3. **Stateless Operations**
   - Every operation is idempotent
   - No session state between calls
   - Safe for any agent architecture

4. **Lock-Based Concurrency**
   - File locking works across all integration methods
   - Multiple agents (any type) can work in parallel
   - No central coordinator needed

5. **Human in the Loop**
   - JSON readable without tools
   - Git-friendly for code review
   - Override any agent decision manually

### Competitive Analysis: Multi-Agent Support

| Feature | Tasuku | Beads | Dots |
|---------|--------|-------|------|
| **MCP Server** | Built-in | Plugin | No |
| **HTTP API** | ✅ Built-in | No | No |
| **JSON CLI Output** | ✅ Built-in | Yes | Yes |
| **OpenAPI Spec** | ✅ `/schema` | No | No |
| **JSON Schema** | ✅ `tk server schema` | No | No |
| **LSP Server** | Planned | No | No |
| **Agent-agnostic** | Yes | Claude-focused | Claude-focused |

**Key Insight:** Tasuku is now the most feature-complete agent-agnostic task manager with MCP, HTTP API, and JSON Schema support.

---

## Sources

- [Beads GitHub](https://github.com/steveyegge/beads)
- [Dots GitHub](https://github.com/joelreymont/dots)
- [Beads Guide - Better Stack](https://betterstack.com/community/guides/ai/beads-issue-tracker-ai-agents/)
- [HN: Replacing Beads](https://news.ycombinator.com/item?id=46487580)
- [MCP Specification](https://modelcontextprotocol.io/)
- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
