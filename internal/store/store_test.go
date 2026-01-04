package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestStore_Init(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	if s.Exists() {
		t.Fatal("store should not exist before init")
	}

	if err := s.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if !s.Exists() {
		t.Fatal("store should exist after init")
	}

	// Second init should fail
	if err := s.Init(); err == nil {
		t.Fatal("second init should fail")
	}
}

func TestStore_AddTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	if err := s.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := s.AddTask("test-task", "Test description"); err != nil {
		t.Fatalf("add task failed: %v", err)
	}

	f, err := s.Read()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	task, ok := f.Tasks["test-task"]
	if !ok {
		t.Fatal("task not found")
	}

	if task.Description != "Test description" {
		t.Errorf("wrong description: %s", task.Description)
	}

	if task.Status != "ready" {
		t.Errorf("wrong status: %s", task.Status)
	}
}

func TestStore_SetStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	if err := s.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := s.AddTask("test-task", "Test"); err != nil {
		t.Fatalf("add task failed: %v", err)
	}

	// Valid transition: ready -> in_progress
	if err := s.SetStatus("test-task", task.StatusInProgress); err != nil {
		t.Fatalf("set status failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["test-task"].Status != task.StatusInProgress {
		t.Errorf("status not updated")
	}

	// Valid transition: in_progress -> done
	if err := s.SetStatus("test-task", task.StatusDone); err != nil {
		t.Fatalf("set status to done failed: %v", err)
	}
}

func TestStore_AddLearning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	id, err := s.AddLearning("Test learning")
	if err != nil {
		t.Fatalf("add learning failed: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty learning ID")
	}

	f, _ := s.Read()
	if len(f.Context.Learnings) != 1 {
		t.Fatalf("expected 1 learning, got %d", len(f.Context.Learnings))
	}

	if f.Context.Learnings[0].Text != "Test learning" {
		t.Errorf("wrong learning: %s", f.Context.Learnings[0].Text)
	}
}

func TestStore_RemoveLearning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add a learning
	id, err := s.AddLearning("Test learning to remove")
	if err != nil {
		t.Fatalf("add learning failed: %v", err)
	}

	// Remove it
	removedText, err := s.RemoveLearning(id)
	if err != nil {
		t.Fatalf("remove learning failed: %v", err)
	}

	if removedText != "Test learning to remove" {
		t.Errorf("wrong removed text: %s", removedText)
	}

	// Verify it's gone
	f, _ := s.Read()
	if len(f.Context.Learnings) != 0 {
		t.Errorf("expected 0 learnings, got %d", len(f.Context.Learnings))
	}

	// Try to remove non-existent learning
	_, err = s.RemoveLearning("nonexistent")
	if err == nil {
		t.Error("expected error when removing non-existent learning")
	}
}

func TestStore_FindLearningByText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add some learnings
	s.AddLearning("Redis connection pooling improves performance")
	s.AddLearning("Always validate user input")
	s.AddLearning("Use indexes for frequent queries")

	// Find by partial text
	learning, err := s.FindLearningByText("redis")
	if err != nil {
		t.Fatalf("find learning failed: %v", err)
	}

	if learning.Text != "Redis connection pooling improves performance" {
		t.Errorf("wrong learning found: %s", learning.Text)
	}

	// Case insensitive
	learning, err = s.FindLearningByText("REDIS")
	if err != nil {
		t.Fatalf("case insensitive find failed: %v", err)
	}

	if learning.Text != "Redis connection pooling improves performance" {
		t.Errorf("case insensitive search failed: %s", learning.Text)
	}

	// Not found
	_, err = s.FindLearningByText("nonexistent")
	if err == nil {
		t.Error("expected error when learning not found")
	}
}

func TestStore_ParallelAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Spawn multiple goroutines to add tasks
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			id := filepath.Join("task", string(rune('a'+n)))
			done <- s.AddTask(id, "Parallel task")
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("parallel add failed: %v", err)
		}
	}

	// Verify file is not corrupted
	f, err := s.Read()
	if err != nil {
		t.Fatalf("read after parallel failed: %v", err)
	}

	if len(f.Tasks) != 10 {
		t.Errorf("expected 10 tasks, got %d", len(f.Tasks))
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
