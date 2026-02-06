package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestDirStore_New(t *testing.T) {
	s := NewDirStore("/tmp/.tasuku")
	if s.Path() != "/tmp/.tasuku" {
		t.Errorf("expected path '/tmp/.tasuku', got %s", s.Path())
	}
}

func TestDirStore_Exists(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	s := NewDirStore(storePath)

	if s.Exists() {
		t.Fatal("store should not exist before directory is created")
	}

	os.MkdirAll(storePath, 0755)
	if !s.Exists() {
		t.Fatal("store should exist after directory is created")
	}
}

func TestDirStore_Read(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	s := NewDirStore(storePath)

	// Create a valid V3 directory structure
	setupV3Directory(t, storePath)

	// Add a task file
	tasksDir := filepath.Join(storePath, "tasks")
	twid := TaskWithID{
		ID: "test-task",
		Task: task.Task{
			Status:      task.StatusReady,
			Description: "Test description",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}
	data, _ := json.MarshalIndent(twid, "", "  ")
	os.WriteFile(filepath.Join(tasksDir, "test-task.json"), data, 0644)

	// Read it back
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(f.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(f.Tasks))
	}

	tk, exists := f.Tasks["test-task"]
	if !exists {
		t.Fatal("task should exist")
	}
	if tk.Description != "Test description" {
		t.Errorf("expected description 'Test description', got %q", tk.Description)
	}
	if tk.Status != "ready" {
		t.Errorf("expected status 'ready', got %q", tk.Status)
	}
}

func TestDirStore_Read_WithLearnings(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	setupV3Directory(t, storePath)

	// Add learnings
	learnings := []task.Learning{
		{ID: "l1", Text: "Always validate input", IsRule: true, CreatedAt: time.Now().UTC()},
		{ID: "l2", Text: "Redis improves cache performance", IsRule: false, CreatedAt: time.Now().UTC()},
	}
	data, _ := json.MarshalIndent(learnings, "", "  ")
	os.WriteFile(filepath.Join(storePath, "context", "learnings.json"), data, 0644)

	s := NewDirStore(storePath)
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(f.Context.Learnings) != 2 {
		t.Fatalf("expected 2 learnings, got %d", len(f.Context.Learnings))
	}
	if f.Context.Learnings[0].Text != "Always validate input" {
		t.Errorf("wrong learning text: %s", f.Context.Learnings[0].Text)
	}
}

func TestDirStore_Read_WithDecisions(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	setupV3Directory(t, storePath)

	// Add decisions
	decisions := []task.Decision{
		{ID: "d1", Chose: "PostgreSQL", Over: []string{"MySQL"}, Because: "Better JSON support", CreatedAt: time.Now().UTC()},
	}
	data, _ := json.MarshalIndent(decisions, "", "  ")
	os.WriteFile(filepath.Join(storePath, "context", "decisions.json"), data, 0644)

	s := NewDirStore(storePath)
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(f.Context.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(f.Context.Decisions))
	}
	if f.Context.Decisions[0].Chose != "PostgreSQL" {
		t.Errorf("wrong decision: %s", f.Context.Decisions[0].Chose)
	}
}

func TestDirStore_Read_WithNotes(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	setupV3Directory(t, storePath)

	// Add a task
	tasksDir := filepath.Join(storePath, "tasks")
	twid := TaskWithID{ID: "note-task", Task: task.NewTask("Task with notes")}
	data, _ := json.MarshalIndent(twid, "", "  ")
	os.WriteFile(filepath.Join(tasksDir, "note-task.json"), data, 0644)

	// Add notes for the task
	notesDir := filepath.Join(storePath, "context", "notes")
	os.MkdirAll(notesDir, 0755)
	notes := []task.Note{
		{ID: "n1", Text: "This is a note", CreatedAt: time.Now().UTC()},
	}
	noteData, _ := json.MarshalIndent(notes, "", "  ")
	os.WriteFile(filepath.Join(notesDir, "note-task.json"), noteData, 0644)

	s := NewDirStore(storePath)
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(f.Context.Notes["note-task"]) != 1 {
		t.Fatalf("expected 1 note, got %d", len(f.Context.Notes["note-task"]))
	}
	if f.Context.Notes["note-task"][0].Text != "This is a note" {
		t.Errorf("wrong note text: %s", f.Context.Notes["note-task"][0].Text)
	}
}

func TestDirStore_Read_WithArchive(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	setupV3Directory(t, storePath)

	// Add an archived task
	archived := task.ArchivedTask{
		Task:       task.NewTask("Archived task"),
		ArchivedAt: time.Now().UTC(),
		Summary:    "Completed",
	}
	data, _ := json.MarshalIndent(archived, "", "  ")
	os.WriteFile(filepath.Join(storePath, "archive", "archived-1.json"), data, 0644)

	s := NewDirStore(storePath)
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(f.Archive) != 1 {
		t.Fatalf("expected 1 archived task, got %d", len(f.Archive))
	}
	if f.Archive["archived-1"].Summary != "Completed" {
		t.Errorf("wrong summary: %s", f.Archive["archived-1"].Summary)
	}
}

func TestDirStore_Read_NotExists(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	s := NewDirStore(storePath)

	_, err := s.Read()
	if err == nil {
		t.Fatal("expected error when reading non-existent directory")
	}
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestDirStore_Read_MultipleTasks(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	setupV3Directory(t, storePath)

	// Add multiple tasks
	tasksDir := filepath.Join(storePath, "tasks")
	for i := 0; i < 5; i++ {
		id := "task-" + string(rune('a'+i))
		twid := TaskWithID{ID: id, Task: task.NewTask("Task " + id)}
		data, _ := json.MarshalIndent(twid, "", "  ")
		os.WriteFile(filepath.Join(tasksDir, id+".json"), data, 0644)
	}

	s := NewDirStore(storePath)
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(f.Tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(f.Tasks))
	}
}

func TestDirStore_MigrationReader(t *testing.T) {
	// Verify DirStore implements MigrationReader
	var _ MigrationReader = NewDirStore("/tmp/.tasuku")
}

// setupV3Directory creates a valid V3 directory structure for testing.
func setupV3Directory(t *testing.T, storePath string) {
	t.Helper()

	dirs := []string{
		storePath,
		filepath.Join(storePath, "tasks"),
		filepath.Join(storePath, "archive"),
		filepath.Join(storePath, "context"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", d, err)
		}
	}

	// Write config
	config := DirConfig{Version: 3}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(storePath, "config.json"), data, 0644)

	// Initialize empty context files
	os.WriteFile(filepath.Join(storePath, "context", "learnings.json"), []byte("[]"), 0644)
	os.WriteFile(filepath.Join(storePath, "context", "decisions.json"), []byte("[]"), 0644)
}
