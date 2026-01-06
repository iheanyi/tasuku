package v4

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewIndex(t *testing.T) {
	idx := NewIndex()

	if idx.Version != "v4" {
		t.Errorf("Version = %q, want v4", idx.Version)
	}
	if idx.Tasks == nil {
		t.Error("Tasks should be initialized")
	}
	if len(idx.Tasks) != 0 {
		t.Errorf("Tasks should be empty, got %d", len(idx.Tasks))
	}
}

func TestIndexAddTask(t *testing.T) {
	idx := NewIndex()
	priority := 2

	fm := TaskFrontmatter{
		Status:    "in_progress",
		Priority:  &priority,
		Tags:      []string{"backend", "api"},
		BlockedBy: []string{"dep-1"},
		ParentID:  "parent-task",
		Owner:     "claude",
		ClaimedBy: "agent-1",
		UpdatedAt: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC),
	}

	idx.AddTask("test-task", fm)

	if len(idx.Tasks) != 1 {
		t.Fatalf("Tasks count = %d, want 1", len(idx.Tasks))
	}

	meta := idx.Tasks["test-task"]
	if meta.Status != "in_progress" {
		t.Errorf("Status = %q, want in_progress", meta.Status)
	}
	if meta.Priority == nil || *meta.Priority != 2 {
		t.Errorf("Priority = %v, want 2", meta.Priority)
	}
	if len(meta.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(meta.Tags))
	}
	if meta.File != "tasks/test-task.md" {
		t.Errorf("File = %q, want tasks/test-task.md", meta.File)
	}
}

func TestIndexRemoveTask(t *testing.T) {
	idx := NewIndex()
	idx.AddTask("task-1", TaskFrontmatter{Status: "ready"})
	idx.AddTask("task-2", TaskFrontmatter{Status: "done"})

	if len(idx.Tasks) != 2 {
		t.Fatalf("Tasks count = %d, want 2", len(idx.Tasks))
	}

	idx.RemoveTask("task-1")

	if len(idx.Tasks) != 1 {
		t.Fatalf("Tasks count after remove = %d, want 1", len(idx.Tasks))
	}
	if _, exists := idx.Tasks["task-1"]; exists {
		t.Error("task-1 should be removed")
	}
	if _, exists := idx.Tasks["task-2"]; !exists {
		t.Error("task-2 should still exist")
	}
}

func TestIndexSetCounts(t *testing.T) {
	idx := NewIndex()
	idx.SetCounts(42, 5, 3)

	if idx.ArchivedCount != 42 {
		t.Errorf("ArchivedCount = %d, want 42", idx.ArchivedCount)
	}
	if idx.LearningsCount != 5 {
		t.Errorf("LearningsCount = %d, want 5", idx.LearningsCount)
	}
	if idx.DecisionsCount != 3 {
		t.Errorf("DecisionsCount = %d, want 3", idx.DecisionsCount)
	}
}

func TestIndexMarshalRoundtrip(t *testing.T) {
	idx := NewIndex()
	priority := 1
	idx.AddTask("test-task", TaskFrontmatter{
		Status:    "ready",
		Priority:  &priority,
		Tags:      []string{"test"},
		UpdatedAt: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC),
	})
	idx.SetCounts(10, 2, 1)

	// Marshal
	data, err := idx.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Parse back
	parsed, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("ParseIndex() error = %v", err)
	}

	if parsed.Version != "v4" {
		t.Errorf("Roundtrip Version = %q, want v4", parsed.Version)
	}
	if len(parsed.Tasks) != 1 {
		t.Errorf("Roundtrip Tasks count = %d, want 1", len(parsed.Tasks))
	}
	if parsed.ArchivedCount != 10 {
		t.Errorf("Roundtrip ArchivedCount = %d, want 10", parsed.ArchivedCount)
	}
	if parsed.LearningsCount != 2 {
		t.Errorf("Roundtrip LearningsCount = %d, want 2", parsed.LearningsCount)
	}

	meta := parsed.Tasks["test-task"]
	if meta.Status != "ready" {
		t.Errorf("Roundtrip Status = %q, want ready", meta.Status)
	}
}

func TestIndexJSON(t *testing.T) {
	idx := NewIndex()
	priority := 2
	idx.AddTask("auth-jwt", TaskFrontmatter{
		Status:    "in_progress",
		Priority:  &priority,
		Tags:      []string{"backend", "api"},
		BlockedBy: []string{"auth-setup"},
		ParentID:  "epic-123",
		Owner:     "claude",
		ClaimedBy: "agent-1",
		UpdatedAt: time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC),
	})
	idx.SetCounts(42, 5, 3)

	data, err := idx.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Validate JSON structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if raw["version"] != "v4" {
		t.Errorf("JSON version = %v, want v4", raw["version"])
	}

	tasks := raw["tasks"].(map[string]interface{})
	if len(tasks) != 1 {
		t.Errorf("JSON tasks count = %d, want 1", len(tasks))
	}

	taskMeta := tasks["auth-jwt"].(map[string]interface{})
	if taskMeta["status"] != "in_progress" {
		t.Errorf("JSON task status = %v, want in_progress", taskMeta["status"])
	}
	if taskMeta["file"] != "tasks/auth-jwt.md" {
		t.Errorf("JSON task file = %v, want tasks/auth-jwt.md", taskMeta["file"])
	}
}
