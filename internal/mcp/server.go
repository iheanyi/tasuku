// Package mcp provides an MCP server for Claude Code integration.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "tasuku"
	ServerVersion   = "0.1.0"
)

// Server is the MCP server.
type Server struct {
	store store.Storage
	in    io.Reader
	out   io.Writer
}

// New creates a new MCP server.
func New(s store.Storage) *Server {
	return &Server{
		store: s,
		in:    os.Stdin,
		out:   os.Stdout,
	}
}

// JSON-RPC 2.0 types

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Protocol types

// InitializeParams are the parameters for the initialize request
type InitializeParams struct {
	ProtocolVersion string      `json:"protocolVersion"`
	Capabilities    interface{} `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// InitializeResult is the result of the initialize request
type InitializeResult struct {
	ProtocolVersion string           `json:"protocolVersion"`
	Capabilities    ServerCapability `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// ServerCapability describes server capabilities
type ServerCapability struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability describes tool capabilities
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolsListResult is the result of tools/list
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams are the parameters for tools/call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResult is the result of tools/call
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in tool results
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Tools returns the list of available tools.
func (s *Server) Tools() []Tool {
	return []Tool{
		{
			Name:        "tk_list",
			Description: "List all tasks, optionally filtered by status, tag, or owner. Use at session start to understand project state, after completing work to see remaining tasks, or when planning to identify what needs attention. Returns task IDs, statuses, descriptions, and blockers.",
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
				},
			},
		},
		{
			Name:        "tk_add",
			Description: "Create a new task with the given description. Use PROACTIVELY when: (1) Breaking down features into subtasks, (2) Discovering follow-up work during implementation, (3) Finding bugs that should be tracked, (4) User requests work that spans multiple steps. Returns the generated task ID.",
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
			Description: "Mark a task as completed. Use IMMEDIATELY when finishing work - don't batch completions. Automatically stops any running timer. After marking done, consider: recording learnings, checking if this unblocks other tasks, archiving if no longer needed.",
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
			Description: "Record a learning or insight discovered while working. Use PROACTIVELY when: (1) Debugging reveals undocumented behavior, (2) Finding gotchas or edge cases, (3) Discovering patterns that work well (or poorly), (4) API behaviors differ from expectations, (5) Performance insights. Use 'Never X' or 'Always Y' prefixes for rules. These persist across sessions and can be promoted to permanent docs.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"insight"},
				"properties": map[string]interface{}{
					"insight": map[string]interface{}{
						"type":        "string",
						"description": "The insight or learning to record",
					},
				},
			},
		},
		{
			Name:        "tk_decide",
			Description: "Record an architectural decision with the choice made, alternatives considered, and reasoning. Use PROACTIVELY when: (1) Selecting technologies or libraries, (2) Choosing between implementation approaches, (3) Making trade-offs (performance vs simplicity, etc.), (4) Deciding on patterns or conventions. Decisions help future agents understand WHY things were built a certain way.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "chose", "over", "because"},
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
						"description": "Alternatives that were considered",
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
			Description: "Get full context including all tasks, learnings, and decisions. Use this at the start of a session to understand current state.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_timer_start",
			Description: "Start a timer on a task to track time spent working on it. Use when beginning focused work on a task. Helps with effort estimation and identifying tasks that take longer than expected. Timer auto-stops on tk_done or tk_pause.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to start timer on",
					},
				},
			},
		},
		{
			Name:        "tk_timer_stop",
			Description: "Stop the timer on a task, recording the elapsed time. Use when pausing work temporarily without completing. Elapsed time accumulates across multiple timer sessions.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to stop timer on",
					},
				},
			},
		},
		{
			Name:        "tk_timer_status",
			Description: "Get the status of all running timers. Use to check if you forgot to stop a timer or to see total time spent on tasks.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_field_set",
			Description: "Set a custom field on a task for structured metadata. Use PROACTIVELY for: (1) Tracking estimates - field 'estimate' with hours/points, (2) Categorization - field 'component' or 'area' for code area, (3) External references - field 'pr', 'issue', or 'commit' for links, (4) Implementation approach - field 'approach' for strategy decisions, (5) Review tracking - field 'needs_review' or 'reviewer'. Fields persist across sessions and enable structured reporting.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "key", "value"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Field name (e.g., 'estimate', 'component', 'pr', 'approach', 'reviewer')",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "Field value",
					},
				},
			},
		},
		{
			Name:        "tk_field_remove",
			Description: "Remove a custom field from a task. Use when field is no longer relevant or was set incorrectly.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "key"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Field name to remove",
					},
				},
			},
		},
		{
			Name:        "tk_tag_add",
			Description: "Add a tag to a task for categorization and filtering. Use PROACTIVELY with tags like: 'bug', 'feature', 'refactor', 'docs', 'test', 'security', 'performance', 'tech-debt', 'urgent'. Tags enable filtering with tk_list and help identify patterns across tasks.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "tag"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Tag to add",
					},
				},
			},
		},
		{
			Name:        "tk_tag_remove",
			Description: "Remove a tag from a task. Use when a tag no longer applies (e.g., 'urgent' after addressing, 'bug' if reclassified as feature).",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "tag"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID",
					},
					"tag": map[string]interface{}{
						"type":        "string",
						"description": "Tag to remove",
					},
				},
			},
		},
		{
			Name:        "tk_archive",
			Description: "Archive a done task to keep the active task list lean. Use after tasks are verified complete and no longer need visibility. Archived tasks are preserved in .tasuku/archive/ for history. The task must be in 'done' status.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"task_id"},
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to archive",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Optional summary of what was accomplished",
					},
				},
			},
		},
		{
			Name:        "tk_archive_restore",
			Description: "Restore an archived task back to active tasks with 'ready' status. Use when archived work needs to be revisited or was archived prematurely.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"task_id"},
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Archived task ID to restore",
					},
				},
			},
		},
		{
			Name:        "tk_archive_list",
			Description: "List all archived tasks. Use to find historical tasks, reference past work, or locate tasks to restore.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_archive_all",
			Description: "Archive all done tasks older than a specified duration. Use for bulk cleanup to reduce clutter. Duration format: 1h (hours), 1d (days), 1w (weeks). Example: '7d' archives tasks done more than 7 days ago.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"older_than"},
				"properties": map[string]interface{}{
					"older_than": map[string]interface{}{
						"type":        "string",
						"description": "Duration threshold (e.g., '7d', '24h', '2w'). Tasks done before this are archived.",
					},
				},
			},
		},
		{
			Name:        "tk_show",
			Description: "Get detailed information about a specific task including notes, priority, timestamps, and custom fields. Use before starting work to understand full context, check notes from previous sessions, or review task metadata.",
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
			Name:        "tk_delete",
			Description: "Permanently delete a task. Use for duplicate tasks, tasks created in error, or work that's no longer relevant. Prefer tk_archive for completed work you might reference later. Also removes associated notes and clears references from other tasks' blocked_by lists.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to delete",
					},
				},
			},
		},
		{
			Name:        "tk_edit",
			Description: "Update a task's description. Use when the scope changes, requirements are clarified, or the original description was unclear. Keep descriptions actionable and specific.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "description"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to edit",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "New description for the task",
					},
				},
			},
		},
		{
			Name:        "tk_pause",
			Description: "Pause work on a task, reverting it from in_progress to ready status. Use when switching to higher-priority work, blocked by external factors, or ending a session with incomplete work. Add a note explaining why paused. Automatically stops any running timer.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to pause",
					},
				},
			},
		},
		{
			Name:        "tk_unblock",
			Description: "Remove blockers from a task. Use when blocking tasks are completed, blockers are resolved externally, or blocking relationship was incorrect. By default removes all blockers; use 'from' to remove a specific one.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to unblock",
					},
					"from": map[string]interface{}{
						"type":        "string",
						"description": "Optional: remove only this specific blocker (partial unblock)",
					},
				},
			},
		},
		{
			Name:        "tk_find",
			Description: "Search across tasks, notes, learnings, and decisions. Use to find related work, check if similar tasks exist before creating new ones, or locate past decisions/learnings on a topic. Case-insensitive text search.",
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
		{
			Name:        "tk_priority",
			Description: "Set task priority level. Use PROACTIVELY to organize work: critical (0) for blocking/urgent issues, high (1) for important near-term work, normal (2) for standard tasks, low (3) for can-wait items, backlog (4) for future ideas. Priority affects tk_ready ordering.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "priority"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "Priority level: 0-4 or critical/high/normal/low/backlog",
					},
				},
			},
		},
		{
			Name:        "tk_owner",
			Description: "Set or clear task owner for assignment tracking. Use to indicate who's responsible for a task, track workload distribution, or filter tasks by assignee. Different from tk_claim which is for active exclusive work.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID",
					},
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Owner name to set. Omit or set empty to clear owner.",
					},
				},
			},
		},
		{
			Name:        "tk_claim",
			Description: "Claim a task for exclusive work by an agent. Use in multi-agent scenarios to prevent duplicate work. Check tk_who before claiming to see what's already claimed. Records claim timestamp for coordination. Release with tk_release when done or pausing.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "agent"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to claim",
					},
					"agent": map[string]interface{}{
						"type":        "string",
						"description": "Agent name claiming the task",
					},
				},
			},
		},
		{
			Name:        "tk_release",
			Description: "Release a claimed task, making it available for other agents. Use when: (1) Completing work (after tk_done), (2) Pausing for extended time, (3) Realizing another agent should handle it, (4) Ending a session with incomplete work.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to release",
					},
				},
			},
		},
		{
			Name:        "tk_suggest",
			Description: "Analyze a task description and suggest whether it should be persisted to tk (project-level) or kept as a session-only TodoWrite item. Use this before adding items to TodoWrite to determine if they should also be tracked in tk.",
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
		// Ready tasks
		{
			Name:        "tk_ready",
			Description: "List tasks that are ready to work on (not blocked, sorted by priority). Use at session start to pick up work, after completing a task to find the next one, or when deciding what to focus on. Shows highest-priority actionable tasks first.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// Who (claimed tasks by owner)
		{
			Name:        "tk_who",
			Description: "Show tasks claimed by each owner/agent. Use before claiming to avoid conflicts, to understand workload distribution, or to find who's working on related tasks. Essential for multi-agent coordination.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// Dependencies
		{
			Name:        "tk_deps",
			Description: "Show the dependency tree for a task - what it's blocked by and what it blocks. Use to understand task relationships, identify critical path, or find tasks that will be unblocked when completing work.",
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
		// Stats
		{
			Name:        "tk_stats",
			Description: "Show task statistics: counts by status, priority distribution, completion rate. Use for project health checks, identifying bottlenecks (high blocked count), or reporting progress. Helps understand overall project state.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// Learning list
		{
			Name:        "tk_learning_list",
			Description: "List all recorded learnings with their IDs and rule status. Use to review accumulated knowledge, find learnings to promote, or refresh memory on project-specific insights before starting work.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// Learning promote
		{
			Name:        "tk_learning_promote",
			Description: "Promote a learning to permanent documentation (CLAUDE.md or similar). Use this for valuable insights that should persist beyond the session. Agents should autonomously promote rule learnings (never/always patterns) that prove useful.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Learning ID to promote",
					},
					"to": map[string]interface{}{
						"type":        "string",
						"description": "Target file (auto-detected if not specified: CLAUDE.md, .cursorrules, etc.)",
					},
					"keep": map[string]interface{}{
						"type":        "boolean",
						"description": "Keep the learning in Tasuku after promoting (default: false)",
					},
				},
			},
		},
		// Learning remove
		{
			Name:        "tk_learning_remove",
			Description: "Remove a learning by ID. Use when a learning is outdated, incorrect, or has been promoted to permanent docs and is no longer needed in Tasuku.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Learning ID to remove",
					},
				},
			},
		},
		// Learning rules (suggest candidates for promotion)
		{
			Name:        "tk_learning_rules",
			Description: "List learnings marked as rules (never/always patterns) that are candidates for promotion to permanent docs. Use this to find learnings that should be promoted.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// Decision list
		{
			Name:        "tk_decision_list",
			Description: "List all recorded architectural decisions. Use to understand why things were built a certain way, before making similar decisions, or to document project architecture for new contributors.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		// Decision remove
		{
			Name:        "tk_decision_remove",
			Description: "Remove a decision by ID. Use when a decision is reversed, superseded by a new decision, or was recorded in error.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Decision ID to remove",
					},
				},
			},
		},
		// Note list
		{
			Name:        "tk_note_list",
			Description: "List notes for a specific task or all notes across tasks. Use to review progress history, understand context from previous sessions, or find specific information captured during work.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to list notes for (omit for all notes)",
					},
				},
			},
		},
		// Note remove
		{
			Name:        "tk_note_remove",
			Description: "Remove a note from a task. Use when a note is outdated, contains incorrect information, or is no longer relevant after task completion.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"task_id", "note_id"},
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID the note belongs to",
					},
					"note_id": map[string]interface{}{
						"type":        "string",
						"description": "Note ID to remove",
					},
				},
			},
		},
		// Health check
		{
			Name:        "tk_health",
			Description: "Get a project health check with actionable recommendations. Use at session start to understand project state, or periodically to identify issues. Returns: task distribution, stale items, rule learnings to promote, and specific recommendations.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// HandleToolCall processes a tool call and returns the result.
func (s *Server) HandleToolCall(name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "tk_list":
		return s.handleList(args)
	case "tk_add":
		return s.handleAdd(args)
	case "tk_start":
		return s.handleStart(args)
	case "tk_done":
		return s.handleDone(args)
	case "tk_block":
		return s.handleBlock(args)
	case "tk_learn":
		return s.handleLearn(args)
	case "tk_decide":
		return s.handleDecide(args)
	case "tk_note":
		return s.handleNote(args)
	case "tk_context":
		return s.handleContext(args)
	case "tk_timer_start":
		return s.handleTimerStart(args)
	case "tk_timer_stop":
		return s.handleTimerStop(args)
	case "tk_timer_status":
		return s.handleTimerStatus(args)
	case "tk_field_set":
		return s.handleFieldSet(args)
	case "tk_field_remove":
		return s.handleFieldRemove(args)
	case "tk_tag_add":
		return s.handleTagAdd(args)
	case "tk_tag_remove":
		return s.handleTagRemove(args)
	case "tk_archive":
		return s.handleArchive(args)
	case "tk_archive_restore":
		return s.handleArchiveRestore(args)
	case "tk_archive_list":
		return s.handleArchiveList(args)
	case "tk_archive_all":
		return s.handleArchiveAll(args)
	case "tk_show":
		return s.handleShow(args)
	case "tk_delete":
		return s.handleDelete(args)
	case "tk_edit":
		return s.handleEdit(args)
	case "tk_pause":
		return s.handlePause(args)
	case "tk_unblock":
		return s.handleUnblock(args)
	case "tk_find":
		return s.handleFind(args)
	case "tk_priority":
		return s.handlePriority(args)
	case "tk_owner":
		return s.handleOwner(args)
	case "tk_claim":
		return s.handleClaim(args)
	case "tk_release":
		return s.handleRelease(args)
	case "tk_suggest":
		return s.handleSuggest(args)
	case "tk_ready":
		return s.handleReady(args)
	case "tk_who":
		return s.handleWho(args)
	case "tk_deps":
		return s.handleDeps(args)
	case "tk_stats":
		return s.handleStats(args)
	case "tk_learning_list":
		return s.handleLearningList(args)
	case "tk_learning_promote":
		return s.handleLearningPromote(args)
	case "tk_learning_remove":
		return s.handleLearningRemove(args)
	case "tk_learning_rules":
		return s.handleLearningRules(args)
	case "tk_decision_list":
		return s.handleDecisionList(args)
	case "tk_decision_remove":
		return s.handleDecisionRemove(args)
	case "tk_note_list":
		return s.handleNoteList(args)
	case "tk_note_remove":
		return s.handleNoteRemove(args)
	case "tk_health":
		return s.handleHealth(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) handleList(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	// Parse filter arguments
	status, _ := args["status"].(string)
	tagFilter, _ := args["tag"].(string)
	ownerFilter, _ := args["owner"].(string)
	treeView, _ := args["tree"].(bool)

	type taskResult struct {
		ID          string         `json:"id"`
		Status      string         `json:"status"`
		Description string         `json:"description"`
		BlockedBy   []string       `json:"blocked_by,omitempty"`
		Owner       *string        `json:"owner,omitempty"`
		ParentID    string         `json:"parent_id,omitempty"`
		Priority    int            `json:"priority,omitempty"`
		Tags        []string       `json:"tags,omitempty"`
		Children    []taskResult   `json:"children,omitempty"`
	}

	// Filter tasks
	var results []taskResult
	for id, t := range f.Tasks {
		if status != "" && string(t.Status) != status {
			continue
		}
		if tagFilter != "" && !t.HasTag(tagFilter) {
			continue
		}
		if ownerFilter != "" {
			if t.Owner == nil || *t.Owner != ownerFilter {
				continue
			}
		}
		parentID := ""
		if t.ParentID != nil {
			parentID = *t.ParentID
		}
		results = append(results, taskResult{
			ID:          id,
			Status:      string(t.Status),
			Description: t.Description,
			BlockedBy:   t.BlockedBy,
			Owner:       t.Owner,
			ParentID:    parentID,
			Priority:    t.GetPriority(),
			Tags:        t.Tags,
		})
	}

	// Sort by status priority, then by task priority, then by ID
	statusOrder := map[task.Status]int{
		task.StatusInProgress: 0,
		task.StatusReady:      1,
		task.StatusBlocked:    2,
		task.StatusDone:       3,
	}
	slices.SortFunc(results, func(a, b taskResult) int {
		// Status order first
		aStatus := statusOrder[task.Status(a.Status)]
		bStatus := statusOrder[task.Status(b.Status)]
		if aStatus != bStatus {
			return aStatus - bStatus
		}
		// Priority second
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		// ID alphabetically last
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	// If tree view requested, nest children under parents
	if treeView {
		// Build lookup map
		taskMap := make(map[string]*taskResult)
		for i := range results {
			taskMap[results[i].ID] = &results[i]
		}

		// Build tree
		var rootTasks []taskResult
		for i := range results {
			t := &results[i]
			if t.ParentID == "" {
				rootTasks = append(rootTasks, *t)
			} else if parent, ok := taskMap[t.ParentID]; ok {
				parent.Children = append(parent.Children, *t)
			} else {
				// Parent not in filtered results, treat as root
				rootTasks = append(rootTasks, *t)
			}
		}
		return rootTasks, nil
	}

	return results, nil
}

func (s *Server) handleAdd(args map[string]interface{}) (interface{}, error) {
	desc, _ := args["description"].(string)
	id, _ := args["id"].(string)
	priorityStr, _ := args["priority"].(string)
	parentID, _ := args["parent_id"].(string)

	// Parse tags - handle both []interface{} from JSON and []string
	var tags []string
	if tagsRaw, ok := args["tags"].([]interface{}); ok {
		for _, t := range tagsRaw {
			if s, ok := t.(string); ok {
				tags = append(tags, strings.TrimSpace(s))
			}
		}
	} else if tagsStr, ok := args["tags"].([]string); ok {
		for _, t := range tagsStr {
			tags = append(tags, strings.TrimSpace(t))
		}
	}

	// Read current state for duplicate detection
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	// Generate ID if not provided, checking for collisions
	existingIDs := make(map[string]struct{})
	for taskID := range f.Tasks {
		existingIDs[taskID] = struct{}{}
	}

	if id == "" {
		id = task.GenerateTaskID(desc, existingIDs)
	}

	// Check for potential duplicates (similar descriptions)
	descLower := strings.ToLower(desc)
	var potentialDuplicates []string
	for taskID, t := range f.Tasks {
		if t.Status != task.StatusDone {
			existingLower := strings.ToLower(t.Description)
			// Check for significant overlap
			if strings.Contains(existingLower, descLower) || strings.Contains(descLower, existingLower) {
				potentialDuplicates = append(potentialDuplicates, taskID)
			}
		}
	}

	// Parse priority (supports both numeric and named values)
	var priorityPtr *int
	if priorityStr != "" {
		priority := task.ParsePriority(priorityStr)
		if priority == -1 {
			return nil, fmt.Errorf("invalid priority: %s (use 0-4 or critical/high/normal/low/backlog)", priorityStr)
		}
		priorityPtr = &priority
	}

	// Create task - either as subtask or regular task
	if parentID != "" {
		if err := s.store.AddSubtask(id, desc, parentID); err != nil {
			return nil, err
		}
		// Apply priority and tags after creation
		if priorityPtr != nil {
			s.store.SetPriority(id, *priorityPtr)
		}
		for _, tag := range tags {
			s.store.AddTag(id, tag)
		}
	} else {
		if err := s.store.AddTaskWithTags(id, desc, priorityPtr, tags); err != nil {
			return nil, err
		}
	}

	result := map[string]interface{}{"id": id, "status": "created"}

	if priorityPtr != nil {
		result["priority"] = task.PriorityName(*priorityPtr)
	}
	if len(tags) > 0 {
		result["tags"] = tags
	}
	if parentID != "" {
		result["parent_id"] = parentID
	}
	if len(potentialDuplicates) > 0 {
		result["warning"] = fmt.Sprintf("Potential duplicate tasks found: %v. Consider checking these before proceeding.", potentialDuplicates)
	}

	return result, nil
}

func (s *Server) handleStart(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	startTimer, _ := args["start_timer"].(bool)
	unblock, _ := args["unblock"].(bool)

	// Read current state for context
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	// Check for other in_progress tasks (warning)
	var otherInProgress []string
	for taskID, t := range f.Tasks {
		if t.Status == task.StatusInProgress && taskID != id {
			otherInProgress = append(otherInProgress, taskID)
		}
	}

	// Get notes for this task (context from previous sessions)
	notes := f.Context.Notes[id]

	// Unblock if requested (for blocked tasks)
	if unblock {
		if err := s.store.UnblockTask(id); err != nil {
			return nil, fmt.Errorf("failed to unblock task: %w", err)
		}
	}

	if err := s.store.SetStatus(id, task.StatusInProgress); err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "in_progress"}

	if unblock {
		result["unblocked"] = true
	}

	if startTimer {
		if err := s.store.StartTimer(id); err != nil {
			result["timer_warning"] = err.Error()
		} else {
			result["timer_started"] = true
		}
	}

	// Add warnings and context
	if len(otherInProgress) > 0 {
		result["warning"] = fmt.Sprintf("You have %d other task(s) in_progress: %v. Consider pausing them with tk_pause.", len(otherInProgress), otherInProgress)
	}

	if len(notes) > 0 {
		// Show recent notes for context
		type noteInfo struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		var recentNotes []noteInfo
		for i := len(notes) - 1; i >= 0 && i >= len(notes)-3; i-- {
			recentNotes = append(recentNotes, noteInfo{ID: notes[i].ID, Text: notes[i].Text})
		}
		result["previous_notes"] = recentNotes
		result["hint"] = "Review previous notes above for context from earlier sessions."
	}

	return result, nil
}

func (s *Server) handleDone(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	// Auto-stop timer if running
	elapsed, wasRunning, err := s.store.StopTimerIfRunning(id)
	if err != nil {
		return nil, err
	}

	// SetStatusAndRead combines setting status + reading file (avoids redundant read)
	f, err := s.store.SetStatusAndRead(id, task.StatusDone)
	if err != nil {
		return nil, err
	}

	// Find tasks that were blocked by this task and are now unblocked
	var nowUnblocked []string
	for taskID, t := range f.Tasks {
		if t.Status == task.StatusBlocked {
			// Check if this task was the only blocker
			allBlockersDone := true
			wasBlockedByUs := false
			for _, blockerID := range t.BlockedBy {
				if blockerID == id {
					wasBlockedByUs = true
				}
				if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
					allBlockersDone = false
					break
				}
			}
			if wasBlockedByUs && allBlockersDone {
				nowUnblocked = append(nowUnblocked, taskID)
			}
		}
	}

	result := map[string]interface{}{"id": id, "status": "done"}
	if wasRunning {
		result["timer_stopped"] = true
		result["elapsed"] = elapsed.String()
	}

	// Add smart suggestions
	var hints []string

	if len(nowUnblocked) > 0 {
		result["unblocked_tasks"] = nowUnblocked
		hints = append(hints, fmt.Sprintf("Completing this task unblocked %d task(s): %v", len(nowUnblocked), nowUnblocked))
	}

	// Check if task has notes worth reviewing for learnings
	if notes := f.Context.Notes[id]; len(notes) > 0 {
		hints = append(hints, "This task has notes - consider if any insights should be recorded as learnings with tk_learn.")
	}

	// Reflection prompt - always shown for significant tasks
	hints = append(hints, "REFLECT: Did completing this task involve decisions (tk_decide) or reveal learnings (tk_learn) worth preserving?")

	// Suggest archiving if appropriate
	hints = append(hints, "Consider archiving with tk_archive if this task is fully verified and no longer needs visibility.")

	if len(hints) > 0 {
		result["hints"] = hints
	}

	return result, nil
}

func (s *Server) handleBlock(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	var blockers []string
	if b, ok := args["blocked_by"].([]interface{}); ok {
		for _, v := range b {
			if s, ok := v.(string); ok {
				blockers = append(blockers, s)
			}
		}
	}

	if err := s.store.BlockTask(id, blockers); err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": id, "status": "blocked", "blocked_by": blockers}, nil
}

func (s *Server) handleLearn(args map[string]interface{}) (interface{}, error) {
	insight, _ := args["insight"].(string)
	id, isRule, err := s.store.AddLearningWithRule(insight, nil)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "status": "added", "is_rule": isRule}, nil
}

func (s *Server) handleDecide(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	chose, _ := args["chose"].(string)
	because, _ := args["because"].(string)

	var over []string
	if o, ok := args["over"].([]interface{}); ok {
		for _, v := range o {
			if s, ok := v.(string); ok {
				over = append(over, s)
			}
		}
	}

	d := task.Decision{
		ID:        id,
		Chose:     chose,
		Over:      over,
		Because:   because,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.store.AddDecision(d); err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "recorded"}, nil
}

func (s *Server) handleNote(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	note, _ := args["note"].(string)

	id, err := s.store.AddNote(taskID, note)
	if err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "task_id": taskID, "status": "added"}, nil
}

func (s *Server) handleContext(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	// Generate smart suggestions based on project state
	var suggestions []string

	// Count tasks by status
	statusCounts := map[string]int{}
	var highPriorityReady []string
	var staleInProgress []string
	now := time.Now()

	for id, t := range f.Tasks {
		statusCounts[string(t.Status)]++

		// Find high-priority ready tasks
		if t.Status == task.StatusReady && t.GetPriority() <= task.PriorityHigh {
			highPriorityReady = append(highPriorityReady, id)
		}

		// Find stale in_progress tasks (>24h without update)
		if t.Status == task.StatusInProgress && now.Sub(t.UpdatedAt) > 24*time.Hour {
			staleInProgress = append(staleInProgress, id)
		}
	}

	// Generate suggestions
	if len(highPriorityReady) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("High-priority tasks ready: %v", highPriorityReady))
	}

	if len(staleInProgress) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Stale in_progress tasks (>24h): %v - consider updating or pausing", staleInProgress))
	}

	if statusCounts["blocked"] > 3 {
		suggestions = append(suggestions, fmt.Sprintf("%d blocked tasks - review blockers to unblock progress", statusCounts["blocked"]))
	}

	// Check for rule learnings that should be promoted
	ruleCount := 0
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			ruleCount++
		}
	}
	if ruleCount > 0 {
		suggestions = append(suggestions, fmt.Sprintf("%d rule learnings ready for promotion to docs - use tk_learning_rules", ruleCount))
	}

	// Build enhanced response
	result := map[string]interface{}{
		"tasks":     f.Tasks,
		"context":   f.Context,
		"version":   f.Version,
		"task_counts": map[string]int{
			"ready":       statusCounts["ready"],
			"in_progress": statusCounts["in_progress"],
			"blocked":     statusCounts["blocked"],
			"done":        statusCounts["done"],
			"total":       len(f.Tasks),
		},
	}

	if len(suggestions) > 0 {
		result["suggestions"] = suggestions
	}

	return result, nil
}

func (s *Server) handleTimerStart(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	if err := s.store.StartTimer(id); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "status": "timer_started"}, nil
}

func (s *Server) handleTimerStop(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	elapsed, err := s.store.StopTimer(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":      id,
		"status":  "timer_stopped",
		"elapsed": elapsed.String(),
	}, nil
}

func (s *Server) handleTimerStatus(args map[string]interface{}) (interface{}, error) {
	timers, err := s.store.GetActiveTimers()
	if err != nil {
		return nil, err
	}

	type timerInfo struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		StartedAt   string `json:"started_at"`
		Elapsed     string `json:"elapsed"`
		Warning     string `json:"warning,omitempty"`
	}

	var results []timerInfo
	var warnings []string

	for id, t := range timers {
		elapsed := time.Since(*t.TimerStart)
		info := timerInfo{
			ID:          id,
			Description: t.Description,
			StartedAt:   t.TimerStart.Format(time.RFC3339),
			Elapsed:     elapsed.Truncate(time.Second).String(),
		}

		// Add warnings for long-running timers
		if elapsed > 4*time.Hour {
			info.Warning = "Running for over 4 hours - consider stopping if not actively working"
			warnings = append(warnings, fmt.Sprintf("%s: running for %s", id, info.Elapsed))
		} else if elapsed > 2*time.Hour {
			info.Warning = "Running for over 2 hours"
		}

		results = append(results, info)
	}

	response := map[string]interface{}{
		"timers": results,
		"count":  len(results),
	}

	if len(warnings) > 0 {
		response["warnings"] = warnings
		response["hint"] = "Long-running timers detected. If you're not actively working, stop them with tk_timer_stop."
	}

	if len(results) == 0 {
		response["hint"] = "No active timers. Start one with tk_timer_start when beginning focused work."
	}

	return response, nil
}

func (s *Server) handleFieldSet(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)

	if err := s.store.SetField(id, key, value); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "key": key, "value": value, "status": "set"}, nil
}

func (s *Server) handleFieldRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	key, _ := args["key"].(string)

	if err := s.store.RemoveField(id, key); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "key": key, "status": "removed"}, nil
}

func (s *Server) handleTagAdd(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	tag, _ := args["tag"].(string)

	if err := s.store.AddTag(id, tag); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "tag": tag, "status": "added"}, nil
}

func (s *Server) handleTagRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	tag, _ := args["tag"].(string)

	if err := s.store.RemoveTag(id, tag); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "tag": tag, "status": "removed"}, nil
}

func (s *Server) handleArchive(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	summary, _ := args["summary"].(string)

	if err := s.store.ArchiveTask(taskID, summary); err != nil {
		return nil, err
	}
	return map[string]string{"id": taskID, "status": "archived"}, nil
}

func (s *Server) handleArchiveRestore(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)

	if err := s.store.RestoreTask(taskID); err != nil {
		return nil, err
	}
	return map[string]string{"id": taskID, "status": "restored"}, nil
}

func (s *Server) handleArchiveList(args map[string]interface{}) (interface{}, error) {
	archived, err := s.store.GetArchivedTasks()
	if err != nil {
		return nil, err
	}

	type archivedResult struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Summary     string `json:"summary,omitempty"`
		ArchivedAt  string `json:"archived_at"`
	}

	var results []archivedResult
	for id, t := range archived {
		results = append(results, archivedResult{
			ID:          id,
			Description: t.Description,
			Summary:     t.Summary,
			ArchivedAt:  t.ArchivedAt.Format(time.RFC3339),
		})
	}

	return results, nil
}

func (s *Server) handleArchiveAll(args map[string]interface{}) (interface{}, error) {
	olderThan, _ := args["older_than"].(string)
	if olderThan == "" {
		return nil, fmt.Errorf("older_than is required")
	}

	// Parse duration string (e.g., "7d", "24h", "2w")
	duration, err := parseDurationString(olderThan)
	if err != nil {
		return nil, fmt.Errorf("invalid duration format '%s': %w", olderThan, err)
	}

	archived, err := s.store.ArchiveDoneTasks(duration)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"archived_count": len(archived),
		"archived_tasks": archived,
		"message":        fmt.Sprintf("Archived %d task(s) older than %s", len(archived), olderThan),
	}, nil
}

// parseDurationString parses a human-friendly duration string like "7d", "24h", "2w"
func parseDurationString(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", valueStr)
	}

	switch unit {
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit: %c (use h, d, or w)", unit)
	}
}

func (s *Server) handleShow(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	t, exists := f.Tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	notes := f.Context.Notes[id]

	type noteInfo struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		CreatedAt string `json:"created_at"`
	}

	var noteResults []noteInfo
	for _, n := range notes {
		noteResults = append(noteResults, noteInfo{
			ID:        n.ID,
			Text:      n.Text,
			CreatedAt: n.CreatedAt.Format(time.RFC3339),
		})
	}

	result := map[string]interface{}{
		"id":          id,
		"description": t.Description,
		"status":      string(t.Status),
		"priority":    t.Priority,
		"created_at":  t.CreatedAt.Format(time.RFC3339),
		"updated_at":  t.UpdatedAt.Format(time.RFC3339),
	}

	if t.Owner != nil {
		result["owner"] = *t.Owner
	}
	if len(t.BlockedBy) > 0 {
		result["blocked_by"] = t.BlockedBy
	}
	if len(t.Tags) > 0 {
		result["tags"] = t.Tags
	}
	if len(t.Fields) > 0 {
		result["fields"] = t.Fields
	}
	if t.Duration > 0 {
		result["duration"] = t.Duration.String()
	}
	if len(noteResults) > 0 {
		result["notes"] = noteResults
	}

	return result, nil
}

func (s *Server) handleDelete(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	err := s.store.Update(func(f *task.File) error {
		if _, exists := f.Tasks[id]; !exists {
			return fmt.Errorf("task not found: %s", id)
		}

		// Delete the task
		delete(f.Tasks, id)

		// Remove notes for this task
		delete(f.Context.Notes, id)

		// Remove this task from any blocked_by arrays in other tasks
		for tid, t := range f.Tasks {
			newBlockedBy := []string{}
			for _, blockerID := range t.BlockedBy {
				if blockerID != id {
					newBlockedBy = append(newBlockedBy, blockerID)
				}
			}
			if len(newBlockedBy) != len(t.BlockedBy) {
				t.BlockedBy = newBlockedBy
				// If no more blockers, set to ready
				if len(newBlockedBy) == 0 && t.Status == task.StatusBlocked {
					t.Status = task.StatusReady
				}
				t.UpdatedAt = time.Now().UTC()
				f.Tasks[tid] = t
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "deleted"}, nil
}

func (s *Server) handleEdit(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	description, _ := args["description"].(string)

	if err := s.store.SetDescription(id, description); err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "updated", "description": description}, nil
}

func (s *Server) handlePause(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	// Auto-stop timer if running
	elapsed, wasRunning, err := s.store.StopTimerIfRunning(id)
	if err != nil {
		return nil, err
	}

	err = s.store.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("task not found: %s", id)
		}

		if t.Status != task.StatusInProgress {
			return fmt.Errorf("task %s is not in_progress (current status: %s)", id, t.Status)
		}

		t.Status = task.StatusReady
		t.Owner = nil
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t

		return nil
	})

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "ready"}
	if wasRunning {
		result["timer_stopped"] = true
		result["elapsed"] = elapsed.String()
	}

	// Prompt for note about why paused
	result["hint"] = "Consider adding a note about why this was paused and current progress: tk_note task_id='" + id + "' note='...'"

	return result, nil
}

func (s *Server) handleUnblock(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	fromBlocker, _ := args["from"].(string)

	if fromBlocker == "" {
		// Clear all blockers
		if err := s.store.UnblockTask(id); err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": id, "status": "ready", "removed": "all"}, nil
	}

	// Partial unblock: remove only the specified blocker
	err := s.store.Update(func(f *task.File) error {
		t, exists := f.Tasks[id]
		if !exists {
			return fmt.Errorf("task %q not found", id)
		}

		found := false
		newBlockers := []string{}
		for _, b := range t.BlockedBy {
			if b == fromBlocker {
				found = true
			} else {
				newBlockers = append(newBlockers, b)
			}
		}

		if !found {
			return fmt.Errorf("task %q is not blocked by %q", id, fromBlocker)
		}

		t.BlockedBy = newBlockers
		if len(newBlockers) == 0 {
			t.Status = task.StatusReady
		}
		t.UpdatedAt = time.Now().UTC()
		f.Tasks[id] = t

		return nil
	})

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": id, "removed": fromBlocker}, nil
}

func (s *Server) handleFind(args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	queryLower := strings.ToLower(query)

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type searchResult struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Content string `json:"content"`
	}

	var results []searchResult

	// Search tasks
	for id, t := range f.Tasks {
		if strings.Contains(strings.ToLower(t.Description), queryLower) ||
			strings.Contains(strings.ToLower(id), queryLower) {
			results = append(results, searchResult{
				Type:    "task",
				ID:      id,
				Content: t.Description,
			})
		}
	}

	// Search notes
	for taskID, notes := range f.Context.Notes {
		for _, n := range notes {
			if strings.Contains(strings.ToLower(n.Text), queryLower) {
				results = append(results, searchResult{
					Type:    "note",
					ID:      taskID + "/" + n.ID,
					Content: n.Text,
				})
			}
		}
	}

	// Search learnings
	for _, l := range f.Context.Learnings {
		if strings.Contains(strings.ToLower(l.Text), queryLower) {
			results = append(results, searchResult{
				Type:    "learning",
				ID:      l.ID,
				Content: l.Text,
			})
		}
	}

	// Search decisions
	for _, d := range f.Context.Decisions {
		if strings.Contains(strings.ToLower(d.ID), queryLower) ||
			strings.Contains(strings.ToLower(d.Chose), queryLower) ||
			strings.Contains(strings.ToLower(d.Because), queryLower) {
			results = append(results, searchResult{
				Type:    "decision",
				ID:      d.ID,
				Content: d.Chose + " because: " + d.Because,
			})
		}
	}

	return results, nil
}

func (s *Server) handlePriority(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	priorityStr, _ := args["priority"].(string)

	var priority int
	switch priorityStr {
	case "0", "critical":
		priority = task.PriorityCritical
	case "1", "high":
		priority = task.PriorityHigh
	case "2", "normal":
		priority = task.PriorityNormal
	case "3", "low":
		priority = task.PriorityLow
	case "4", "backlog":
		priority = task.PriorityBacklog
	default:
		return nil, fmt.Errorf("invalid priority: %s (use 0-4 or critical/high/normal/low/backlog)", priorityStr)
	}

	if err := s.store.SetPriority(id, priority); err != nil {
		return nil, err
	}

	return map[string]interface{}{"id": id, "priority": priority, "status": "updated"}, nil
}

func (s *Server) handleOwner(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	owner, hasOwner := args["owner"].(string)

	if !hasOwner || owner == "" {
		// Clear owner
		if err := s.store.ClearOwner(id); err != nil {
			return nil, err
		}
		return map[string]string{"id": id, "status": "owner_cleared"}, nil
	}

	// Set owner
	if err := s.store.SetOwner(id, owner); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "owner": owner, "status": "owner_set"}, nil
}

func (s *Server) handleClaim(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	agent, _ := args["agent"].(string)

	// Check current state for warnings
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	t, exists := f.Tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	result := map[string]interface{}{"id": id, "agent": agent}

	// Warn if already claimed by someone else
	if t.Owner != nil && *t.Owner != "" && *t.Owner != agent {
		result["warning"] = fmt.Sprintf("Task was claimed by '%s'. Overriding claim.", *t.Owner)
	}

	// Check if agent already has other claimed tasks
	var otherClaimed []string
	for taskID, task := range f.Tasks {
		if task.Owner != nil && *task.Owner == agent && taskID != id {
			otherClaimed = append(otherClaimed, taskID)
		}
	}
	if len(otherClaimed) > 0 {
		result["note"] = fmt.Sprintf("You already have %d other claimed task(s): %v", len(otherClaimed), otherClaimed)
	}

	if err := s.store.ClaimTask(id, agent); err != nil {
		return nil, err
	}

	result["status"] = "claimed"
	return result, nil
}

func (s *Server) handleRelease(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	// Check if task has notes before releasing
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	notes := f.Context.Notes[id]

	if err := s.store.ReleaseTask(id); err != nil {
		return nil, err
	}

	result := map[string]interface{}{"id": id, "status": "released"}

	if len(notes) == 0 {
		result["hint"] = "Consider adding a note about current progress before releasing: tk_note"
	}

	return result, nil
}

func (s *Server) handleSuggest(args map[string]interface{}) (interface{}, error) {
	description, _ := args["description"].(string)
	desc := strings.ToLower(description)

	// Keywords that indicate project-level tasks (should persist to tk)
	projectKeywords := []string{
		"implement", "add feature", "build", "create", "develop",
		"fix bug", "bugfix", "hotfix", "patch",
		"refactor", "rewrite", "redesign", "rearchitect",
		"migrate", "upgrade", "update dependency",
		"integrate", "connect", "setup", "configure",
		"support", "enable", "add support",
		"milestone", "epic", "feature", "story",
		"api endpoint", "database", "schema",
		"authentication", "authorization", "security",
		"performance", "optimize", "cache",
		"deploy", "release", "ship",
	}

	// Keywords that indicate session-level tasks (TodoWrite only)
	sessionKeywords := []string{
		"fix type error", "fix typo", "fix lint",
		"update file", "edit file", "modify file",
		"read file", "check file", "review file",
		"run test", "run build", "run script",
		"verify", "check", "confirm", "ensure",
		"debug", "investigate", "look into",
		"format", "cleanup", "tidy",
		"add comment", "add docstring", "add import",
		"remove unused", "delete unused",
		"rename variable", "rename function",
	}

	shouldPersist := false
	reason := "No strong project-level indicators found"
	matchedKeyword := ""

	// Check for project keywords
	for _, kw := range projectKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = true
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains project-level keyword '%s' - this looks like a feature, bug, or significant change that should be tracked across sessions", kw)
			break
		}
	}

	// Session keywords can override if they match
	for _, kw := range sessionKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = false
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains session-level keyword '%s' - this looks like an implementation step that doesn't need to persist", kw)
			break
		}
	}

	result := map[string]interface{}{
		"should_persist":    shouldPersist,
		"reason":            reason,
		"matched_keyword":   matchedKeyword,
		"original_description": description,
	}

	if shouldPersist {
		result["suggested_command"] = fmt.Sprintf("tk task add %q", description)
		result["recommendation"] = "Add this to tk for persistent tracking across sessions"
	} else {
		result["recommendation"] = "Keep this in TodoWrite only - it's a session-level implementation step"
	}

	return result, nil
}

func (s *Server) handleReady(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type readyTask struct {
		ID          string  `json:"id"`
		Description string  `json:"description"`
		Priority    int     `json:"priority"`
		Owner       *string `json:"owner,omitempty"`
	}

	var results []readyTask
	inProgressCount := 0
	blockedCount := 0

	for id, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			inProgressCount++
		}
		if t.Status == task.StatusBlocked {
			blockedCount++
		}
		if t.Status != task.StatusReady {
			continue
		}
		// Check if actually blocked
		blocked := false
		for _, blockerID := range t.BlockedBy {
			if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
				blocked = true
				break
			}
		}
		if !blocked {
			results = append(results, readyTask{
				ID:          id,
				Description: t.Description,
				Priority:    t.GetPriority(),
				Owner:       t.Owner,
			})
		}
	}

	// Sort by priority (lower number = higher priority) - O(n log n) vs O(n²) bubble sort
	slices.SortFunc(results, func(a, b readyTask) int {
		return a.Priority - b.Priority
	})

	// Build response with stats
	response := map[string]interface{}{
		"tasks": results,
		"stats": map[string]int{
			"ready":       len(results),
			"in_progress": inProgressCount,
			"blocked":     blockedCount,
		},
	}

	if len(results) > 0 {
		response["recommendation"] = fmt.Sprintf("Highest priority task: %s", results[0].ID)
	} else if inProgressCount > 0 {
		response["recommendation"] = "No ready tasks - focus on completing in_progress work"
	} else if blockedCount > 0 {
		response["recommendation"] = "All tasks blocked - review blockers with tk_list status=blocked"
	}

	return response, nil
}

func (s *Server) handleWho(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type ownedTask struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	ownerMap := make(map[string][]ownedTask)
	for id, t := range f.Tasks {
		if t.Owner != nil && *t.Owner != "" {
			ownerMap[*t.Owner] = append(ownerMap[*t.Owner], ownedTask{
				ID:          id,
				Description: t.Description,
				Status:      string(t.Status),
			})
		}
	}

	return ownerMap, nil
}

func (s *Server) handleDeps(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	t, exists := f.Tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	// Find what this task is blocked by
	type depInfo struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	var blockedBy []depInfo
	for _, blockerID := range t.BlockedBy {
		if blocker, exists := f.Tasks[blockerID]; exists {
			blockedBy = append(blockedBy, depInfo{
				ID:          blockerID,
				Description: blocker.Description,
				Status:      string(blocker.Status),
			})
		}
	}

	// Find what this task blocks
	var blocks []depInfo
	for otherID, other := range f.Tasks {
		for _, blockerID := range other.BlockedBy {
			if blockerID == id {
				blocks = append(blocks, depInfo{
					ID:          otherID,
					Description: other.Description,
					Status:      string(other.Status),
				})
				break
			}
		}
	}

	return map[string]interface{}{
		"id":          id,
		"description": t.Description,
		"status":      string(t.Status),
		"blocked_by":  blockedBy,
		"blocks":      blocks,
	}, nil
}

func (s *Server) handleStats(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	statusCounts := map[string]int{
		"ready":       0,
		"in_progress": 0,
		"blocked":     0,
		"done":        0,
	}

	priorityCounts := map[string]int{
		"critical": 0,
		"high":     0,
		"normal":   0,
		"low":      0,
		"backlog":  0,
	}

	total := len(f.Tasks)
	for _, t := range f.Tasks {
		statusCounts[string(t.Status)]++

		switch t.GetPriority() {
		case task.PriorityCritical:
			priorityCounts["critical"]++
		case task.PriorityHigh:
			priorityCounts["high"]++
		case task.PriorityNormal:
			priorityCounts["normal"]++
		case task.PriorityLow:
			priorityCounts["low"]++
		case task.PriorityBacklog:
			priorityCounts["backlog"]++
		}
	}

	completionRate := 0.0
	if total > 0 {
		completionRate = float64(statusCounts["done"]) / float64(total) * 100
	}

	return map[string]interface{}{
		"total":           total,
		"by_status":       statusCounts,
		"by_priority":     priorityCounts,
		"completion_rate": fmt.Sprintf("%.1f%%", completionRate),
		"learnings_count": len(f.Context.Learnings),
		"decisions_count": len(f.Context.Decisions),
	}, nil
}

func (s *Server) handleLearningList(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type learningResult struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		IsRule    bool   `json:"is_rule"`
		CreatedAt string `json:"created_at"`
	}

	var results []learningResult
	for _, l := range f.Context.Learnings {
		results = append(results, learningResult{
			ID:        l.ID,
			Text:      l.Text,
			IsRule:    l.IsRule,
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
		})
	}

	return results, nil
}

func (s *Server) handleLearningPromote(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	targetFile, _ := args["to"].(string)
	keep, _ := args["keep"].(bool)

	if targetFile == "" {
		targetFile = detectContextFile()
	}

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	// Find the learning
	var foundLearning *task.Learning
	var foundIdx int
	for i := range f.Context.Learnings {
		if f.Context.Learnings[i].ID == id {
			foundLearning = &f.Context.Learnings[i]
			foundIdx = i
			break
		}
	}

	if foundLearning == nil {
		return nil, fmt.Errorf("learning not found: %s", id)
	}

	// Append to context file
	if err := appendToContextFile(targetFile, foundLearning.Text); err != nil {
		return nil, fmt.Errorf("failed to write to %s: %w", targetFile, err)
	}

	// Remove from learnings if not keeping
	if !keep {
		if _, err := s.store.RemoveLearning(id); err != nil {
			return nil, err
		}
	}

	result := map[string]interface{}{
		"id":          id,
		"promoted_to": targetFile,
		"text":        foundLearning.Text,
		"kept":        keep,
	}

	_ = foundIdx // Used for potential future optimization

	return result, nil
}

func (s *Server) handleLearningRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	removedText, err := s.store.RemoveLearning(id)
	if err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "removed": removedText}, nil
}

func (s *Server) handleLearningRules(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type ruleResult struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		CreatedAt string `json:"created_at"`
		Hint      string `json:"hint"`
	}

	var results []ruleResult
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			results = append(results, ruleResult{
				ID:        l.ID,
				Text:      l.Text,
				CreatedAt: l.CreatedAt.Format(time.RFC3339),
				Hint:      "Consider promoting with tk_learning_promote",
			})
		}
	}

	return map[string]interface{}{
		"rules":          results,
		"count":          len(results),
		"recommendation": "Rule learnings (never/always patterns) should be promoted to permanent docs when they prove useful",
	}, nil
}

func (s *Server) handleDecisionList(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type decisionResult struct {
		ID        string   `json:"id"`
		Chose     string   `json:"chose"`
		Over      []string `json:"over"`
		Because   string   `json:"because"`
		CreatedAt string   `json:"created_at"`
	}

	var results []decisionResult
	for _, d := range f.Context.Decisions {
		results = append(results, decisionResult{
			ID:        d.ID,
			Chose:     d.Chose,
			Over:      d.Over,
			Because:   d.Because,
			CreatedAt: d.CreatedAt.Format(time.RFC3339),
		})
	}

	return results, nil
}

func (s *Server) handleDecisionRemove(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	err := s.store.Update(func(f *task.File) error {
		for i, d := range f.Context.Decisions {
			if d.ID == id {
				f.Context.Decisions = append(f.Context.Decisions[:i], f.Context.Decisions[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("decision not found: %s", id)
	})

	if err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "removed"}, nil
}

func (s *Server) handleNoteList(args map[string]interface{}) (interface{}, error) {
	taskID, hasTaskID := args["task_id"].(string)

	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	type noteResult struct {
		TaskID    string `json:"task_id"`
		NoteID    string `json:"note_id"`
		Text      string `json:"text"`
		CreatedAt string `json:"created_at"`
	}

	var results []noteResult

	if hasTaskID && taskID != "" {
		// List notes for specific task
		notes := f.Context.Notes[taskID]
		for _, n := range notes {
			results = append(results, noteResult{
				TaskID:    taskID,
				NoteID:    n.ID,
				Text:      n.Text,
				CreatedAt: n.CreatedAt.Format(time.RFC3339),
			})
		}
	} else {
		// List all notes
		for tid, notes := range f.Context.Notes {
			for _, n := range notes {
				results = append(results, noteResult{
					TaskID:    tid,
					NoteID:    n.ID,
					Text:      n.Text,
					CreatedAt: n.CreatedAt.Format(time.RFC3339),
				})
			}
		}
	}

	return results, nil
}

func (s *Server) handleNoteRemove(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	noteID, _ := args["note_id"].(string)

	removedText, err := s.store.RemoveNote(taskID, noteID)
	if err != nil {
		return nil, err
	}

	return map[string]string{"task_id": taskID, "note_id": noteID, "removed": removedText}, nil
}

func (s *Server) handleHealth(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Collect metrics
	statusCounts := map[string]int{}
	priorityCounts := map[string]int{}
	var staleInProgress []string
	var staleDone []string
	var longRunningTimers []string
	var highPriorityBlocked []string

	for id, t := range f.Tasks {
		statusCounts[string(t.Status)]++

		switch t.GetPriority() {
		case task.PriorityCritical:
			priorityCounts["critical"]++
		case task.PriorityHigh:
			priorityCounts["high"]++
		case task.PriorityNormal:
			priorityCounts["normal"]++
		case task.PriorityLow:
			priorityCounts["low"]++
		case task.PriorityBacklog:
			priorityCounts["backlog"]++
		}

		// Stale in_progress (>24h)
		if t.Status == task.StatusInProgress && now.Sub(t.UpdatedAt) > 24*time.Hour {
			staleInProgress = append(staleInProgress, id)
		}

		// Stale done tasks (>7 days, not archived)
		if t.Status == task.StatusDone && now.Sub(t.UpdatedAt) > 7*24*time.Hour {
			staleDone = append(staleDone, id)
		}

		// High priority blocked
		if t.Status == task.StatusBlocked && t.GetPriority() <= task.PriorityHigh {
			highPriorityBlocked = append(highPriorityBlocked, id)
		}

		// Long-running timers
		if t.IsTimerRunning() && now.Sub(*t.TimerStart) > 4*time.Hour {
			longRunningTimers = append(longRunningTimers, id)
		}
	}

	// Count rule learnings
	ruleCount := 0
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			ruleCount++
		}
	}

	// Build recommendations
	var recommendations []string

	if len(staleInProgress) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("STALE: %d in_progress task(s) not updated in 24h: %v - update or pause them", len(staleInProgress), staleInProgress))
	}

	if len(highPriorityBlocked) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("BLOCKED: %d high-priority task(s) blocked: %v - unblock to make progress", len(highPriorityBlocked), highPriorityBlocked))
	}

	if len(longRunningTimers) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("TIMERS: %d timer(s) running 4+ hours: %v - stop if not active", len(longRunningTimers), longRunningTimers))
	}

	if len(staleDone) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("ARCHIVE: %d done task(s) older than 7 days: consider archiving with tk_archive", len(staleDone)))
	}

	if ruleCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("PROMOTE: %d rule learning(s) ready for promotion to docs - use tk_learning_rules", ruleCount))
	}

	// Calculate health score (simple heuristic)
	healthScore := 100
	healthScore -= len(staleInProgress) * 10
	healthScore -= len(highPriorityBlocked) * 15
	healthScore -= len(longRunningTimers) * 5
	if healthScore < 0 {
		healthScore = 0
	}

	var healthStatus string
	if healthScore >= 80 {
		healthStatus = "healthy"
	} else if healthScore >= 50 {
		healthStatus = "needs attention"
	} else {
		healthStatus = "unhealthy"
	}

	return map[string]interface{}{
		"health_score":  healthScore,
		"health_status": healthStatus,
		"task_counts": map[string]int{
			"total":       len(f.Tasks),
			"ready":       statusCounts["ready"],
			"in_progress": statusCounts["in_progress"],
			"blocked":     statusCounts["blocked"],
			"done":        statusCounts["done"],
		},
		"priority_counts": priorityCounts,
		"issues": map[string]interface{}{
			"stale_in_progress":     staleInProgress,
			"high_priority_blocked": highPriorityBlocked,
			"long_running_timers":   longRunningTimers,
			"stale_done_count":      len(staleDone),
			"rule_learnings":        ruleCount,
		},
		"recommendations": recommendations,
		"learnings_count": len(f.Context.Learnings),
		"decisions_count": len(f.Context.Decisions),
	}, nil
}

// Helper functions for learning promote

func detectContextFile() string {
	contextFiles := []string{
		"CLAUDE.md",
		".cursorrules",
		".github/copilot-instructions.md",
		"AGENTS.md",
		"AI.md",
	}

	for _, cf := range contextFiles {
		if _, err := os.Stat(cf); err == nil {
			return cf
		}
	}

	return "CLAUDE.md"
}

func appendToContextFile(filePath, learning string) error {
	existing, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(existing)
	entry := fmt.Sprintf("- %s\n", learning)

	if strings.Contains(text, "## Learnings") {
		idx := strings.Index(text, "## Learnings")
		endOfLine := strings.Index(text[idx:], "\n") + idx + 1

		nextSection := strings.Index(text[endOfLine:], "\n## ")
		if nextSection == -1 {
			text = text + entry
		} else {
			insertAt := endOfLine + nextSection
			text = text[:insertAt] + entry + text[insertAt:]
		}
	} else {
		if len(text) > 0 && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "\n## Learnings\n\n" + entry
	}

	return os.WriteFile(filePath, []byte(text), 0644)
}

// generateID creates a kebab-case ID from description (without collision check).
// Used for tests. Production code should use task.GenerateTaskID with existingIDs.
func generateID(desc string) string {
	return task.GenerateTaskID(desc, nil)
}

// Run starts the MCP server in stdio mode using JSON-RPC 2.0.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.in)
	// Increase buffer size for large messages
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *Server) handleInitialize(req *Request) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapability{
			Tools: &ToolsCapability{},
		},
	}
	result.ServerInfo.Name = ServerName
	result.ServerInfo.Version = ServerVersion

	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req *Request) {
	result := ToolsListResult{
		Tools: s.Tools(),
	}
	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsCall(req *Request) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	result, err := s.HandleToolCall(params.Name, params.Arguments)
	if err != nil {
		// Return error as tool result, not JSON-RPC error
		s.sendResult(req.ID, ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
		return
	}

	// Convert result to JSON text
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	s.sendResult(req.ID, ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	})
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.sendResponse(resp)
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.sendResponse(resp)
}

func (s *Server) sendResponse(resp Response) {
	data, _ := json.Marshal(resp)
	fmt.Fprintln(s.out, string(data))
}
