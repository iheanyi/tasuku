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
	if learnings[0] != "Test insight" {
		t.Errorf("expected 'Test insight', got '%v'", learnings[0])
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
