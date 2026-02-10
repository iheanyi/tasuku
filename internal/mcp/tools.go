package mcp

// Tools returns the list of available tools.
func (s *Server) Tools() []Tool {
	return []Tool{
		// === TIER 1: Core tools (kept individual) ===
		{
			Name:        "tk_help",
			Description: "Get help and reference for Tasuku tools. Use this when you need to understand how to use Tasuku, discover available actions, or get guidance on workflows. Topics: overview (default), tasks, metadata, knowledge, multiagent, archive, install.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"overview", "tasks", "metadata", "knowledge", "multiagent", "archive", "install"},
						"description": "Help topic to display (default: overview)",
					},
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Get detailed help for a specific tool (e.g., 'tk_task', 'tk_metadata')",
					},
				},
			},
		},
		{
			Name:        "tk_list",
			Description: "List all tasks (todos), optionally filtered by status, tag, or owner. Use at session start to understand project state, after completing work to see remaining tasks, or when planning to identify what needs attention. Returns task IDs, statuses, descriptions, and blockers. Aliases: show todos, view tasks, get task list.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"ready", "in_progress", "blocked", "done"},
						"description": "Filter by status",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Filter by tag (e.g., 'bug', 'feature', 'urgent')",
					},
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Filter by owner name",
					},
					"tree": map[string]interface{}{
						"type":        "boolean",
						"description": "Show tasks in hierarchical tree view with subtasks nested under parents (default: false)",
					},
					"include_archived": map[string]interface{}{
						"type":        "boolean",
						"description": "Include archived tasks in the list (default: false)",
					},
				},
			},
		},
		{
			Name:        "tk_add",
			Description: "Create a new task (todo) with the given description. Use PROACTIVELY when: (1) Breaking down features into subtasks, (2) Discovering follow-up work during implementation, (3) Finding bugs that should be tracked, (4) User requests work that spans multiple steps. Returns the generated task ID. Aliases: add todo, new task, create task.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"description"},
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Task description",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Optional task ID (auto-generated from description if not provided)",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "Priority level: critical (0), high (1), normal (2), low (3), backlog (4). Accepts both names and numbers.",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags for categorization (e.g., ['bug', 'backend'])",
					},
					"parent_id": map[string]interface{}{
						"type":        "string",
						"description": "Parent task ID to create this as a subtask",
					},
				},
			},
		},
		{
			Name:        "tk_start",
			Description: "Mark a task as in_progress. Use this when you begin working on a task. Optionally starts a timer.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to start",
					},
					"start_timer": map[string]interface{}{
						"type":        "boolean",
						"description": "Also start a timer on the task (default: false)",
					},
					"unblock": map[string]interface{}{
						"type":        "boolean",
						"description": "Clear blockers before starting (for blocked tasks). Allows starting blocked tasks in one command.",
					},
				},
			},
		},
		{
			Name:        "tk_done",
			Description: "Mark a task as completed (done/finished). Use IMMEDIATELY when finishing work - don't batch completions. Automatically stops any running timer. After marking done, consider: recording learnings, checking if this unblocks other tasks, archiving if no longer needed. Aliases: complete task, finish task, mark complete.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to mark as done",
					},
				},
			},
		},
		{
			Name:        "tk_block",
			Description: "Mark a task as blocked by other tasks. Use when: (1) Work cannot proceed until another task completes, (2) External dependencies are discovered, (3) Prerequisites are identified during implementation. Blocked tasks won't appear in tk_ready. Auto-unblocks when blocking tasks are done.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "blocked_by"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to mark as blocked",
					},
					"blocked_by": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of blocking task IDs",
					},
				},
			},
		},
		{
			Name:        "tk_learn",
			Description: "Record a learning or insight discovered while working. Use PROACTIVELY when: (1) Debugging reveals undocumented behavior, (2) Finding gotchas or edge cases, (3) Discovering patterns that work well (or poorly), (4) API behaviors differ from expectations, (5) Performance insights. Use 'Never X' or 'Always Y' prefixes for rules. Use 'scope' to apply learnings to specific file patterns. These persist across sessions and auto-sync to editor rules (e.g., .claude/rules/, .gemini/rules/). AGENT SELF-AWARENESS: Call this tool immediately when you: (a) Say 'got it', 'I see', 'now I understand' after user clarification, (b) Say 'you're right', 'my mistake' after being corrected, (c) Discover something unexpected ('turns out', 'interesting'). Capture the insight while context is fresh - don't wait!",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"insight"},
				"properties": map[string]interface{}{
					"insight": map[string]interface{}{
						"type":        "string",
						"description": "The insight or learning to record",
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Optional glob pattern for path-scoped rules (e.g., 'src/api/**', 'src/components/**/*.tsx'). Scoped learnings are synced to separate rule files with paths frontmatter.",
					},
				},
			},
		},
		{
			Name:        "tk_decide",
			Description: "Record an architectural decision with the choice made, alternatives considered, and reasoning. Use PROACTIVELY when: (1) Selecting technologies or libraries, (2) Choosing between implementation approaches, (3) Making trade-offs (performance vs simplicity, etc.), (4) Deciding on patterns or conventions. Decisions help future agents understand WHY things were built a certain way. AGENT SELF-AWARENESS: Call this immediately when user explains 'because we...', 'we chose X over Y because...', or 'the reason is...'. Record the decision while context is fresh!",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "chose", "because"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Decision identifier (e.g., 'auth-strategy', 'database-choice')",
					},
					"chose": map[string]interface{}{
						"type":        "string",
						"description": "The option that was chosen",
					},
					"over": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Alternatives that were considered (optional)",
					},
					"because": map[string]interface{}{
						"type":        "string",
						"description": "Reasoning for the decision",
					},
				},
			},
		},
		{
			Name:        "tk_note",
			Description: "Add a note to a task to capture context, progress, or insights. PROACTIVELY use this when: (1) Starting a task - note your planned approach, (2) Making progress - note milestones or partial completions, (3) Encountering issues - note blockers, failed approaches, or workarounds, (4) Discovering context - note relevant findings that future agents should know. Notes persist across sessions and help maintain continuity.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"task_id", "note"},
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to add note to",
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "The note text",
					},
				},
			},
		},
		{
			Name:        "tk_context",
			Description: "Get full project context including all tasks (todos), learnings, and decisions. Use this at the start of a session to understand current state. Returns a complete overview/summary of project status for agent consumption.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_ready",
			Description: "List tasks that are ready to work on (status=ready and not blocked). Use when picking up work, planning next steps, or checking what can be started. Returns tasks sorted by priority.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_deps",
			Description: "Show dependencies for a task: what blocks it and what it blocks. Use when understanding task relationships, planning order, or checking impact of completing a task.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to show dependencies for",
					},
				},
			},
		},
		{
			Name:        "tk_suggest",
			Description: "Suggest whether a task should be in tk (Tasuku) or a session TodoWrite. Use when user provides a task description to determine the right tracking system.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"description"},
				"properties": map[string]interface{}{
					"description": map[string]interface{}{
						"type":        "string",
						"description": "The task description to analyze",
					},
				},
			},
		},
		// === TIER 2: Consolidated tools ===
		{
			Name: "tk_task",
			Description: `Perform task operations: edit, delete, pause, block, unblock, priority, owner, archive, restore, claim, release, who.

Actions:
- edit: Update task description. Params: id, description
- delete: Permanently delete a task. Params: id
- pause: Revert in_progress → ready. Params: id
- block: Mark task as blocked. Params: id, blocked_by (array)
- unblock: Remove blockers. Params: id, from (optional specific blocker)
- priority: Set priority (0-4 or critical/high/normal/low/backlog). Params: id, priority
- owner: Set or clear task owner. Params: id, owner (empty to clear)
- archive: Archive a done task. Params: id, summary (optional)
- restore: Restore archived task. Params: id
- claim: Claim for exclusive work. Params: id, agent
- release: Release claimed task. Params: id
- who: Show task assignments by owner. No params required.

Use tk_help command=tk_task for detailed documentation.`,
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"action"},
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"edit", "delete", "pause", "block", "unblock", "priority", "owner", "archive", "restore", "claim", "release", "who"},
						"description": "The task action to perform",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID (required for most actions)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "New description (for edit action)",
					},
					"blocked_by": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of blocking task IDs (for block action)",
					},
					"from": map[string]interface{}{
						"type":        "string",
						"description": "Specific blocker to remove (for unblock action, omit to remove all)",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "Priority level: 0-4 or critical/high/normal/low/backlog (for priority action)",
					},
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Owner name (for owner action, empty to clear)",
					},
					"agent": map[string]interface{}{
						"type":        "string",
						"description": "Agent name (for claim action)",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Summary of accomplishment (for archive action)",
					},
				},
			},
		},
		{
			Name: "tk_metadata",
			Description: `Manage task metadata: tags, fields, notes.

Actions:
- tag_add: Add a tag to a task. Params: id, tag
- tag_remove: Remove a tag from a task. Params: id, tag
- field_set: Set a custom field. Params: id, key, value
- field_remove: Remove a custom field. Params: id, key
- note_list: List notes for a task or all notes. Params: task_id (optional)
- note_remove: Remove a note. Params: task_id, note_id

Common tags: bug, feature, refactor, docs, test, security, performance, tech-debt, urgent
Common fields: estimate, component, pr, issue, approach, reviewer

Use tk_help command=tk_metadata for detailed documentation.`,
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"action"},
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"tag_add", "tag_remove", "field_set", "field_remove", "note_list", "note_remove"},
						"description": "The metadata action to perform",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID (for tag/field operations)",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Tag name (for tag_add/tag_remove)",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Field key (for field_set/field_remove)",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "Field value (for field_set)",
					},
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID (for note_list/note_remove)",
					},
					"note_id": map[string]interface{}{
						"type":        "string",
						"description": "Note ID (for note_remove)",
					},
				},
			},
		},
		{
			Name: "tk_manage",
			Description: `Manage learnings, decisions, and archive operations.

Actions:
- learning_list: List all learnings with IDs and rule status
- learning_promote: Promote learning to permanent docs. Params: id, to (optional target file), keep (optional bool)
- learning_remove: Remove a learning by ID. Params: id
- learning_rules: List learnings marked as rules (never/always patterns)
- decision_list: List all architectural decisions
- decision_remove: Remove a decision by ID. Params: id
- archive_list: List all archived tasks
- archive_all: Archive done tasks older than duration. Params: older_than (e.g., '7d', '24h', '2w')

Use tk_help command=tk_manage for detailed documentation.`,
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"action"},
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"learning_list", "learning_promote", "learning_remove", "learning_rules", "decision_list", "decision_remove", "archive_list", "archive_all"},
						"description": "The management action to perform",
					},
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Learning or decision ID (for promote/remove actions)",
					},
					"to": map[string]interface{}{
						"type":        "string",
						"description": "Target file for promotion (default: auto-detected)",
					},
					"keep": map[string]interface{}{
						"type":        "boolean",
						"description": "Keep learning in Tasuku after promoting (default: false)",
					},
					"older_than": map[string]interface{}{
						"type":        "string",
						"description": "Duration threshold for archive_all (e.g., '7d', '24h', '2w')",
					},
				},
			},
		},
		{
			Name:        "tk_show",
			Description: "View detailed information about a specific task including notes, priority, timestamps, custom fields, and dependencies (what it blocks and what blocks it). Use before starting work to understand full context, check notes from previous sessions, or review task metadata. Aliases: task details, inspect task, get task info.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to show details for",
					},
				},
			},
		},
		{
			Name:        "tk_find",
			Description: "Search/lookup across tasks, notes, learnings, and decisions. Use to find related work, check if similar tasks exist before creating new ones, or locate past decisions/learnings on a topic. Case-insensitive text search. Aliases: query tasks, search todos.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query string",
					},
				},
			},
		},
		// === TIER 3: Useful but less frequent tools (kept individual) ===
		{
			Name:        "tk_stats",
			Description: "Show task statistics and metrics: counts by status, priority distribution, completion rate, progress overview. Use for project health checks, identifying bottlenecks (high blocked count), or reporting progress.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_health",
			Description: "Get a project health check with actionable recommendations and diagnostics. Use at session start to understand project state, or periodically to identify issues. Returns: task distribution, stale items, rule learnings to promote, and specific recommendations.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}
