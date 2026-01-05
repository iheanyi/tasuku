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

func TestStore_AddLearningWithRule_AutoDetect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Test auto-detection of rule learning (starts with Never)
	id, isRule, err := s.AddLearningWithRule("Never use raw SQL queries", nil)
	if err != nil {
		t.Fatalf("add learning failed: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty learning ID")
	}
	if !isRule {
		t.Error("expected isRule to be true for 'Never' learning")
	}

	// Verify it was saved correctly
	f, _ := s.Read()
	found := false
	for _, l := range f.Context.Learnings {
		if l.ID == id {
			found = true
			if !l.IsRule {
				t.Error("expected IsRule to be true in stored learning")
			}
		}
	}
	if !found {
		t.Error("learning not found in store")
	}
}

func TestStore_AddLearningWithRule_AutoDetect_Always(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Test auto-detection of rule learning (starts with Always)
	id, isRule, err := s.AddLearningWithRule("Always validate input", nil)
	if err != nil {
		t.Fatalf("add learning failed: %v", err)
	}
	if !isRule {
		t.Errorf("expected isRule to be true for 'Always' learning, id=%s", id)
	}
}

func TestStore_AddLearningWithRule_AutoDetect_NoRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Test auto-detection of non-rule learning
	_, isRule, err := s.AddLearningWithRule("Redis improves cache performance", nil)
	if err != nil {
		t.Fatalf("add learning failed: %v", err)
	}
	if isRule {
		t.Error("expected isRule to be false for regular learning")
	}
}

func TestStore_AddLearningWithRule_ForceRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Force a learning to be a rule even if it doesn't match patterns
	forceTrue := true
	id, isRule, err := s.AddLearningWithRule("This is important", &forceTrue)
	if err != nil {
		t.Fatalf("add learning failed: %v", err)
	}
	if !isRule {
		t.Error("expected isRule to be true when forced")
	}

	// Verify it was saved correctly
	f, _ := s.Read()
	for _, l := range f.Context.Learnings {
		if l.ID == id {
			if !l.IsRule {
				t.Error("expected IsRule to be true in stored learning when forced")
			}
		}
	}
}

func TestStore_AddLearningWithRule_ForceNotRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Force a learning to NOT be a rule even if it matches patterns
	forceFalse := false
	_, isRule, err := s.AddLearningWithRule("Never use eval()", &forceFalse)
	if err != nil {
		t.Fatalf("add learning failed: %v", err)
	}
	if isRule {
		t.Error("expected isRule to be false when forced to false")
	}
}

func TestStore_ErrNotInitialized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)

	// Read on non-existent file should return ErrNotInitialized
	_, err := s.Read()
	if err == nil {
		t.Fatal("expected error when reading non-existent file")
	}
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}

	// Update on non-existent file should return ErrNotInitialized
	err = s.AddTask("test", "Test task")
	if err == nil {
		t.Fatal("expected error when adding task to non-existent file")
	}
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}

	// Verify the error message is helpful
	expectedMsg := "no .tasuku.json found in current directory - run 'tk init' to create one"
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
