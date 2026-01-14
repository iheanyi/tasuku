package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	expectedMsg := "no Tasuku storage found - run 'tk init' to create one"
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

// =============================================================================
// Archive Tests
// =============================================================================

func TestStore_ArchiveTask_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add a task and mark it done
	if err := s.AddTask("test-task", "Test description"); err != nil {
		t.Fatalf("add task failed: %v", err)
	}
	if err := s.SetStatus("test-task", task.StatusInProgress); err != nil {
		t.Fatalf("set status to in_progress failed: %v", err)
	}
	if err := s.SetStatus("test-task", task.StatusDone); err != nil {
		t.Fatalf("set status to done failed: %v", err)
	}

	// Archive the task
	if err := s.ArchiveTask("test-task", "Completed successfully"); err != nil {
		t.Fatalf("archive task failed: %v", err)
	}

	// Verify task is no longer in active tasks
	f, _ := s.Read()
	if _, exists := f.Tasks["test-task"]; exists {
		t.Error("task should not exist in active tasks after archiving")
	}

	// Verify task is in archive
	archived, exists := f.Archive["test-task"]
	if !exists {
		t.Fatal("task should exist in archive")
	}

	if archived.Summary != "Completed successfully" {
		t.Errorf("wrong summary: got %q, want %q", archived.Summary, "Completed successfully")
	}

	if archived.Description != "Test description" {
		t.Errorf("wrong description: got %q, want %q", archived.Description, "Test description")
	}

	if archived.ArchivedAt.IsZero() {
		t.Error("archived_at should be set")
	}
}

func TestStore_ArchiveTask_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	err := s.ArchiveTask("nonexistent", "summary")
	if err == nil {
		t.Fatal("expected error when archiving nonexistent task")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestStore_ArchiveTask_NotDone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add a task but don't mark it done
	if err := s.AddTask("test-task", "Test description"); err != nil {
		t.Fatalf("add task failed: %v", err)
	}

	// Try to archive a ready task
	err := s.ArchiveTask("test-task", "summary")
	if err == nil {
		t.Fatal("expected error when archiving non-done task")
	}

	if !strings.Contains(err.Error(), "must be done") {
		t.Errorf("error should mention 'must be done': %v", err)
	}

	// Try with in_progress status
	if err := s.SetStatus("test-task", task.StatusInProgress); err != nil {
		t.Fatalf("set status failed: %v", err)
	}

	err = s.ArchiveTask("test-task", "summary")
	if err == nil {
		t.Fatal("expected error when archiving in_progress task")
	}
}

func TestStore_ArchiveTask_PreservesSummaryAndTotalTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add a task with duration
	if err := s.AddTask("test-task", "Test description"); err != nil {
		t.Fatalf("add task failed: %v", err)
	}

	// Start and stop timer to accumulate duration
	if err := s.StartTimer("test-task"); err != nil {
		t.Fatalf("start timer failed: %v", err)
	}

	// Manually set some duration by updating the file
	if err := s.Update(func(f *task.File) error {
		t := f.Tasks["test-task"]
		t.Duration = task.Duration(time.Hour + 30*time.Minute)
		t.TimerStart = nil
		f.Tasks["test-task"] = t
		return nil
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Mark done
	if err := s.SetStatus("test-task", task.StatusInProgress); err != nil {
		t.Fatalf("set status to in_progress failed: %v", err)
	}
	if err := s.SetStatus("test-task", task.StatusDone); err != nil {
		t.Fatalf("set status to done failed: %v", err)
	}

	// Archive with summary
	if err := s.ArchiveTask("test-task", "Implemented feature X"); err != nil {
		t.Fatalf("archive task failed: %v", err)
	}

	// Verify archived task preserves data
	archived, err := s.GetArchivedTask("test-task")
	if err != nil {
		t.Fatalf("get archived task failed: %v", err)
	}

	if archived.Summary != "Implemented feature X" {
		t.Errorf("wrong summary: got %q, want %q", archived.Summary, "Implemented feature X")
	}

	expectedDuration := time.Hour + 30*time.Minute
	if archived.TotalTime.TimeDuration() != expectedDuration {
		t.Errorf("wrong total time: got %v, want %v", archived.TotalTime.TimeDuration(), expectedDuration)
	}
}

func TestStore_ArchiveDoneTasks_OlderThanCutoff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add and complete two tasks
	s.AddTask("old-task", "Old task")
	s.AddTask("new-task", "New task")

	// Set both to done
	s.SetStatus("old-task", task.StatusInProgress)
	s.SetStatus("old-task", task.StatusDone)
	s.SetStatus("new-task", task.StatusInProgress)
	s.SetStatus("new-task", task.StatusDone)

	// Manually set old-task's UpdatedAt to be old
	s.Update(func(f *task.File) error {
		t := f.Tasks["old-task"]
		t.UpdatedAt = time.Now().Add(-48 * time.Hour) // 2 days ago
		f.Tasks["old-task"] = t
		return nil
	})

	// Archive tasks older than 24 hours
	archived, err := s.ArchiveDoneTasks(24 * time.Hour)
	if err != nil {
		t.Fatalf("archive done tasks failed: %v", err)
	}

	// Should only archive old-task
	if len(archived) != 1 {
		t.Fatalf("expected 1 archived task, got %d", len(archived))
	}

	if archived[0] != "old-task" {
		t.Errorf("expected 'old-task' to be archived, got %q", archived[0])
	}

	// Verify old-task is in archive
	f, _ := s.Read()
	if _, exists := f.Archive["old-task"]; !exists {
		t.Error("old-task should be in archive")
	}

	// Verify new-task is still active
	if _, exists := f.Tasks["new-task"]; !exists {
		t.Error("new-task should still be in active tasks")
	}
}

func TestStore_ArchiveDoneTasks_SkipsNewerThanCutoff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add and complete a task (will have recent UpdatedAt)
	s.AddTask("recent-task", "Recent task")
	s.SetStatus("recent-task", task.StatusInProgress)
	s.SetStatus("recent-task", task.StatusDone)

	// Archive tasks older than 24 hours
	archived, err := s.ArchiveDoneTasks(24 * time.Hour)
	if err != nil {
		t.Fatalf("archive done tasks failed: %v", err)
	}

	// Should archive nothing
	if len(archived) != 0 {
		t.Errorf("expected 0 archived tasks, got %d", len(archived))
	}

	// Task should still be active
	f, _ := s.Read()
	if _, exists := f.Tasks["recent-task"]; !exists {
		t.Error("recent-task should still be in active tasks")
	}
}

func TestStore_RestoreTask_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add, complete, and archive a task
	s.AddTask("test-task", "Test description")
	s.SetStatus("test-task", task.StatusInProgress)
	s.SetStatus("test-task", task.StatusDone)
	s.ArchiveTask("test-task", "Completed")

	// Restore the task
	if err := s.RestoreTask("test-task"); err != nil {
		t.Fatalf("restore task failed: %v", err)
	}

	// Verify task is back in active tasks
	f, _ := s.Read()
	restoredTask, exists := f.Tasks["test-task"]
	if !exists {
		t.Fatal("task should exist in active tasks after restore")
	}

	// Restored task should have status "ready"
	if restoredTask.Status != task.StatusReady {
		t.Errorf("restored task should have status 'ready', got %q", restoredTask.Status)
	}

	// Verify task is no longer in archive
	if _, exists := f.Archive["test-task"]; exists {
		t.Error("task should not exist in archive after restore")
	}
}

func TestStore_RestoreTask_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	err := s.RestoreTask("nonexistent")
	if err == nil {
		t.Fatal("expected error when restoring nonexistent archived task")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestStore_RestoreTask_IDConflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add, complete, and archive a task
	s.AddTask("test-task", "Original task")
	s.SetStatus("test-task", task.StatusInProgress)
	s.SetStatus("test-task", task.StatusDone)
	s.ArchiveTask("test-task", "Completed")

	// Create a new task with the same ID
	s.AddTask("test-task", "New task with same ID")

	// Try to restore - should fail due to ID conflict
	err := s.RestoreTask("test-task")
	if err == nil {
		t.Fatal("expected error when restoring to existing task ID")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists': %v", err)
	}
}

func TestStore_GetArchivedTasks_EmptyArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	archived, err := s.GetArchivedTasks()
	if err != nil {
		t.Fatalf("get archived tasks failed: %v", err)
	}

	if archived == nil {
		t.Fatal("archived tasks map should not be nil")
	}

	if len(archived) != 0 {
		t.Errorf("expected 0 archived tasks, got %d", len(archived))
	}
}

func TestStore_GetArchivedTask_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add, complete, and archive a task
	s.AddTask("test-task", "Test description")
	s.SetStatus("test-task", task.StatusInProgress)
	s.SetStatus("test-task", task.StatusDone)
	s.ArchiveTask("test-task", "Summary here")

	// Get the archived task
	archived, err := s.GetArchivedTask("test-task")
	if err != nil {
		t.Fatalf("get archived task failed: %v", err)
	}

	if archived.Description != "Test description" {
		t.Errorf("wrong description: got %q, want %q", archived.Description, "Test description")
	}

	if archived.Summary != "Summary here" {
		t.Errorf("wrong summary: got %q, want %q", archived.Summary, "Summary here")
	}
}

func TestStore_GetArchivedTask_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	_, err := s.GetArchivedTask("nonexistent")
	if err == nil {
		t.Fatal("expected error when getting nonexistent archived task")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestStore_ClearArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Add and archive multiple tasks
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("task-%d", i)
		s.AddTask(id, "Description")
		s.SetStatus(id, task.StatusInProgress)
		s.SetStatus(id, task.StatusDone)
		s.ArchiveTask(id, "Summary")
	}

	// Verify archive has tasks
	archived, _ := s.GetArchivedTasks()
	if len(archived) != 3 {
		t.Fatalf("expected 3 archived tasks, got %d", len(archived))
	}

	// Clear archive
	count, err := s.ClearArchive()
	if err != nil {
		t.Fatalf("clear archive failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected count of 3, got %d", count)
	}

	// Verify archive is empty
	archived, _ = s.GetArchivedTasks()
	if len(archived) != 0 {
		t.Errorf("expected 0 archived tasks after clear, got %d", len(archived))
	}
}

func TestStore_ClearArchive_EmptyArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	// Clear empty archive
	count, err := s.ClearArchive()
	if err != nil {
		t.Fatalf("clear archive failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected count of 0 for empty archive, got %d", count)
	}
}

func TestStore_AddDecision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	d := task.Decision{
		ID:      "test-decision",
		Chose:   "Option A",
		Over:    []string{"Option B", "Option C"},
		Because: "It was the best choice",
	}

	err := s.AddDecision(d)
	if err != nil {
		t.Fatalf("add decision failed: %v", err)
	}

	f, _ := s.Read()
	if len(f.Context.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(f.Context.Decisions))
	}

	if f.Context.Decisions[0].Chose != "Option A" {
		t.Errorf("wrong chose: %s", f.Context.Decisions[0].Chose)
	}
}

// Note: RemoveDecision is not implemented in Store - decisions are removed via
// the decision CLI command which manipulates the file directly

func TestStore_AddNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("note-task", "Task with note")

	noteID, err := s.AddNote("note-task", "This is a note")
	if err != nil {
		t.Fatalf("add note failed: %v", err)
	}
	if noteID == "" {
		t.Error("expected non-empty note ID")
	}

	f, _ := s.Read()
	notes := f.Context.Notes["note-task"]
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Text != "This is a note" {
		t.Errorf("wrong note text: %s", notes[0].Text)
	}
}

func TestStore_AddNoteTaskNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	_, err := s.AddNote("nonexistent", "Note text")
	if err == nil {
		t.Error("expected error when adding note to non-existent task")
	}
}

func TestStore_RemoveNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("note-task", "Task")
	noteID, _ := s.AddNote("note-task", "Note to remove")

	removed, err := s.RemoveNote("note-task", noteID)
	if err != nil {
		t.Fatalf("remove note failed: %v", err)
	}
	if removed != "Note to remove" {
		t.Errorf("wrong removed text: %s", removed)
	}

	f, _ := s.Read()
	if len(f.Context.Notes["note-task"]) != 0 {
		t.Error("note should be removed")
	}
}

func TestStore_BlockTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("blocker", "Blocker task")
	s.AddTask("blocked", "Blocked task")

	err := s.BlockTask("blocked", []string{"blocker"})
	if err != nil {
		t.Fatalf("block task failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["blocked"].Status != task.StatusBlocked {
		t.Error("task should be blocked")
	}
	if len(f.Tasks["blocked"].BlockedBy) != 1 {
		t.Error("task should have one blocker")
	}
}

func TestStore_UnblockTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("blocker", "Blocker")
	s.AddTask("blocked", "Blocked")
	s.BlockTask("blocked", []string{"blocker"})

	err := s.UnblockTask("blocked")
	if err != nil {
		t.Fatalf("unblock task failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["blocked"].Status != task.StatusReady {
		t.Error("task should be ready after unblock")
	}
}

func TestStore_SetOwner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("owned-task", "Task")

	err := s.SetOwner("owned-task", "test-owner")
	if err != nil {
		t.Fatalf("set owner failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["owned-task"].Owner == nil || *f.Tasks["owned-task"].Owner != "test-owner" {
		t.Error("owner should be set")
	}

	// Clear owner
	err = s.ClearOwner("owned-task")
	if err != nil {
		t.Fatalf("clear owner failed: %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["owned-task"].Owner != nil {
		t.Error("owner should be cleared")
	}
}

func TestStore_SetTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("tag-task", "Task")

	err := s.AddTag("tag-task", "important")
	if err != nil {
		t.Fatalf("add tag failed: %v", err)
	}

	f, _ := s.Read()
	if !f.Tasks["tag-task"].HasTag("important") {
		t.Error("task should have tag")
	}
}

func TestStore_RemoveTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("tag-task", "Task")
	s.AddTag("tag-task", "remove-me")

	err := s.RemoveTag("tag-task", "remove-me")
	if err != nil {
		t.Fatalf("remove tag failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["tag-task"].HasTag("remove-me") {
		t.Error("tag should be removed")
	}
}

func TestStore_SetField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("field-task", "Task")

	err := s.SetField("field-task", "estimate", "2h")
	if err != nil {
		t.Fatalf("set field failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["field-task"].Fields["estimate"] != "2h" {
		t.Error("field should be set")
	}
}

func TestStore_RemoveField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("field-task", "Task")
	s.SetField("field-task", "estimate", "2h")

	err := s.RemoveField("field-task", "estimate")
	if err != nil {
		t.Fatalf("remove field failed: %v", err)
	}

	f, _ := s.Read()
	if _, exists := f.Tasks["field-task"].Fields["estimate"]; exists {
		t.Error("field should be removed")
	}
}

func TestStore_Timer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("timer-task", "Task")

	// Start timer
	err := s.StartTimer("timer-task")
	if err != nil {
		t.Fatalf("start timer failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["timer-task"].TimerStart == nil {
		t.Error("timer should be running")
	}

	// Stop timer
	elapsed, err := s.StopTimer("timer-task")
	if err != nil {
		t.Fatalf("stop timer failed: %v", err)
	}

	if elapsed < 0 {
		t.Error("elapsed time should be positive")
	}
}

func TestStore_TimerNotRunning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("no-timer", "Task")

	_, err := s.StopTimer("no-timer")
	if err == nil {
		t.Error("stopping non-running timer should fail")
	}
}

func TestStore_StartTimerAlreadyRunning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("timer-task", "Task")
	s.StartTimer("timer-task")

	err := s.StartTimer("timer-task")
	if err == nil {
		t.Error("starting timer twice should fail")
	}
}

func TestStore_ClaimRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("claim-task", "Task")

	err := s.ClaimTask("claim-task", "agent-1")
	if err != nil {
		t.Fatalf("claim task failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["claim-task"].Owner == nil || *f.Tasks["claim-task"].Owner != "agent-1" {
		t.Error("task should be claimed")
	}

	err = s.ReleaseTask("claim-task")
	if err != nil {
		t.Fatalf("release task failed: %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["claim-task"].Owner != nil {
		t.Error("task should be released")
	}
}

func TestStore_DeleteTask(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("delete-me", "Task to delete")

	err := s.DeleteTask("delete-me")
	if err != nil {
		t.Fatalf("delete task failed: %v", err)
	}

	f, _ := s.Read()
	if _, exists := f.Tasks["delete-me"]; exists {
		t.Error("task should be deleted")
	}
}

func TestStore_DeleteTaskNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	err := s.DeleteTask("nonexistent")
	if err == nil {
		t.Error("deleting nonexistent task should fail")
	}
}

func TestStore_SetDescription(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".tasuku.json")
	s := New(path)
	s.Init()

	s.AddTask("desc-task", "Original")

	err := s.SetDescription("desc-task", "Updated description")
	if err != nil {
		t.Fatalf("set description failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["desc-task"].Description != "Updated description" {
		t.Error("description should be updated")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
