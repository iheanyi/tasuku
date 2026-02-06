// Package mcp provides an MCP server for Claude Code integration.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iheanyi/tasuku/internal/store"
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

		// Detect batch requests (JSON arrays) - not supported per MCP spec
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			s.sendError(nil, -32600, "Invalid Request", "Batch requests not supported")
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
	case "initialized", "notifications/initialized":
		// Notification, no response needed
	case "notifications/cancelled":
		// Client cancelled a request, no response needed
	case "notifications/progress":
		// Client reporting progress on a request, no response needed
	case "notifications/roots/list_changed":
		// Client's filesystem roots changed, no response needed
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
		Instructions: "Tasuku is an agent-first task management system. " +
			"Use tk_context at session start to load project state. " +
			"Use tk_learn to capture insights and tk_decide for architectural decisions.",
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
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		s.sendError(req.ID, -32603, fmt.Sprintf("failed to marshal result: %v", err), nil)
		return
	}
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
	data, err := json.Marshal(resp)
	if err != nil {
		// Log to stderr since this is a protocol-level failure
		fmt.Fprintf(os.Stderr, "mcp: failed to marshal response: %v\n", err)
		return
	}
	fmt.Fprintln(s.out, string(data))
}
