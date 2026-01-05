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
		"tk_archive", "tk_archive_restore", "tk_archive_list",
		"tk_show", "tk_delete", "tk_edit", "tk_pause", "tk_unblock",
		"tk_find", "tk_priority", "tk_owner", "tk_claim", "tk_release",
		"tk_suggest",
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
		input          string
		expectedPrefix string
	}{
		{"Fix authentication bug", "fix-authentication-bug-"},
		{"Add logout button", "add-logout-button-"},
		{"UPPERCASE TEST", "uppercase-test-"},
		{"Multiple   Spaces", "multiple-spaces-"},
		{"Special!@#Characters", "specialcharacters-"},
		{"", "task-"}, // Empty string generates task-xxx
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateID(tt.input)
			if !strings.HasPrefix(result, tt.expectedPrefix) {
				t.Errorf("generateID(%q) = %q, expected prefix %q",
					tt.input, result, tt.expectedPrefix)
			}
		})
	}

	// Test uniqueness - same description should produce different IDs
	id1 := generateID("Same description")
	id2 := generateID("Same description")
	if id1 == id2 {
		t.Errorf("generateID should produce unique IDs, but got same: %s", id1)
	}
}

func TestGenerateID_TruncatesLongDescriptions(t *testing.T) {
	long := "This is a very long description that exceeds thirty two characters"
	result := generateID(long)

	// Max 24 chars for description + 1 for hyphen + 3 for suffix = 28 chars max
	if len(result) > 28 {
		t.Errorf("generateID should truncate to ~28 chars, got %d: %s", len(result), result)
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

	if len(tools) != 30 {
		t.Errorf("expected 30 tools, got %d", len(tools))
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

func TestHandleToolCall_Archive(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Test task for archive",
		"id":          "archive-test",
	})

	// Complete the task
	server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "archive-test",
	})

	// Archive the task
	result, err := server.HandleToolCall("tk_archive", map[string]interface{}{
		"task_id": "archive-test",
		"summary": "Completed successfully",
	})
	if err != nil {
		t.Fatalf("archive error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "archived" {
		t.Errorf("expected status 'archived', got %s", r["status"])
	}
	if r["id"] != "archive-test" {
		t.Errorf("expected id 'archive-test', got %s", r["id"])
	}

	// List archived tasks
	listResult, err := server.HandleToolCall("tk_archive_list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("archive_list error: %v", err)
	}

	// Convert to JSON and back to verify structure
	data, _ := json.Marshal(listResult)
	var list []map[string]interface{}
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("failed to unmarshal archive list: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("expected 1 archived task, got %d", len(list))
	}

	// Verify archived task has expected fields
	if len(list) > 0 {
		if list[0]["id"] != "archive-test" {
			t.Errorf("expected archived task id 'archive-test', got %v", list[0]["id"])
		}
		if list[0]["summary"] != "Completed successfully" {
			t.Errorf("expected summary 'Completed successfully', got %v", list[0]["summary"])
		}
	}

	// Restore the task
	restoreResult, err := server.HandleToolCall("tk_archive_restore", map[string]interface{}{
		"task_id": "archive-test",
	})
	if err != nil {
		t.Fatalf("archive_restore error: %v", err)
	}

	rr := restoreResult.(map[string]string)
	if rr["status"] != "restored" {
		t.Errorf("expected status 'restored', got %s", rr["status"])
	}

	// Verify task is back in active tasks
	listActiveResult, err := server.HandleToolCall("tk_list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}

	data, _ = json.Marshal(listActiveResult)
	var activeTasks []map[string]interface{}
	json.Unmarshal(data, &activeTasks)

	found := false
	for _, task := range activeTasks {
		if task["id"] == "archive-test" {
			found = true
			if task["status"] != "ready" {
				t.Errorf("expected restored task status 'ready', got %v", task["status"])
			}
			break
		}
	}
	if !found {
		t.Error("restored task not found in active tasks")
	}
}

func TestHandleToolCall_Show(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Test task for show",
		"id":          "show-test",
	})

	// Add a note to the task
	server.HandleToolCall("tk_note", map[string]interface{}{
		"task_id": "show-test",
		"note":    "Test note",
	})

	// Show the task
	result, err := server.HandleToolCall("tk_show", map[string]interface{}{
		"id": "show-test",
	})
	if err != nil {
		t.Fatalf("show error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["id"] != "show-test" {
		t.Errorf("expected id 'show-test', got %v", r["id"])
	}
	if r["description"] != "Test task for show" {
		t.Errorf("expected description 'Test task for show', got %v", r["description"])
	}
	if r["status"] != "ready" {
		t.Errorf("expected status 'ready', got %v", r["status"])
	}

	// Verify notes are included
	notes, ok := r["notes"].([]interface{})
	if !ok {
		// Try the typed version
		_, ok = r["notes"]
		if !ok {
			t.Error("expected notes field")
		}
	} else if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
}

func TestHandleToolCall_Delete(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add two tasks
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Task to delete",
		"id":          "delete-test",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Task that blocks",
		"id":          "blocker-test",
	})

	// Block one task with another
	server.HandleToolCall("tk_block", map[string]interface{}{
		"id":         "delete-test",
		"blocked_by": []interface{}{"blocker-test"},
	})

	// Delete the blocker
	result, err := server.HandleToolCall("tk_delete", map[string]interface{}{
		"id": "blocker-test",
	})
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "deleted" {
		t.Errorf("expected status 'deleted', got %s", r["status"])
	}

	// Verify the blocked task is now unblocked (ready)
	showResult, _ := server.HandleToolCall("tk_show", map[string]interface{}{
		"id": "delete-test",
	})
	sr := showResult.(map[string]interface{})
	if sr["status"] != "ready" {
		t.Errorf("expected blocked task to become ready after blocker deleted, got %v", sr["status"])
	}
}

func TestHandleToolCall_Edit(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Original description",
		"id":          "edit-test",
	})

	// Edit the task
	result, err := server.HandleToolCall("tk_edit", map[string]interface{}{
		"id":          "edit-test",
		"description": "Updated description",
	})
	if err != nil {
		t.Fatalf("edit error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "updated" {
		t.Errorf("expected status 'updated', got %s", r["status"])
	}
	if r["description"] != "Updated description" {
		t.Errorf("expected description 'Updated description', got %s", r["description"])
	}

	// Verify with show
	showResult, _ := server.HandleToolCall("tk_show", map[string]interface{}{
		"id": "edit-test",
	})
	sr := showResult.(map[string]interface{})
	if sr["description"] != "Updated description" {
		t.Errorf("show returned wrong description: %v", sr["description"])
	}
}

func TestHandleToolCall_Pause(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add and start a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Task to pause",
		"id":          "pause-test",
	})
	server.HandleToolCall("tk_start", map[string]interface{}{
		"id": "pause-test",
	})

	// Pause the task
	result, err := server.HandleToolCall("tk_pause", map[string]interface{}{
		"id": "pause-test",
	})
	if err != nil {
		t.Fatalf("pause error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "ready" {
		t.Errorf("expected status 'ready', got %s", r["status"])
	}

	// Verify with show
	showResult, _ := server.HandleToolCall("tk_show", map[string]interface{}{
		"id": "pause-test",
	})
	sr := showResult.(map[string]interface{})
	if sr["status"] != "ready" {
		t.Errorf("show returned wrong status: %v", sr["status"])
	}
}

func TestHandleToolCall_Unblock(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add tasks
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "blocked-task",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "blocker-1",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "blocker-2",
	})

	// Block task
	server.HandleToolCall("tk_block", map[string]interface{}{
		"id":         "blocked-task",
		"blocked_by": []interface{}{"blocker-1", "blocker-2"},
	})

	// Partial unblock
	result, err := server.HandleToolCall("tk_unblock", map[string]interface{}{
		"id":   "blocked-task",
		"from": "blocker-1",
	})
	if err != nil {
		t.Fatalf("unblock error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["removed"] != "blocker-1" {
		t.Errorf("expected removed 'blocker-1', got %v", r["removed"])
	}

	// Full unblock
	result2, err := server.HandleToolCall("tk_unblock", map[string]interface{}{
		"id": "blocked-task",
	})
	if err != nil {
		t.Fatalf("full unblock error: %v", err)
	}

	r2 := result2.(map[string]interface{})
	if r2["status"] != "ready" {
		t.Errorf("expected status 'ready', got %v", r2["status"])
	}
}

func TestHandleToolCall_Find(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add tasks with various descriptions
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Fix authentication bug",
		"id":          "auth-fix",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Add new feature",
		"id":          "new-feature",
	})
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Authentication requires special handling",
	})

	// Search for "auth"
	result, err := server.HandleToolCall("tk_find", map[string]interface{}{
		"query": "auth",
	})
	if err != nil {
		t.Fatalf("find error: %v", err)
	}

	// Convert to check results
	data, _ := json.Marshal(result)
	var results []map[string]interface{}
	json.Unmarshal(data, &results)

	if len(results) < 2 {
		t.Errorf("expected at least 2 results (task + learning), got %d", len(results))
	}

	// Verify we found both task and learning
	foundTask := false
	foundLearning := false
	for _, r := range results {
		if r["type"] == "task" && r["id"] == "auth-fix" {
			foundTask = true
		}
		if r["type"] == "learning" {
			foundLearning = true
		}
	}
	if !foundTask {
		t.Error("expected to find auth-fix task")
	}
	if !foundLearning {
		t.Error("expected to find learning about authentication")
	}
}

func TestHandleToolCall_Priority(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Priority test",
		"id":          "priority-test",
	})

	// Set priority with string
	result, err := server.HandleToolCall("tk_priority", map[string]interface{}{
		"id":       "priority-test",
		"priority": "critical",
	})
	if err != nil {
		t.Fatalf("priority error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["status"] != "updated" {
		t.Errorf("expected status 'updated', got %v", r["status"])
	}
	if r["priority"] != float64(0) && r["priority"] != 0 {
		t.Errorf("expected priority 0 (critical), got %v", r["priority"])
	}

	// Set priority with number
	result2, err := server.HandleToolCall("tk_priority", map[string]interface{}{
		"id":       "priority-test",
		"priority": "3",
	})
	if err != nil {
		t.Fatalf("priority error: %v", err)
	}

	r2 := result2.(map[string]interface{})
	if r2["priority"] != float64(3) && r2["priority"] != 3 {
		t.Errorf("expected priority 3 (low), got %v", r2["priority"])
	}
}

func TestHandleToolCall_Owner(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Owner test",
		"id":          "owner-test",
	})

	// Set owner
	result, err := server.HandleToolCall("tk_owner", map[string]interface{}{
		"id":    "owner-test",
		"owner": "agent-1",
	})
	if err != nil {
		t.Fatalf("owner set error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "owner_set" {
		t.Errorf("expected status 'owner_set', got %s", r["status"])
	}
	if r["owner"] != "agent-1" {
		t.Errorf("expected owner 'agent-1', got %s", r["owner"])
	}

	// Clear owner
	result2, err := server.HandleToolCall("tk_owner", map[string]interface{}{
		"id": "owner-test",
	})
	if err != nil {
		t.Fatalf("owner clear error: %v", err)
	}

	r2 := result2.(map[string]string)
	if r2["status"] != "owner_cleared" {
		t.Errorf("expected status 'owner_cleared', got %s", r2["status"])
	}
}

func TestHandleToolCall_ClaimRelease(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Claim test",
		"id":          "claim-test",
	})

	// Claim the task
	result, err := server.HandleToolCall("tk_claim", map[string]interface{}{
		"id":    "claim-test",
		"agent": "agent-1",
	})
	if err != nil {
		t.Fatalf("claim error: %v", err)
	}

	r := result.(map[string]string)
	if r["status"] != "claimed" {
		t.Errorf("expected status 'claimed', got %s", r["status"])
	}
	if r["agent"] != "agent-1" {
		t.Errorf("expected agent 'agent-1', got %s", r["agent"])
	}

	// Release the task
	result2, err := server.HandleToolCall("tk_release", map[string]interface{}{
		"id": "claim-test",
	})
	if err != nil {
		t.Fatalf("release error: %v", err)
	}

	r2 := result2.(map[string]string)
	if r2["status"] != "released" {
		t.Errorf("expected status 'released', got %s", r2["status"])
	}
}

func TestSuggest(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name          string
		description   string
		shouldPersist bool
	}{
		// Project-level tasks (should persist)
		{"implement feature", "Implement user authentication", true},
		{"add feature", "Add feature for dark mode", true},
		{"fix bug", "Fix bug in login flow", true},
		{"refactor", "Refactor the database layer", true},
		{"migrate", "Migrate to new API version", true},
		{"build", "Build the dashboard component", true},
		{"deploy", "Deploy to production", true},
		{"database work", "Add database index for users table", true},
		{"security", "Implement security headers", true},
		{"api endpoint", "Create API endpoint for user profile", true},

		// Session-level tasks (should not persist)
		{"fix type error", "Fix type error in auth.ts", false},
		{"fix typo", "Fix typo in error message", false},
		{"update file", "Update file imports", false},
		{"run tests", "Run test suite", false},
		{"debug", "Debug the failing test", false},
		{"verify", "Verify the output", false},
		{"add import", "Add import statement", false},
		{"rename variable", "Rename variable to be clearer", false},

		// Edge cases
		{"no keywords", "Do something", false},
		{"mixed - session wins", "Fix type error while implementing auth", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.HandleToolCall("tk_suggest", map[string]interface{}{
				"description": tt.description,
			})
			if err != nil {
				t.Fatalf("suggest error: %v", err)
			}

			r := result.(map[string]interface{})
			shouldPersist, ok := r["should_persist"].(bool)
			if !ok {
				t.Fatalf("should_persist not a bool")
			}

			if shouldPersist != tt.shouldPersist {
				t.Errorf("for %q: expected should_persist=%v, got %v (reason: %s)",
					tt.description, tt.shouldPersist, shouldPersist, r["reason"])
			}

			// Verify other fields are present
			if _, ok := r["reason"].(string); !ok {
				t.Error("missing reason field")
			}
			if _, ok := r["recommendation"].(string); !ok {
				t.Error("missing recommendation field")
			}
			if _, ok := r["original_description"].(string); !ok {
				t.Error("missing original_description field")
			}

			// Verify suggested_command is present when should persist
			if shouldPersist {
				if _, ok := r["suggested_command"].(string); !ok {
					t.Error("missing suggested_command for persistent task")
				}
			}
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
