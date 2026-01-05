package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iheanyi/tasuku/internal/store"
)

func setupTestServer(t *testing.T) (*Server, string) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := store.New(path)
	if err := s.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	return New(s), path
}

func TestTools(t *testing.T) {
	server, _ := setupTestServer(t)
	tools := server.Tools()

	expectedTools := []string{
		"tk_list", "tk_add", "tk_start", "tk_done",
		"tk_block", "tk_learn", "tk_decide", "tk_note", "tk_context",
		"tk_timer_start", "tk_timer_stop", "tk_timer_status",
		"tk_field_set", "tk_field_remove",
		"tk_tag_add", "tk_tag_remove",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("missing tool: %s", expected)
		}
	}
}

func TestHandleToolCall_Add(t *testing.T) {
	server, _ := setupTestServer(t)

	result, err := server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Test task",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", result)
	}

	if r["status"] != "created" {
		t.Errorf("expected status 'created', got %s", r["status"])
	}

	if r["id"] == "" {
		t.Error("expected non-empty id")
	}
}

func TestHandleToolCall_AddWithID(t *testing.T) {
	server, _ := setupTestServer(t)

	result, err := server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Test task",
		"id":          "custom-id",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := result.(map[string]string)
	if r["id"] != "custom-id" {
		t.Errorf("expected id 'custom-id', got %s", r["id"])
	}
}

func TestHandleToolCall_StartDone(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Test task",
		"id":          "test-task",
	})

	// Start it
	result, err := server.HandleToolCall("tk_start", map[string]interface{}{
		"id": "test-task",
	})
	if err != nil {
		t.Fatalf("start error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %s", r["status"])
	}

	// Complete it
	result, err = server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "test-task",
	})
	if err != nil {
		t.Fatalf("done error: %v", err)
	}

	r = result.(map[string]string)
	if r["status"] != "done" {
		t.Errorf("expected status 'done', got %s", r["status"])
	}
}

func TestHandleToolCall_Learn(t *testing.T) {
	server, _ := setupTestServer(t)

	result, err := server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Test learning",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["status"] != "added" {
		t.Errorf("expected status 'added', got %s", r["status"])
	}
	// Verify is_rule is included (auto-detected as false for this text)
	if _, ok := r["is_rule"]; !ok {
		t.Error("expected is_rule field in response")
	}
}

func TestHandleToolCall_Decide(t *testing.T) {
	server, _ := setupTestServer(t)

	result, err := server.HandleToolCall("tk_decide", map[string]interface{}{
		"id":      "test-decision",
		"chose":   "Option A",
		"over":    []interface{}{"Option B", "Option C"},
		"because": "It's simpler",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "recorded" {
		t.Errorf("expected status 'recorded', got %s", r["status"])
	}
}

func TestHandleToolCall_Context(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add some data
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Test task",
		"id":          "test-task",
	})
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Test learning",
	})

	// Get context
	result, err := server.HandleToolCall("tk_context", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's a task.File
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), "test-task") {
		t.Error("context should contain test-task")
	}
	if !strings.Contains(string(data), "Test learning") {
		t.Error("context should contain Test learning")
	}
}

func TestHandleToolCall_List(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add tasks
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Task 1",
		"id":          "task-1",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Task 2",
		"id":          "task-2",
	})
	server.HandleToolCall("tk_start", map[string]interface{}{
		"id": "task-1",
	})

	// List all
	result, err := server.HandleToolCall("tk_list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Convert to JSON and back to verify structure
	data, _ := json.Marshal(result)
	var list []map[string]interface{}
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("failed to unmarshal list: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(list))
	}

	// List with filter
	result, err = server.HandleToolCall("tk_list", map[string]interface{}{
		"status": "in_progress",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ = json.Marshal(result)
	var filtered []map[string]interface{}
	json.Unmarshal(data, &filtered)
	if len(filtered) != 1 {
		t.Errorf("expected 1 in_progress task, got %d", len(filtered))
	}
}

func TestHandleToolCall_Unknown(t *testing.T) {
	server, _ := setupTestServer(t)

	_, err := server.HandleToolCall("unknown_tool", map[string]interface{}{})

	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestGenerateID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Fix authentication bug", "fix-authentication-bug"},
		{"Add logout button", "add-logout-button"},
		{"UPPERCASE TEST", "uppercase-test"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special!@#Characters", "specialcharacters"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateID(tt.input)
			if result != tt.expected {
				t.Errorf("generateID(%q) = %q, expected %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateID_TruncatesAt32(t *testing.T) {
	long := "This is a very long description that exceeds thirty two characters"
	result := generateID(long)

	if len(result) > 32 {
		t.Errorf("generateID should truncate to 32 chars, got %d", len(result))
	}
}

func TestMCPProtocol_Initialize(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create input/output buffers
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
`)
	output := &bytes.Buffer{}

	server.in = input
	server.out = output

	// Run server (will exit after reading all input)
	server.Run()

	// Parse response
	var resp Response
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nOutput: %s", err, output.String())
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	if resp.ID != float64(1) {
		t.Errorf("expected id 1, got %v", resp.ID)
	}
}

func TestMCPProtocol_ToolsList(t *testing.T) {
	server, _ := setupTestServer(t)

	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}
`)
	output := &bytes.Buffer{}

	server.in = input
	server.out = output

	server.Run()

	var resp Response
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be map, got %T", resp.Result)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools to be array, got %T", result["tools"])
	}

	if len(tools) != 16 {
		t.Errorf("expected 16 tools, got %d", len(tools))
	}
}

func TestMCPProtocol_ToolsCall(t *testing.T) {
	server, _ := setupTestServer(t)

	// First add a task, then call tk_list
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tk_add","arguments":{"description":"Test task"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"tk_list","arguments":{}}}
`)
	output := &bytes.Buffer{}

	server.in = input
	server.out = output

	server.Run()

	// Parse both responses
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}

	var resp1, resp2 Response
	json.Unmarshal([]byte(lines[0]), &resp1)
	json.Unmarshal([]byte(lines[1]), &resp2)

	// Check first response (add)
	if resp1.Error != nil {
		t.Errorf("add error: %v", resp1.Error)
	}

	// Check second response (list)
	if resp2.Error != nil {
		t.Errorf("list error: %v", resp2.Error)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
