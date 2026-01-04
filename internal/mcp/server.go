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
	store *store.Store
	in    io.Reader
	out   io.Writer
}

// New creates a new MCP server.
func New(s *store.Store) *Server {
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
	id, err := s.store.AddLearning(insight)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id, "status": "added"}, nil
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

// generateID creates a kebab-case ID from description.
func generateID(desc string) string {
	result := ""
	for _, r := range desc {
		if r >= 'a' && r <= 'z' {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else if r == ' ' && len(result) > 0 && result[len(result)-1] != '-' {
			result += "-"
		}
	}
	result = strings.TrimSuffix(result, "-")
	if len(result) > 32 {
		result = result[:32]
	}
	return result
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
