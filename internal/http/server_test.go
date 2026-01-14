package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/iheanyi/tasuku/internal/store"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Create temp directory with .tasuku.json
	dir := t.TempDir()
	tasukuPath := filepath.Join(dir, ".tasuku.json")
	initialData := `{
		"version": 1,
		"tasks": {},
		"context": {
			"learnings": [],
			"decisions": [],
			"notes": {}
		}
	}`
	if err := os.WriteFile(tasukuPath, []byte(initialData), 0644); err != nil {
		t.Fatalf("failed to create .tasuku.json: %v", err)
	}

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(dir)

	s := store.Default()
	srv := New(s)

	cleanup := func() {
		os.Chdir(oldDir)
	}

	return srv, cleanup
}

func TestHealthCheck(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}
}

func TestCreateTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"description": "Test task", "priority": 1}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "created" {
		t.Errorf("expected status 'created', got '%s'", resp["status"])
	}
	if resp["id"] == "" {
		t.Error("expected non-empty id")
	}
}

func TestCreateTaskWithID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"id": "custom-id", "description": "Custom task"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] != "custom-id" {
		t.Errorf("expected id 'custom-id', got '%s'", resp["id"])
	}
}

func TestCreateTaskMissingDescription(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"id": "no-desc"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestListTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task first
	body := bytes.NewBufferString(`{"description": "List test task"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// List tasks
	req = httptest.NewRequest("GET", "/tasks", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var tasks []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
}

func TestListTasksWithStatusFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "filter-test", "description": "Filter test"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Filter by ready (should find the task)
	req = httptest.NewRequest("GET", "/tasks?status=ready", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var tasks []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &tasks)

	if len(tasks) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(tasks))
	}

	// Filter by in_progress (should be empty)
	req = httptest.NewRequest("GET", "/tasks?status=in_progress", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &tasks)

	if len(tasks) != 0 {
		t.Errorf("expected 0 in_progress tasks, got %d", len(tasks))
	}
}

func TestGetTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "get-test", "description": "Get test task"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Get the task
	req = httptest.NewRequest("GET", "/tasks/get-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	if task["id"] != "get-test" {
		t.Errorf("expected id 'get-test', got '%v'", task["id"])
	}
	if task["description"] != "Get test task" {
		t.Errorf("expected description 'Get test task', got '%v'", task["description"])
	}
}

func TestGetTaskNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/tasks/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "update-test", "description": "Update test"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Update status
	body = bytes.NewBufferString(`{"status": "in_progress"}`)
	req = httptest.NewRequest("PUT", "/tasks/update-test", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the update
	req = httptest.NewRequest("GET", "/tasks/update-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	if task["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got '%v'", task["status"])
	}
}

func TestUpdateTaskPriority(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "priority-test", "description": "Priority test"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Update priority
	body = bytes.NewBufferString(`{"priority": 0}`)
	req = httptest.NewRequest("PUT", "/tasks/priority-test", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify the update
	req = httptest.NewRequest("GET", "/tasks/priority-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	if task["priority"] != float64(0) {
		t.Errorf("expected priority 0, got '%v'", task["priority"])
	}
}

func TestDeleteTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "delete-test", "description": "Delete test"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Delete the task
	req = httptest.NewRequest("DELETE", "/tasks/delete-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify it's gone
	req = httptest.NewRequest("GET", "/tasks/delete-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestReadyTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create tasks with different statuses
	tasks := []string{
		`{"id": "ready-1", "description": "Ready 1"}`,
		`{"id": "ready-2", "description": "Ready 2"}`,
	}

	for _, taskJSON := range tasks {
		body := bytes.NewBufferString(taskJSON)
		req := httptest.NewRequest("POST", "/tasks", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
	}

	// Mark one as in_progress
	body := bytes.NewBufferString(`{"status": "in_progress"}`)
	req := httptest.NewRequest("PUT", "/tasks/ready-1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Get ready tasks
	req = httptest.NewRequest("GET", "/ready", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var readyTasks []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &readyTasks)

	if len(readyTasks) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(readyTasks))
	}
}

func TestAddLearning(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"learning": "Test insight"}`)
	req := httptest.NewRequest("POST", "/learnings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify via context
	req = httptest.NewRequest("GET", "/context", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var ctx map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ctx)

	ctxData := ctx["context"].(map[string]interface{})
	learnings := ctxData["learnings"].([]interface{})

	if len(learnings) != 1 {
		t.Errorf("expected 1 learning, got %d", len(learnings))
	}
	learning := learnings[0].(map[string]interface{})
	if learning["text"] != "Test insight" {
		t.Errorf("expected 'Test insight', got '%v'", learning["text"])
	}
}

func TestAddLearningMissing(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/learnings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAddDecision(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{
		"id": "test-decision",
		"chose": "Option A",
		"over": ["Option B", "Option C"],
		"because": "It's simpler"
	}`)
	req := httptest.NewRequest("POST", "/decisions", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify via context
	req = httptest.NewRequest("GET", "/context", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var ctx map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ctx)

	ctxData := ctx["context"].(map[string]interface{})
	decisions := ctxData["decisions"].([]interface{})

	if len(decisions) != 1 {
		t.Errorf("expected 1 decision, got %d", len(decisions))
	}
}

func TestAddDecisionMissingFields(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"id": "incomplete"}`)
	req := httptest.NewRequest("POST", "/decisions", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetContext(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/context", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var ctx map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if ctx["version"] != float64(1) {
		t.Errorf("expected version 1, got '%v'", ctx["version"])
	}
}

func TestGetSchema(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/schema", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &schema); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if schema["$schema"] == nil {
		t.Error("expected $schema field in response")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("PATCH", "/tasks", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("OPTIONS", "/tasks", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header Access-Control-Allow-Origin: *")
	}
}

func TestListArchivedTasksEmpty(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/archive", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var tasks []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("expected 0 archived tasks, got %d", len(tasks))
	}
}

func TestArchiveTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "archive-test", "description": "Task to archive"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Mark it as done
	body = bytes.NewBufferString(`{"status": "done"}`)
	req = httptest.NewRequest("PUT", "/tasks/archive-test", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Archive the task
	body = bytes.NewBufferString(`{"summary": "Completed successfully"}`)
	req = httptest.NewRequest("POST", "/tasks/archive-test/archive", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "archived" {
		t.Errorf("expected status 'archived', got '%s'", resp["status"])
	}

	// Verify task is no longer in active tasks
	req = httptest.NewRequest("GET", "/tasks/archive-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 (task should be archived), got %d", w.Code)
	}

	// Verify task is in archive
	req = httptest.NewRequest("GET", "/archive", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var archivedTasks []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &archivedTasks)

	if len(archivedTasks) != 1 {
		t.Errorf("expected 1 archived task, got %d", len(archivedTasks))
	}

	if archivedTasks[0]["id"] != "archive-test" {
		t.Errorf("expected archived task id 'archive-test', got '%v'", archivedTasks[0]["id"])
	}

	if archivedTasks[0]["summary"] != "Completed successfully" {
		t.Errorf("expected summary 'Completed successfully', got '%v'", archivedTasks[0]["summary"])
	}
}

func TestArchiveTaskNotDone(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task (status will be "ready")
	body := bytes.NewBufferString(`{"id": "not-done-task", "description": "Task not done"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Try to archive it without marking done (should fail)
	body = bytes.NewBufferString(`{"summary": "Trying to archive"}`)
	req = httptest.NewRequest("POST", "/tasks/not-done-task/archive", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestRestoreArchivedTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create and complete a task
	body := bytes.NewBufferString(`{"id": "restore-test", "description": "Task to restore"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Mark it as done
	body = bytes.NewBufferString(`{"status": "done"}`)
	req = httptest.NewRequest("PUT", "/tasks/restore-test", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Archive the task
	body = bytes.NewBufferString(`{}`)
	req = httptest.NewRequest("POST", "/tasks/restore-test/archive", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Restore the task
	req = httptest.NewRequest("POST", "/archive/restore-test/restore", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "restored" {
		t.Errorf("expected status 'restored', got '%s'", resp["status"])
	}

	// Verify task is back in active tasks
	req = httptest.NewRequest("GET", "/tasks/restore-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (task should be restored), got %d", w.Code)
	}

	// Verify task is no longer in archive
	req = httptest.NewRequest("GET", "/archive", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var archivedTasks []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &archivedTasks)

	if len(archivedTasks) != 0 {
		t.Errorf("expected 0 archived tasks, got %d", len(archivedTasks))
	}
}

func TestRestoreNonexistentArchivedTask(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/archive/nonexistent/restore", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestClearArchive(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create and archive multiple tasks
	for i := 0; i < 3; i++ {
		id := "clear-test-" + string(rune('a'+i))
		body := bytes.NewBufferString(`{"id": "` + id + `", "description": "Task ` + id + `"}`)
		req := httptest.NewRequest("POST", "/tasks", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		// Mark as done
		body = bytes.NewBufferString(`{"status": "done"}`)
		req = httptest.NewRequest("PUT", "/tasks/"+id, body)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		srv.ServeHTTP(w, req)

		// Archive
		req = httptest.NewRequest("POST", "/tasks/"+id+"/archive", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		srv.ServeHTTP(w, req)
	}

	// Verify 3 tasks are archived
	req := httptest.NewRequest("GET", "/archive", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var archivedTasks []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &archivedTasks)

	if len(archivedTasks) != 3 {
		t.Errorf("expected 3 archived tasks, got %d", len(archivedTasks))
	}

	// Clear the archive
	req = httptest.NewRequest("DELETE", "/archive", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "cleared" {
		t.Errorf("expected status 'cleared', got '%v'", resp["status"])
	}

	if resp["count"] != float64(3) {
		t.Errorf("expected count 3, got '%v'", resp["count"])
	}

	// Verify archive is empty
	req = httptest.NewRequest("GET", "/archive", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &archivedTasks)

	if len(archivedTasks) != 0 {
		t.Errorf("expected 0 archived tasks after clear, got %d", len(archivedTasks))
	}
}

func TestArchiveCORSHeaders(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("OPTIONS", "/archive", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header Access-Control-Allow-Origin: *")
	}
}

func TestTimerStartStop(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "timer-test", "description": "Timer test task"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Start timer
	req = httptest.NewRequest("POST", "/tasks/timer-test/timer/start", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "timer_started" {
		t.Errorf("expected status 'timer_started', got '%v'", resp["status"])
	}

	// Check timers endpoint - returns array of timer info objects
	req = httptest.NewRequest("GET", "/timers", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var timers []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &timers)

	found := false
	for _, timer := range timers {
		if timer["id"] == "timer-test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected timer-test to have active timer")
	}

	// Stop timer
	req = httptest.NewRequest("POST", "/tasks/timer-test/timer/stop", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "timer_stopped" {
		t.Errorf("expected status 'timer_stopped', got '%v'", resp["status"])
	}
}

func TestTimerTaskNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/tasks/nonexistent/timer/start", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Returns 400 because the error is from the store, not a route issue
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAddTag(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "tag-test", "description": "Tag test task"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Add tag - tag name is in the URL path
	req = httptest.NewRequest("PUT", "/tasks/tag-test/tags/important", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify tag was added
	req = httptest.NewRequest("GET", "/tasks/tag-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	// Tags may be nil if empty
	if task["tags"] != nil {
		tags := task["tags"].([]interface{})
		found := false
		for _, tag := range tags {
			if tag == "important" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected tag 'important' to be present")
		}
	} else {
		t.Error("expected tags to be non-nil after adding")
	}
}

func TestRemoveTag(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task with tags
	body := bytes.NewBufferString(`{"id": "tag-remove-test", "description": "Remove tag test"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Add tag first - tag name in URL path
	req = httptest.NewRequest("PUT", "/tasks/tag-remove-test/tags/removeme", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Remove tag - tag name in URL path
	req = httptest.NewRequest("DELETE", "/tasks/tag-remove-test/tags/removeme", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify tag was removed
	req = httptest.NewRequest("GET", "/tasks/tag-remove-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	if task["tags"] != nil {
		tags := task["tags"].([]interface{})
		for _, tag := range tags {
			if tag == "removeme" {
				t.Error("tag 'removeme' should have been removed")
			}
		}
	}
}

func TestSetField(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "field-test", "description": "Field test task"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Set field - key in URL path, value in body
	body = bytes.NewBufferString(`{"value": "2h"}`)
	req = httptest.NewRequest("PUT", "/tasks/field-test/fields/estimate", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify field was set
	req = httptest.NewRequest("GET", "/tasks/field-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	if task["fields"] != nil {
		fields := task["fields"].(map[string]interface{})
		if fields["estimate"] != "2h" {
			t.Errorf("expected estimate '2h', got '%v'", fields["estimate"])
		}
	} else {
		t.Error("expected fields to be non-nil after setting")
	}
}

func TestRemoveField(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "field-remove-test", "description": "Field remove test"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Set field first - key in URL path
	body = bytes.NewBufferString(`{"value": "auth"}`)
	req = httptest.NewRequest("PUT", "/tasks/field-remove-test/fields/component", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Remove field - key in URL path
	req = httptest.NewRequest("DELETE", "/tasks/field-remove-test/fields/component", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify field was removed
	req = httptest.NewRequest("GET", "/tasks/field-remove-test", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	if task["fields"] != nil {
		fields := task["fields"].(map[string]interface{})
		if fields["component"] != nil {
			t.Error("field 'component' should have been removed")
		}
	}
}

func TestGetTimersEmpty(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/timers", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var timers []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &timers)

	if len(timers) != 0 {
		t.Errorf("expected empty timers array, got %d entries", len(timers))
	}
}

func TestDashboard(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Dashboard should return HTML
	contentType := w.Header().Get("Content-Type")
	if contentType != "" && contentType != "text/html; charset=utf-8" && contentType != "text/html" {
		// Accept empty or text/html
		body := w.Body.String()
		if len(body) > 0 && body[0] != '<' {
			t.Errorf("expected HTML content, got content-type: %s", contentType)
		}
	}
}

func TestPartialTasks(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a task
	body := bytes.NewBufferString(`{"id": "partial-test", "description": "Partial test task"}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Get partial tasks
	req = httptest.NewRequest("GET", "/partials/tasks", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Should return HTML partial
	body2 := w.Body.String()
	if len(body2) == 0 {
		t.Error("expected non-empty partial response")
	}
}

func TestPartialStats(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/partials/stats", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestPartialProgress(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/partials/progress", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// Note: UpdateTaskDescription removed - HTTP API only supports status/priority updates
// Description updates are available via CLI only

func TestUpdateTaskNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"status": "done"}`)
	req := httptest.NewRequest("PUT", "/tasks/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Returns 400 with error message, not 404
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("DELETE", "/tasks/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Returns 400 with error message, not 404
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestArchiveTaskNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/tasks/nonexistent/archive", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Returns 400 with error message, not 404
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// Tag and Field route tests removed - API uses path-based routing
// e.g., /tasks/{id}/tags/{tag} not /tasks/{id}/tags with JSON body

func TestInvalidJSON(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateTaskWithTags(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := bytes.NewBufferString(`{"id": "tagged-task", "description": "Task with tags", "tags": ["bug", "urgent"]}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify tags
	req = httptest.NewRequest("GET", "/tasks/tagged-task", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var task map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &task)

	tags := task["tags"].([]interface{})
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestListTasksWithTagFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create tasks with different tags
	body := bytes.NewBufferString(`{"id": "bug-task", "description": "Bug task", "tags": ["bug"]}`)
	req := httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	body = bytes.NewBufferString(`{"id": "feature-task", "description": "Feature task", "tags": ["feature"]}`)
	req = httptest.NewRequest("POST", "/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Filter by bug tag
	req = httptest.NewRequest("GET", "/tasks?tag=bug", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var tasks []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &tasks)

	if len(tasks) != 1 {
		t.Errorf("expected 1 task with bug tag, got %d", len(tasks))
	}

	if tasks[0]["id"] != "bug-task" {
		t.Errorf("expected bug-task, got %v", tasks[0]["id"])
	}
}
