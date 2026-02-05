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

	// 17 tools total: 12 core + 3 action-based + 2 utility
	expectedTools := []string{
		// Tier 1: Core tools (kept individual)
		"tk_help",
		"tk_list",
		"tk_add",
		"tk_start",
		"tk_done",
		"tk_block", // Kept individual - frequently used for blocking tasks
		"tk_learn",
		"tk_decide",
		"tk_note",
		"tk_context",
		"tk_show",
		"tk_find",
		// Tier 2: Consolidated tools
		"tk_task",     // edit, delete, pause, unblock, priority, owner, archive, restore, claim, release, who
		"tk_metadata", // tag_add, tag_remove, field_set, field_remove, note_list, note_remove
		"tk_manage",   // learning_list, learning_promote, learning_remove, learning_rules, decision_list, decision_remove, archive_list, archive_all
		// Tier 3: Less frequent but useful tools
		"tk_stats",
		"tk_health",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
		// Print actual tool names for debugging
		for _, tool := range tools {
			t.Logf("  got tool: %s", tool.Name)
		}
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

	r, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}

	if r["status"] != "created" {
		t.Errorf("expected status 'created', got %v", r["status"])
	}

	if r["id"] == "" || r["id"] == nil {
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

	r := result.(map[string]interface{})
	if r["id"] != "custom-id" {
		t.Errorf("expected id 'custom-id', got %v", r["id"])
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

	r := result.(map[string]interface{})
	if r["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %v", r["status"])
	}

	// Complete it
	result, err = server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "test-task",
	})
	if err != nil {
		t.Fatalf("done error: %v", err)
	}

	r = result.(map[string]interface{})
	if r["status"] != "done" {
		t.Errorf("expected status 'done', got %v", r["status"])
	}
}

func TestHandleToolCall_DoneBugFix(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a bug-fix task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Fix overlay rendering bug",
		"id":          "fix-overlay",
	})

	// Start and complete it
	server.HandleToolCall("tk_start", map[string]interface{}{
		"id": "fix-overlay",
	})

	result, err := server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "fix-overlay",
	})
	if err != nil {
		t.Fatalf("done error: %v", err)
	}

	r := result.(map[string]interface{})

	// Should detect as bug fix
	if r["is_bug_fix"] != true {
		t.Errorf("expected is_bug_fix=true for bug fix task")
	}

	// Should have hints including bug-fix specific prompts
	hints, ok := r["hints"].([]string)
	if !ok {
		t.Fatal("expected hints to be []string")
	}

	foundBugFixHint := false
	for _, h := range hints {
		if strings.Contains(h, "BUG FIX COMPLETED") {
			foundBugFixHint = true
			break
		}
	}
	if !foundBugFixHint {
		t.Errorf("expected bug fix learning prompt in hints, got: %v", hints)
	}
}

func TestHandleToolCall_DoneNonBugFix(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a regular task (not bug-fix)
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Add logout button",
		"id":          "add-logout",
	})

	// Start and complete it
	server.HandleToolCall("tk_start", map[string]interface{}{
		"id": "add-logout",
	})

	result, err := server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "add-logout",
	})
	if err != nil {
		t.Fatalf("done error: %v", err)
	}

	r := result.(map[string]interface{})

	// Should NOT be detected as bug fix
	if r["is_bug_fix"] == true {
		t.Errorf("expected is_bug_fix to NOT be set for non-bug-fix task")
	}

	// Should have hints with REFLECT prompt instead
	hints, ok := r["hints"].([]string)
	if !ok {
		t.Fatal("expected hints to be []string")
	}

	foundReflect := false
	for _, h := range hints {
		if strings.Contains(h, "REFLECT") {
			foundReflect = true
			break
		}
	}
	if !foundReflect {
		t.Errorf("expected REFLECT prompt in hints for non-bug-fix task, got: %v", hints)
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

	r := result.(map[string]interface{})
	if r["status"] != "recorded" {
		t.Errorf("expected status 'recorded', got %v", r["status"])
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

	// Empty string generates task-xxx with random suffix
	emptyResult := generateID("")
	if !strings.HasPrefix(emptyResult, "task-") {
		t.Errorf("generateID(\"\") = %q, expected prefix \"task-\"", emptyResult)
	}
}

func TestGenerateID_TruncatesAt32(t *testing.T) {
	long := "This is a very long description that exceeds thirty two characters"
	result := generateID(long)

	if len(result) > 32 {
		t.Errorf("generateID should truncate to 32 chars, got %d: %s", len(result), result)
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

	if len(tools) != 17 {
		t.Errorf("expected 17 tools, got %d", len(tools))
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action":  "archive",
		"id":      "archive-test",
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
	listResult, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "archive_list",
	})
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
	restoreResult, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "restore",
		"id":     "archive-test",
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

func TestHandleToolCall_ListArchivedFilters(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add tasks with different tags
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Bug task",
		"id":          "bug-task",
		"tags":        []interface{}{"bug"},
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Feature task",
		"id":          "feature-task",
		"tags":        []interface{}{"feature"},
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Another bug task",
		"id":          "another-bug-task",
		"tags":        []interface{}{"bug"},
	})

	// Complete and archive one bug task and one feature task
	server.HandleToolCall("tk_done", map[string]interface{}{"id": "bug-task"})
	server.HandleToolCall("tk_done", map[string]interface{}{"id": "feature-task"})
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "archive",
		"id":     "bug-task",
	})
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "archive",
		"id":     "feature-task",
	})

	// Test 1: List with tag=bug and include_archived=true
	// Should return: another-bug-task (active) + bug-task (archived)
	// Should NOT return: feature-task (archived, wrong tag)
	result, err := server.HandleToolCall("tk_list", map[string]interface{}{
		"tag":              "bug",
		"include_archived": true,
	})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}

	data, _ := json.Marshal(result)
	var tasks []map[string]interface{}
	json.Unmarshal(data, &tasks)

	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks with tag=bug (1 active + 1 archived), got %d", len(tasks))
		for _, task := range tasks {
			t.Logf("  task: %v, status: %v, tags: %v", task["id"], task["status"], task["tags"])
		}
	}

	// Verify the correct tasks are returned
	foundAnotherBug := false
	foundBugTask := false
	foundFeatureTask := false
	for _, task := range tasks {
		switch task["id"] {
		case "another-bug-task":
			foundAnotherBug = true
		case "bug-task":
			foundBugTask = true
			if task["status"] != "archived" {
				t.Errorf("bug-task should have status 'archived', got %v", task["status"])
			}
		case "feature-task":
			foundFeatureTask = true
		}
	}

	if !foundAnotherBug {
		t.Error("expected to find 'another-bug-task' (active task with bug tag)")
	}
	if !foundBugTask {
		t.Error("expected to find 'bug-task' (archived task with bug tag)")
	}
	if foundFeatureTask {
		t.Error("should NOT find 'feature-task' (archived task with wrong tag)")
	}

	// Test 2: List with status=in_progress and include_archived=true
	// Should NOT return any archived tasks (they have status 'archived', not 'in_progress')
	result2, err := server.HandleToolCall("tk_list", map[string]interface{}{
		"status":           "in_progress",
		"include_archived": true,
	})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}

	data2, _ := json.Marshal(result2)
	var tasks2 []map[string]interface{}
	json.Unmarshal(data2, &tasks2)

	for _, task := range tasks2 {
		if task["status"] == "archived" {
			t.Errorf("should not return archived tasks when status=in_progress, got task %v", task["id"])
		}
	}

	// Test 3: List with status=archived and include_archived=true
	// Should return only the archived tasks
	result3, err := server.HandleToolCall("tk_list", map[string]interface{}{
		"status":           "archived",
		"include_archived": true,
	})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}

	data3, _ := json.Marshal(result3)
	var tasks3 []map[string]interface{}
	json.Unmarshal(data3, &tasks3)

	// Should have 2 archived tasks
	if len(tasks3) != 2 {
		t.Errorf("expected 2 tasks with status=archived, got %d", len(tasks3))
	}
	for _, task := range tasks3 {
		if task["status"] != "archived" {
			t.Errorf("expected all tasks to have status 'archived', got %v for task %v", task["status"], task["id"])
		}
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "delete",
		"id":     "blocker-test",
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action":      "edit",
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "pause",
		"id":     "pause-test",
	})
	if err != nil {
		t.Fatalf("pause error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["status"] != "ready" {
		t.Errorf("expected status 'ready', got %v", r["status"])
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "unblock",
		"id":     "blocked-task",
		"from":   "blocker-1",
	})
	if err != nil {
		t.Fatalf("unblock error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["removed"] != "blocker-1" {
		t.Errorf("expected removed 'blocker-1', got %v", r["removed"])
	}

	// Full unblock
	result2, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "unblock",
		"id":     "blocked-task",
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action":   "priority",
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
	result2, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action":   "priority",
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "owner",
		"id":     "owner-test",
		"owner":  "agent-1",
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
	result2, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "owner",
		"id":     "owner-test",
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
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "claim",
		"id":     "claim-test",
		"agent":  "agent-1",
	})
	if err != nil {
		t.Fatalf("claim error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["status"] != "claimed" {
		t.Errorf("expected status 'claimed', got %v", r["status"])
	}
	if r["agent"] != "agent-1" {
		t.Errorf("expected agent 'agent-1', got %v", r["agent"])
	}

	// Release the task
	result2, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "release",
		"id":     "claim-test",
	})
	if err != nil {
		t.Fatalf("release error: %v", err)
	}

	r2 := result2.(map[string]interface{})
	if r2["status"] != "released" {
		t.Errorf("expected status 'released', got %v", r2["status"])
	}
}

func TestHandleToolCall_FieldSetRemove(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Field test task",
		"id":          "field-test",
	})

	// Set field
	result, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "field_set",
		"id":     "field-test",
		"key":    "estimate",
		"value":  "2h",
	})
	if err != nil {
		t.Fatalf("field_set error: %v", err)
	}

	data, _ := json.Marshal(result)
	var r map[string]interface{}
	json.Unmarshal(data, &r)
	if r["status"] != "set" {
		t.Errorf("expected status 'set', got %v", r["status"])
	}

	// Verify field is set via show
	showResult, _ := server.HandleToolCall("tk_show", map[string]interface{}{
		"id": "field-test",
	})
	data, _ = json.Marshal(showResult)
	var sr map[string]interface{}
	json.Unmarshal(data, &sr)
	fields, ok := sr["fields"].(map[string]interface{})
	if !ok {
		t.Fatal("expected fields to be a map")
	}
	if fields["estimate"] != "2h" {
		t.Errorf("expected estimate '2h', got %v", fields["estimate"])
	}

	// Remove field
	removeResult, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "field_remove",
		"id":     "field-test",
		"key":    "estimate",
	})
	if err != nil {
		t.Fatalf("field_remove error: %v", err)
	}

	data, _ = json.Marshal(removeResult)
	var rr map[string]interface{}
	json.Unmarshal(data, &rr)
	if rr["status"] != "removed" {
		t.Errorf("expected status 'removed', got %v", rr["status"])
	}
}

func TestHandleToolCall_TagAddRemove(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Tag test task",
		"id":          "tag-test",
	})

	// Add tag
	result, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "tag_add",
		"id":     "tag-test",
		"tag":    "urgent",
	})
	if err != nil {
		t.Fatalf("tag_add error: %v", err)
	}

	data, _ := json.Marshal(result)
	var r map[string]interface{}
	json.Unmarshal(data, &r)
	if r["status"] != "added" {
		t.Errorf("expected status 'added', got %v", r["status"])
	}

	// Verify tag is set via show
	showResult, _ := server.HandleToolCall("tk_show", map[string]interface{}{
		"id": "tag-test",
	})
	data, _ = json.Marshal(showResult)
	var sr map[string]interface{}
	json.Unmarshal(data, &sr)
	tags, ok := sr["tags"].([]interface{})
	if !ok {
		t.Fatal("expected tags to be an array")
	}
	foundTag := false
	for _, tag := range tags {
		if tag == "urgent" {
			foundTag = true
			break
		}
	}
	if !foundTag {
		t.Error("expected 'urgent' tag to be set")
	}

	// Remove tag
	removeResult, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "tag_remove",
		"id":     "tag-test",
		"tag":    "urgent",
	})
	if err != nil {
		t.Fatalf("tag_remove error: %v", err)
	}

	data, _ = json.Marshal(removeResult)
	var rr map[string]interface{}
	json.Unmarshal(data, &rr)
	if rr["status"] != "removed" {
		t.Errorf("expected status 'removed', got %v", rr["status"])
	}
}

func TestHandleToolCall_ArchiveAll(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add and complete two tasks
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Archive all test 1",
		"id":          "archive-all-1",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Archive all test 2",
		"id":          "archive-all-2",
	})
	server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "archive-all-1",
	})
	server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "archive-all-2",
	})

	// Archive all done tasks older than 0 hours (i.e., all done tasks)
	result, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action":     "archive_all",
		"older_than": "0h",
	})
	if err != nil {
		t.Fatalf("archive_all error: %v", err)
	}

	data, _ := json.Marshal(result)
	var r map[string]interface{}
	json.Unmarshal(data, &r)

	count, _ := r["archived_count"].(float64)
	if int(count) != 2 {
		t.Errorf("expected archived_count 2, got %v", count)
	}
}

func TestParseDurationString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64 // in nanoseconds
		hasError bool
	}{
		{"1h", int64(3600 * 1e9), false},
		{"24h", int64(24 * 3600 * 1e9), false},
		{"7d", int64(7 * 24 * 3600 * 1e9), false},
		{"2w", int64(14 * 24 * 3600 * 1e9), false},
		{"invalid", 0, true},
		{"", 0, true},
		{"30m", 0, true}, // only h, d, w are supported
		{"60s", 0, true}, // only h, d, w are supported
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dur, err := parseDurationString(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.input, err)
				}
				if int64(dur) != tt.expected {
					t.Errorf("parseDurationString(%q) = %v, expected %v", tt.input, dur, tt.expected)
				}
			}
		})
	}
}

func TestHandleToolCall_Ready(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add tasks with different priorities and statuses
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "High priority task",
		"id":          "high-pri",
	})
	// Set priority
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action":   "priority",
		"id":       "high-pri",
		"priority": "high",
	})

	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Normal priority task",
		"id":          "normal-pri",
	})

	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Started task",
		"id":          "started",
	})
	server.HandleToolCall("tk_start", map[string]interface{}{
		"id": "started",
	})

	// Get ready tasks using tk_list with status filter
	result, err := server.HandleToolCall("tk_list", map[string]interface{}{
		"status": "ready",
	})
	if err != nil {
		t.Fatalf("ready error: %v", err)
	}

	data, _ := json.Marshal(result)
	var tasks []map[string]interface{}
	json.Unmarshal(data, &tasks)

	// Should only show ready tasks (not in_progress)
	for _, task := range tasks {
		if task["status"] == "in_progress" {
			t.Error("ready list should not contain in_progress tasks")
		}
	}

	// High priority should be first (if multiple ready tasks)
	if len(tasks) >= 2 {
		if tasks[0]["id"] != "high-pri" {
			t.Errorf("expected high-pri to be first, got %v", tasks[0]["id"])
		}
	}
}

func TestHandleToolCall_Who(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add and claim tasks
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "who-test-1",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "who-test-2",
	})
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "claim",
		"id":     "who-test-1",
		"agent":  "agent-1",
	})
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "claim",
		"id":     "who-test-2",
		"agent":  "agent-2",
	})

	// Check who
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "who",
	})
	if err != nil {
		t.Fatalf("who error: %v", err)
	}

	// Result should be a map of agents to their tasks
	data, _ := json.Marshal(result)
	var who map[string]interface{}
	json.Unmarshal(data, &who)

	if who["agent-1"] == nil {
		t.Error("expected agent-1 in who list")
	}
	if who["agent-2"] == nil {
		t.Error("expected agent-2 in who list")
	}
}

func TestHandleToolCall_Stats(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add tasks with different statuses
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "stats-ready",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "stats-progress",
	})
	server.HandleToolCall("tk_start", map[string]interface{}{
		"id": "stats-progress",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "stats-done",
	})
	server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "stats-done",
	})

	// Get stats
	result, err := server.HandleToolCall("tk_stats", map[string]interface{}{})
	if err != nil {
		t.Fatalf("stats error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["total"] == nil {
		t.Error("expected total in stats")
	}
	if r["by_status"] == nil {
		t.Error("expected by_status in stats")
	}
}

func TestHandleToolCall_LearningOperations(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add learnings
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Regular learning",
	})
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Never do X",
	})

	// List learnings
	listResult, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_list",
	})
	if err != nil {
		t.Fatalf("learning_list error: %v", err)
	}

	data, _ := json.Marshal(listResult)
	var learnings []map[string]interface{}
	json.Unmarshal(data, &learnings)

	if len(learnings) != 2 {
		t.Errorf("expected 2 learnings, got %d", len(learnings))
	}

	// List rules only - response is {"rules": [...], "count": ..., "recommendation": ...}
	rulesResult, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_rules",
	})
	if err != nil {
		t.Fatalf("learning_rules error: %v", err)
	}

	data, _ = json.Marshal(rulesResult)
	var rulesResp map[string]interface{}
	json.Unmarshal(data, &rulesResp)

	// Should only have the "Never" rule
	count, _ := rulesResp["count"].(float64)
	if int(count) != 1 {
		t.Errorf("expected 1 rule learning, got %d (response: %s)", int(count), string(data))
	}
}

func TestHandleToolCall_DecisionOperations(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add decisions
	server.HandleToolCall("tk_decide", map[string]interface{}{
		"id":      "dec-1",
		"chose":   "A",
		"over":    []interface{}{"B"},
		"because": "reason",
	})
	server.HandleToolCall("tk_decide", map[string]interface{}{
		"id":      "dec-2",
		"chose":   "X",
		"over":    []interface{}{"Y"},
		"because": "other reason",
	})

	// List decisions
	listResult, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "decision_list",
	})
	if err != nil {
		t.Fatalf("decision_list error: %v", err)
	}

	data, _ := json.Marshal(listResult)
	var decisions []map[string]interface{}
	json.Unmarshal(data, &decisions)

	if len(decisions) != 2 {
		t.Errorf("expected 2 decisions, got %d", len(decisions))
	}

	// Remove decision
	removeResult, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "decision_remove",
		"id":     "dec-1",
	})
	if err != nil {
		t.Fatalf("decision_remove error: %v", err)
	}

	r := removeResult.(map[string]interface{})
	if r["status"] != "removed" {
		t.Errorf("expected status 'removed', got %v", r["status"])
	}
}

func TestHandleToolCall_NoteOperations(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task with notes
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "note-ops-test",
	})
	server.HandleToolCall("tk_note", map[string]interface{}{
		"task_id": "note-ops-test",
		"note":    "First note",
	})
	server.HandleToolCall("tk_note", map[string]interface{}{
		"task_id": "note-ops-test",
		"note":    "Second note",
	})

	// List notes for task
	listResult, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action":  "note_list",
		"task_id": "note-ops-test",
	})
	if err != nil {
		t.Fatalf("note_list error: %v", err)
	}

	data, _ := json.Marshal(listResult)
	var notes []map[string]interface{}
	json.Unmarshal(data, &notes)

	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
	}

	// Get note ID from first note
	noteID := ""
	if len(notes) > 0 {
		noteID, _ = notes[0]["id"].(string)
	}

	// Remove note
	if noteID != "" {
		removeResult, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
			"action":  "note_remove",
			"task_id": "note-ops-test",
			"note_id": noteID,
		})
		if err != nil {
			t.Fatalf("note_remove error: %v", err)
		}

		r := removeResult.(map[string]interface{})
		if r["status"] != "removed" {
			t.Errorf("expected status 'removed', got %v", r["status"])
		}
	}
}

func TestHandleToolCall_Deps(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create task dependency chain: A blocked by B, B blocked by C
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "dep-a",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "dep-b",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "dep-c",
	})
	server.HandleToolCall("tk_block", map[string]interface{}{
		"id":         "dep-a",
		"blocked_by": []interface{}{"dep-b"},
	})
	server.HandleToolCall("tk_block", map[string]interface{}{
		"id":         "dep-b",
		"blocked_by": []interface{}{"dep-c"},
	})

	// Get deps for A using tk_show (deps are now included in show output)
	result, err := server.HandleToolCall("tk_show", map[string]interface{}{
		"id": "dep-a",
	})
	if err != nil {
		t.Fatalf("show error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["id"] != "dep-a" {
		t.Errorf("expected id 'dep-a', got %v", r["id"])
	}
	if r["blocked_by"] == nil {
		t.Error("expected blocked_by in show result")
	}
}

func TestHandleToolCall_Health(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add some data
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "health-test",
	})
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Never do this",
	})

	// Get health
	result, err := server.HandleToolCall("tk_health", map[string]interface{}{})
	if err != nil {
		t.Fatalf("health error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["task_counts"] == nil {
		t.Error("expected task_counts in health result")
	}
	if r["recommendations"] == nil {
		t.Error("expected recommendations in health result")
	}
}

func TestHandleToolCall_StartWithUnblock(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create blocked task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "blocked-start",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "blocker",
	})
	server.HandleToolCall("tk_block", map[string]interface{}{
		"id":         "blocked-start",
		"blocked_by": []interface{}{"blocker"},
	})

	// Try to start blocked task with unblock flag
	result, err := server.HandleToolCall("tk_start", map[string]interface{}{
		"id":      "blocked-start",
		"unblock": true,
	})
	if err != nil {
		t.Fatalf("start with unblock error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %v", r["status"])
	}
}

func TestHandleToolCall_LearningPromote(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a learning first
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Never use global variables in tests",
	})

	// List learnings to get the ID - use JSON to handle internal types
	listResult, _ := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_list",
	})
	jsonData, _ := json.Marshal(listResult)
	var learnings []map[string]interface{}
	json.Unmarshal(jsonData, &learnings)
	if len(learnings) == 0 {
		t.Fatal("expected at least one learning")
	}
	learningID := learnings[0]["id"].(string)

	// Create CLAUDE.md for promotion target
	dir := filepath.Dir(server.store.Path())
	claudemdPath := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(claudemdPath, []byte("# Project\n\n## Learnings\n"), 0644)

	// Change to directory so detectContextFile works
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Promote the learning
	result, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_promote",
		"id":     learningID,
	})
	if err != nil {
		t.Fatalf("promote error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["id"] != learningID {
		t.Errorf("expected id '%s', got %v", learningID, r["id"])
	}
	if r["promoted_to"] == nil {
		t.Error("expected promoted_to in result")
	}
}

func TestHandleToolCall_LearningPromoteKeep(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a learning first
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Always check error returns",
	})

	// List learnings to get the ID - use JSON to handle internal types
	listResult, _ := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_list",
	})
	jsonData, _ := json.Marshal(listResult)
	var learnings []map[string]interface{}
	json.Unmarshal(jsonData, &learnings)
	learningID := learnings[0]["id"].(string)

	// Create CLAUDE.md for promotion target
	dir := filepath.Dir(server.store.Path())
	claudemdPath := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(claudemdPath, []byte("# Project\n\n## Learnings\n"), 0644)

	// Change to directory so detectContextFile works
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Promote with keep=true
	result, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_promote",
		"id":     learningID,
		"keep":   true,
	})
	if err != nil {
		t.Fatalf("promote error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["kept"] != true {
		t.Errorf("expected kept=true, got %v", r["kept"])
	}

	// Verify learning still exists
	listResult2, _ := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_list",
	})
	jsonData2, _ := json.Marshal(listResult2)
	var learnings2 []map[string]interface{}
	json.Unmarshal(jsonData2, &learnings2)
	if len(learnings2) == 0 {
		t.Error("expected learning to still exist with keep=true")
	}
}

func TestHandleToolCall_LearningRemove(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a learning
	server.HandleToolCall("tk_learn", map[string]interface{}{
		"insight": "Test learning to remove",
	})

	// List learnings to get the ID - use JSON to handle internal types
	listResult, _ := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_list",
	})
	jsonData, _ := json.Marshal(listResult)
	var learnings []map[string]interface{}
	json.Unmarshal(jsonData, &learnings)
	learningID := learnings[0]["id"].(string)

	// Remove the learning
	result, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_remove",
		"id":     learningID,
	})
	if err != nil {
		t.Fatalf("remove error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["id"] != learningID {
		t.Errorf("expected id '%s', got %v", learningID, r["id"])
	}
	if r["removed"] == nil {
		t.Error("expected removed text in result")
	}

	// Verify learning is gone
	listResult2, _ := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_list",
	})
	jsonData2, _ := json.Marshal(listResult2)
	var learnings2 []map[string]interface{}
	json.Unmarshal(jsonData2, &learnings2)
	if len(learnings2) != 0 {
		t.Errorf("expected no learnings after remove, got %d", len(learnings2))
	}
}

func TestHandleToolCall_LearningRemoveNotFound(t *testing.T) {
	server, _ := setupTestServer(t)

	// Try to remove non-existent learning
	_, err := server.HandleToolCall("tk_manage", map[string]interface{}{
		"action": "learning_remove",
		"id":     "non-existent-id",
	})
	if err == nil {
		t.Error("expected error for non-existent learning")
	}
}

func TestHandleToolCall_NoteRemove(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task with a note
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Task with note",
		"id":          "note-remove-test",
	})
	noteResult, _ := server.HandleToolCall("tk_note", map[string]interface{}{
		"task_id": "note-remove-test",
		"note":    "Test note to remove",
	})
	// tk_note returns map[string]string with "id" as the note ID
	nr := noteResult.(map[string]string)
	noteID := nr["id"]

	// Remove the note
	result, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action":  "note_remove",
		"task_id": "note-remove-test",
		"note_id": noteID,
	})
	if err != nil {
		t.Fatalf("note remove error: %v", err)
	}

	// Result is map[string]string
	r := result.(map[string]string)
	if r["task_id"] != "note-remove-test" {
		t.Errorf("expected task_id 'note-remove-test', got %s", r["task_id"])
	}
	if r["note_id"] != noteID {
		t.Errorf("expected note_id '%s', got %s", noteID, r["note_id"])
	}
}

func TestHandleToolCall_NoteRemoveNotFound(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add a task
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Task without note",
		"id":          "note-remove-test-2",
	})

	// Try to remove non-existent note
	_, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action":  "note_remove",
		"task_id": "note-remove-test-2",
		"note_id": "non-existent-note",
	})
	if err == nil {
		t.Error("expected error for non-existent note")
	}
}

func TestCountFileLines(t *testing.T) {
	server, _ := setupTestServer(t)
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644)

	lines := server.countFileLines(testFile)
	if lines != 3 {
		t.Errorf("expected 3 lines, got %d", lines)
	}
}

func TestCountFileLinesEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	dir := t.TempDir()
	testFile := filepath.Join(dir, "empty.txt")
	os.WriteFile(testFile, []byte(""), 0644)

	lines := server.countFileLines(testFile)
	if lines != 0 {
		t.Errorf("expected 0 lines, got %d", lines)
	}
}

func TestCountFileLinesNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	lines := server.countFileLines("/nonexistent/file.txt")
	if lines != 0 {
		t.Errorf("expected 0 lines for non-existent file, got %d", lines)
	}
}

func TestCountRulesFiles(t *testing.T) {
	server, _ := setupTestServer(t)
	dir := filepath.Dir(server.store.Path())
	rulesDir := filepath.Join(dir, ".claude", "rules")
	os.MkdirAll(rulesDir, 0755)

	// Create some rule files
	os.WriteFile(filepath.Join(rulesDir, "rule1.md"), []byte("# Rule 1\n"), 0644)
	os.WriteFile(filepath.Join(rulesDir, "rule2.md"), []byte("# Rule 2\n"), 0644)
	os.WriteFile(filepath.Join(rulesDir, "not-md.txt"), []byte("not a rule\n"), 0644)

	// Change to the directory where .claude/rules is
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	count := server.countRulesFiles()
	if count != 2 {
		t.Errorf("expected 2 rule files, got %d", count)
	}
}

func TestCountRulesFilesEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	dir := filepath.Dir(server.store.Path())
	rulesDir := filepath.Join(dir, ".claude", "rules")
	os.MkdirAll(rulesDir, 0755)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	count := server.countRulesFiles()
	if count != 0 {
		t.Errorf("expected 0 rule files, got %d", count)
	}
}

func TestDetectContextFile(t *testing.T) {
	dir := t.TempDir()

	// Create CLAUDE.md
	claudemdPath := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(claudemdPath, []byte("# Project\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	path := detectContextFile()
	if path != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md, got %s", path)
	}
}

func TestDetectContextFileDefault(t *testing.T) {
	dir := t.TempDir()

	// Don't create any context file
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// When no context file exists, it defaults to CLAUDE.md
	path := detectContextFile()
	if path != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md as default, got %s", path)
	}
}

func TestAppendToContextFile(t *testing.T) {
	dir := t.TempDir()
	contextFile := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(contextFile, []byte("# Project\n\n## Learnings\n\n"), 0644)

	err := appendToContextFile(contextFile, "New learning content")
	if err != nil {
		t.Fatalf("append error: %v", err)
	}

	content, _ := os.ReadFile(contextFile)
	if !strings.Contains(string(content), "New learning content") {
		t.Error("expected appended content in file")
	}
}

func TestAppendToContextFileNewSection(t *testing.T) {
	dir := t.TempDir()
	contextFile := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(contextFile, []byte("# Project\n\nSome content.\n"), 0644)

	err := appendToContextFile(contextFile, "New learning that creates section")
	if err != nil {
		t.Fatalf("append error: %v", err)
	}

	content, _ := os.ReadFile(contextFile)
	if !strings.Contains(string(content), "## Learnings") {
		t.Error("expected Learnings section header in file")
	}
	if !strings.Contains(string(content), "New learning that creates section") {
		t.Error("expected new learning content in file")
	}
}

func TestSendError(t *testing.T) {
	var buf bytes.Buffer
	server := &Server{out: &buf}

	server.sendError(123, -32600, "Test error message", nil)

	var response struct {
		Jsonrpc string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(&buf).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Jsonrpc != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got %s", response.Jsonrpc)
	}
	if response.ID != 123 {
		t.Errorf("expected id 123, got %d", response.ID)
	}
	if response.Error.Code != -32600 {
		t.Errorf("expected error code -32600, got %d", response.Error.Code)
	}
	if response.Error.Message != "Test error message" {
		t.Errorf("expected error message 'Test error message', got %s", response.Error.Message)
	}
}

func TestMCPProtocol_NotificationProgress(t *testing.T) {
	server, _ := setupTestServer(t)

	// Send notifications/progress - should be silently ignored (no response)
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"abc","progress":50,"total":100}}
`)
	output := &bytes.Buffer{}

	server.in = input
	server.out = output

	server.Run()

	// No response expected for notifications
	if output.Len() != 0 {
		t.Errorf("expected no response for notification, got: %s", output.String())
	}
}

func TestMCPProtocol_NotificationRootsListChanged(t *testing.T) {
	server, _ := setupTestServer(t)

	// Send notifications/roots/list_changed - should be silently ignored
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"notifications/roots/list_changed"}
`)
	output := &bytes.Buffer{}

	server.in = input
	server.out = output

	server.Run()

	// No response expected for notifications
	if output.Len() != 0 {
		t.Errorf("expected no response for notification, got: %s", output.String())
	}
}

func TestMCPProtocol_BatchRequestRejected(t *testing.T) {
	server, _ := setupTestServer(t)

	// Send batch request (JSON array) - should be rejected with -32600
	input := bytes.NewBufferString(`[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","id":2,"method":"tools/list"}]
`)
	output := &bytes.Buffer{}

	server.in = input
	server.out = output

	server.Run()

	// Should get an error response
	var resp Response
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nOutput: %s", err, output.String())
	}

	if resp.Error == nil {
		t.Fatal("expected error response for batch request")
	}

	if resp.Error.Code != -32600 {
		t.Errorf("expected error code -32600 (Invalid Request), got %d", resp.Error.Code)
	}

	// Error message is "Invalid Request", but Data contains "Batch requests not supported"
	data, ok := resp.Error.Data.(string)
	if !ok || !strings.Contains(data, "Batch") {
		t.Errorf("expected error data to mention batch requests, got message: %s, data: %v", resp.Error.Message, resp.Error.Data)
	}
}

func TestMCPProtocol_InitializeIncludesInstructions(t *testing.T) {
	server, _ := setupTestServer(t)

	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
`)
	output := &bytes.Buffer{}

	server.in = input
	server.out = output

	server.Run()

	var resp Response
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nOutput: %s", err, output.String())
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be map, got %T", resp.Result)
	}

	instructions, ok := result["instructions"].(string)
	if !ok {
		t.Fatalf("expected instructions to be string, got %T", result["instructions"])
	}

	if !strings.Contains(instructions, "Tasuku") {
		t.Errorf("expected instructions to mention Tasuku, got: %s", instructions)
	}

	if !strings.Contains(instructions, "tk_context") {
		t.Errorf("expected instructions to mention tk_context, got: %s", instructions)
	}
}

func TestHandleToolCall_ListIncludeArchived(t *testing.T) {
	server, _ := setupTestServer(t)

	// Add tasks with different tags and owners
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Active task",
		"id":          "active-1",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Will be archived with tag",
		"id":          "archived-tagged",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Will be archived without tag",
		"id":          "archived-untagged",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Will be archived with owner",
		"id":          "archived-owned",
	})

	// Add tags and owners before archiving
	server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "tag_add",
		"id":     "archived-tagged",
		"tag":    "bug",
	})
	owner := "alice"
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "owner",
		"id":     "archived-owned",
		"owner":  owner,
	})

	// Complete and archive tasks
	for _, id := range []string{"archived-tagged", "archived-untagged", "archived-owned"} {
		server.HandleToolCall("tk_done", map[string]interface{}{"id": id})
		server.HandleToolCall("tk_task", map[string]interface{}{
			"action": "archive",
			"id":     id,
		})
	}

	t.Run("include_archived shows all", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"include_archived": true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		// 1 active + 3 archived = 4
		if len(list) != 4 {
			t.Errorf("expected 4 tasks, got %d", len(list))
		}
	})

	t.Run("include_archived with tag filter", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"include_archived": true,
			"tag":              "bug",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		// Only archived-tagged has the "bug" tag
		if len(list) != 1 {
			t.Errorf("expected 1 task with tag 'bug', got %d", len(list))
		}
		if len(list) > 0 && list[0]["id"] != "archived-tagged" {
			t.Errorf("expected id 'archived-tagged', got %v", list[0]["id"])
		}
	})

	t.Run("include_archived with owner filter", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"include_archived": true,
			"owner":            "alice",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		// Only archived-owned has owner "alice"
		if len(list) != 1 {
			t.Errorf("expected 1 task with owner 'alice', got %d", len(list))
		}
		if len(list) > 0 && list[0]["id"] != "archived-owned" {
			t.Errorf("expected id 'archived-owned', got %v", list[0]["id"])
		}
	})

	t.Run("include_archived with status filter excludes archived", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"include_archived": true,
			"status":           "ready",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		// Only active-1 is "ready", archived tasks should be filtered out
		if len(list) != 1 {
			t.Errorf("expected 1 ready task, got %d", len(list))
		}
		if len(list) > 0 && list[0]["id"] != "active-1" {
			t.Errorf("expected id 'active-1', got %v", list[0]["id"])
		}
	})

	t.Run("include_archived with status=archived", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"include_archived": true,
			"status":           "archived",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		// Only the 3 archived tasks
		if len(list) != 3 {
			t.Errorf("expected 3 archived tasks, got %d", len(list))
		}
		for _, task := range list {
			if task["status"] != "archived" {
				t.Errorf("expected status 'archived', got %v", task["status"])
			}
		}
	})

	t.Run("without include_archived excludes archived", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		if len(list) != 1 {
			t.Errorf("expected 1 active task, got %d", len(list))
		}
		for _, task := range list {
			if task["status"] == "archived" {
				t.Error("should not include archived tasks without include_archived flag")
			}
		}
	})
}

func TestHandleToolCall_ListFilters(t *testing.T) {
	server, _ := setupTestServer(t)

	// Set up tasks with various attributes
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Bug fix",
		"id":          "bug-1",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Feature work",
		"id":          "feature-1",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"description": "Another bug",
		"id":          "bug-2",
	})
	server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "tag_add",
		"id":     "bug-1",
		"tag":    "bug",
	})
	server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "tag_add",
		"id":     "bug-2",
		"tag":    "bug",
	})
	server.HandleToolCall("tk_metadata", map[string]interface{}{
		"action": "tag_add",
		"id":     "feature-1",
		"tag":    "feature",
	})
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "owner",
		"id":     "bug-1",
		"owner":  "bob",
	})
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "owner",
		"id":     "feature-1",
		"owner":  "bob",
	})
	server.HandleToolCall("tk_start", map[string]interface{}{"id": "feature-1"})

	t.Run("filter by tag", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"tag": "bug",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		if len(list) != 2 {
			t.Errorf("expected 2 tasks with tag 'bug', got %d", len(list))
		}
	})

	t.Run("filter by owner", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"owner": "bob",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		if len(list) != 2 {
			t.Errorf("expected 2 tasks with owner 'bob', got %d", len(list))
		}
	})

	t.Run("filter by status and tag", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"status": "ready",
			"tag":    "bug",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		if len(list) != 2 {
			t.Errorf("expected 2 ready tasks with tag 'bug', got %d", len(list))
		}
	})

	t.Run("filter by status and owner", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"status": "in_progress",
			"owner":  "bob",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		if len(list) != 1 {
			t.Errorf("expected 1 in_progress task with owner 'bob', got %d", len(list))
		}
		if len(list) > 0 && list[0]["id"] != "feature-1" {
			t.Errorf("expected id 'feature-1', got %v", list[0]["id"])
		}
	})

	t.Run("filter with no matches", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_list", map[string]interface{}{
			"owner": "nobody",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var list []map[string]interface{}
		json.Unmarshal(data, &list)

		if len(list) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(list))
		}
	})
}

func TestConsolidatedTool_MissingAction(t *testing.T) {
	server, _ := setupTestServer(t)

	t.Run("tk_task missing action", func(t *testing.T) {
		_, err := server.HandleToolCall("tk_task", map[string]interface{}{})
		if err == nil {
			t.Error("expected error for missing action")
		}
	})

	t.Run("tk_metadata missing action", func(t *testing.T) {
		_, err := server.HandleToolCall("tk_metadata", map[string]interface{}{})
		if err == nil {
			t.Error("expected error for missing action")
		}
	})

	t.Run("tk_manage missing action", func(t *testing.T) {
		_, err := server.HandleToolCall("tk_manage", map[string]interface{}{})
		if err == nil {
			t.Error("expected error for missing action")
		}
	})
}

func TestConsolidatedTool_UnknownAction(t *testing.T) {
	server, _ := setupTestServer(t)

	t.Run("tk_task unknown action", func(t *testing.T) {
		_, err := server.HandleToolCall("tk_task", map[string]interface{}{
			"action": "invalid",
		})
		if err == nil {
			t.Error("expected error for unknown action")
		}
		if !strings.Contains(err.Error(), "unknown action") {
			t.Errorf("expected 'unknown action' in error, got: %v", err)
		}
	})

	t.Run("tk_metadata unknown action", func(t *testing.T) {
		_, err := server.HandleToolCall("tk_metadata", map[string]interface{}{
			"action": "invalid",
		})
		if err == nil {
			t.Error("expected error for unknown action")
		}
		if !strings.Contains(err.Error(), "unknown action") {
			t.Errorf("expected 'unknown action' in error, got: %v", err)
		}
	})

	t.Run("tk_manage unknown action", func(t *testing.T) {
		_, err := server.HandleToolCall("tk_manage", map[string]interface{}{
			"action": "invalid",
		})
		if err == nil {
			t.Error("expected error for unknown action")
		}
		if !strings.Contains(err.Error(), "unknown action") {
			t.Errorf("expected 'unknown action' in error, got: %v", err)
		}
	})
}

func TestHandleToolCall_Help(t *testing.T) {
	server, _ := setupTestServer(t)

	t.Run("default overview", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["topic"] != "Tasuku Overview" {
			t.Errorf("expected topic 'Tasuku Overview', got %v", help["topic"])
		}
	})

	t.Run("tasks topic", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"topic": "tasks",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["topic"] != "Task Operations" {
			t.Errorf("expected topic 'Task Operations', got %v", help["topic"])
		}
		if help["tk_task_actions"] == nil {
			t.Error("expected tk_task_actions in response")
		}
	})

	t.Run("metadata topic", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"topic": "metadata",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["topic"] != "Metadata Operations" {
			t.Errorf("expected topic 'Metadata Operations', got %v", help["topic"])
		}
	})

	t.Run("knowledge topic", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"topic": "knowledge",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["topic"] != "Knowledge Capture" {
			t.Errorf("expected topic 'Knowledge Capture', got %v", help["topic"])
		}
	})

	t.Run("multiagent topic", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"topic": "multiagent",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["topic"] != "Multi-Agent Coordination" {
			t.Errorf("expected topic 'Multi-Agent Coordination', got %v", help["topic"])
		}
	})

	t.Run("archive topic", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"topic": "archive",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["topic"] != "Archive Operations" {
			t.Errorf("expected topic 'Archive Operations', got %v", help["topic"])
		}
	})

	t.Run("install topic", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"topic": "install",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["topic"] != "Installation" {
			t.Errorf("expected topic 'Installation', got %v", help["topic"])
		}
	})

	t.Run("unknown topic falls through to overview", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"topic": "nonexistent",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		// Falls through to default which is the overview
		if help["topic"] != "Tasuku Overview" {
			t.Errorf("expected topic 'Tasuku Overview' for unknown topic, got %v", help["topic"])
		}
	})

	t.Run("command reference", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"command": "tk_task",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["command"] != "tk_task" {
			t.Errorf("expected command 'tk_task', got %v", help["command"])
		}
		if help["actions"] == nil {
			t.Error("expected actions in command help")
		}
	})

	t.Run("unknown command reference", func(t *testing.T) {
		result, err := server.HandleToolCall("tk_help", map[string]interface{}{
			"command": "tk_nonexistent",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, _ := json.Marshal(result)
		var help map[string]interface{}
		json.Unmarshal(data, &help)

		if help["error"] == nil {
			t.Error("expected error for unknown command")
		}
	})
}

func TestHandleToolCall_HelpOverviewToolCount(t *testing.T) {
	server, _ := setupTestServer(t)

	result, err := server.HandleToolCall("tk_help", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := json.Marshal(result)
	var help map[string]interface{}
	json.Unmarshal(data, &help)

	toolCount := int(help["tool_count"].(float64))
	if toolCount != 17 {
		t.Errorf("expected tool_count 17, got %d", toolCount)
	}

	tools := help["tools"].(map[string]interface{})
	if tools["tk_block"] == nil {
		t.Error("expected tk_block in help overview tools")
	}

	// Verify count matches actual tool entries
	if len(tools) != toolCount {
		t.Errorf("tool_count (%d) doesn't match actual tools map length (%d)", toolCount, len(tools))
	}
}

func TestHandleToolCall_ListArchivedSortOrder(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create tasks in different states
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "active-task",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "done-task",
	})
	server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "done-task",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "to-archive",
	})
	server.HandleToolCall("tk_done", map[string]interface{}{
		"id": "to-archive",
	})
	server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "archive",
		"id":     "to-archive",
	})

	result, err := server.HandleToolCall("tk_list", map[string]interface{}{
		"include_archived": true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := json.Marshal(result)
	var list []map[string]interface{}
	json.Unmarshal(data, &list)

	if len(list) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(list))
	}

	// Archived should be last (after ready and done)
	lastTask := list[len(list)-1]
	if lastTask["status"] != "archived" {
		t.Errorf("expected archived task to sort last, got status %v at last position", lastTask["status"])
	}

	// Ready should come before done
	if list[0]["status"] != "ready" {
		t.Errorf("expected ready task first, got %v", list[0]["status"])
	}
}

func TestConsolidatedTool_BlockViaTaskAction(t *testing.T) {
	server, _ := setupTestServer(t)

	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "blocker",
	})
	server.HandleToolCall("tk_add", map[string]interface{}{
		"id": "blocked",
	})

	// Block via tk_task consolidated tool
	result, err := server.HandleToolCall("tk_task", map[string]interface{}{
		"action":     "block",
		"id":         "blocked",
		"blocked_by": []interface{}{"blocker"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := result.(map[string]interface{})
	if r["status"] != "blocked" {
		t.Errorf("expected status 'blocked', got %v", r["status"])
	}

	// Unblock via tk_task consolidated tool
	result, err = server.HandleToolCall("tk_task", map[string]interface{}{
		"action": "unblock",
		"id":     "blocked",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r = result.(map[string]interface{})
	if r["status"] != "ready" {
		t.Errorf("expected status 'ready', got %v", r["status"])
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
