// Package mcp provides an MCP server for Claude Code integration.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
			Description: "List all tasks, optionally filtered by status. Returns task IDs, statuses, descriptions, and blockers.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"ready", "in_progress", "blocked", "done"},
						"description": "Filter by status",
					},
				},
			},
		},
		{
			Name:        "tk_add",
			Description: "Create a new task with the given description. Returns the generated task ID.",
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
				},
			},
		},
		{
			Name:        "tk_start",
			Description: "Mark a task as in_progress. Use this when you begin working on a task.",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"required": []string{"id"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID to start",
					},
				},
			},
		},
		{
			Name:        "tk_done",
			Description: "Mark a task as completed. Use this when you finish working on a task.",
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
			Description: "Mark a task as blocked by other tasks.",
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
			Description: "Record a learning or insight discovered while working. These persist across sessions.",
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
			Description: "Record an architectural decision with the choice made, alternatives considered, and reasoning.",
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
			Description: "Add a note to a specific task. Notes persist across sessions.",
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
			Description: "Start a timer on a task to track time spent working on it.",
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
			Description: "Stop the timer on a task, recording the elapsed time.",
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
			Description: "Get the status of all running timers.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_field_set",
			Description: "Set a custom field on a task (key-value metadata).",
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
						"description": "Field name",
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
			Description: "Remove a custom field from a task.",
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
			Description: "Add a tag to a task for categorization and filtering.",
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
			Description: "Remove a tag from a task.",
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
			Description: "Archive a done task. The task must be in 'done' status to be archived.",
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
			Description: "Restore an archived task back to active tasks with 'ready' status.",
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
			Description: "List all archived tasks.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "tk_show",
			Description: "Get detailed information about a specific task including notes, priority, and timestamps.",
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
			Description: "Permanently delete a task. Also removes associated notes and clears references from other tasks' blocked_by lists.",
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
			Description: "Update a task's description.",
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
			Description: "Pause work on a task, reverting it from in_progress to ready status and clearing the owner.",
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
			Description: "Remove blockers from a task. By default removes all blockers; use 'from' to remove a specific one.",
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
			Description: "Search across tasks, notes, learnings, and decisions. Case-insensitive text search.",
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
			Description: "Set task priority level. Levels: 0/critical, 1/high, 2/normal, 3/low, 4/backlog.",
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
			Description: "Set or clear task owner.",
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
			Description: "Claim a task for exclusive work by an agent. Records claim timestamp for coordination.",
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
			Description: "Release a claimed task, making it available for other agents.",
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
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) handleList(args map[string]interface{}) (interface{}, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	status, _ := args["status"].(string)

	type taskResult struct {
		ID          string   `json:"id"`
		Status      string   `json:"status"`
		Description string   `json:"description"`
		BlockedBy   []string `json:"blocked_by,omitempty"`
		Owner       *string  `json:"owner,omitempty"`
	}

	var results []taskResult
	for id, t := range f.Tasks {
		if status != "" && string(t.Status) != status {
			continue
		}
		results = append(results, taskResult{
			ID:          id,
			Status:      string(t.Status),
			Description: t.Description,
			BlockedBy:   t.BlockedBy,
			Owner:       t.Owner,
		})
	}

	return results, nil
}

func (s *Server) handleAdd(args map[string]interface{}) (interface{}, error) {
	desc, _ := args["description"].(string)
	id, _ := args["id"].(string)

	if id == "" {
		id = generateID(desc)
	}

	if err := s.store.AddTask(id, desc); err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "created"}, nil
}

func (s *Server) handleStart(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	if err := s.store.SetStatus(id, task.StatusInProgress); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "status": "in_progress"}, nil
}

func (s *Server) handleDone(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)
	if err := s.store.SetStatus(id, task.StatusDone); err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "status": "done"}, nil
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
	return s.store.Read()
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
	}

	var results []timerInfo
	for id, t := range timers {
		results = append(results, timerInfo{
			ID:          id,
			Description: t.Description,
			StartedAt:   t.TimerStart.Format(time.RFC3339),
			Elapsed:     time.Since(*t.TimerStart).Truncate(time.Second).String(),
		})
	}
	return results, nil
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

	err := s.store.Update(func(f *task.File) error {
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

	return map[string]string{"id": id, "status": "ready"}, nil
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

	if err := s.store.ClaimTask(id, agent); err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "agent": agent, "status": "claimed"}, nil
}

func (s *Server) handleRelease(args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	if err := s.store.ReleaseTask(id); err != nil {
		return nil, err
	}

	return map[string]string{"id": id, "status": "released"}, nil
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

// generateID creates a unique kebab-case ID from description.
func generateID(desc string) string {
	return task.GenerateTaskID(desc)
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
