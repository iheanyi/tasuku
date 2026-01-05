package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestDirStore_Init(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, ".tasuku")
	s := NewDirStore(storePath)

	// Should not exist initially
	if s.Exists() {
		t.Fatal("store should not exist before init")
	}

	// Init should create directory structure
	if err := s.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Should exist after init
	if !s.Exists() {
		t.Fatal("store should exist after init")
	}

	// Verify directory structure
	dirs := []string{
		storePath,
		filepath.Join(storePath, "tasks"),
		filepath.Join(storePath, "archive"),
		filepath.Join(storePath, "context"),
	}
	for _, d := range dirs {
		if info, err := os.Stat(d); err != nil || !info.IsDir() {
			t.Errorf("expected directory %s to exist", d)
		}
	}

	// Verify config file
	configPath := filepath.Join(storePath, "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config.json should exist: %v", err)
	}

	// Init again should fail
	if err := s.Init(); err == nil {
		t.Fatal("Init should fail if already exists")
	}
}

func TestDirStore_AddAndReadTask(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	// Add a task
	err := s.AddTask("test-task", "Test description")
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Read all tasks
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if len(f.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(f.Tasks))
	}

	task, exists := f.Tasks["test-task"]
	if !exists {
		t.Fatal("task should exist")
	}

	if task.Description != "Test description" {
		t.Errorf("expected description 'Test description', got %q", task.Description)
	}

	if task.Status != "ready" {
		t.Errorf("expected status 'ready', got %q", task.Status)
	}
}

func TestDirStore_AddTaskWithPriority(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	priority := 1
	err := s.AddTaskWithPriority("high-priority", "High priority task", &priority)
	if err != nil {
		t.Fatalf("AddTaskWithPriority failed: %v", err)
	}

	f, _ := s.Read()
	task := f.Tasks["high-priority"]
	if task.Priority == nil || *task.Priority != 1 {
		t.Errorf("expected priority 1, got %v", task.Priority)
	}
}

func TestDirStore_AddTaskWithTags(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	tags := []string{"bug", "urgent"}
	err := s.AddTaskWithTags("tagged-task", "Task with tags", nil, tags)
	if err != nil {
		t.Fatalf("AddTaskWithTags failed: %v", err)
	}

	f, _ := s.Read()
	task := f.Tasks["tagged-task"]
	if len(task.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(task.Tags))
	}
}

func TestDirStore_DuplicateTask(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("dup-task", "First")
	err := s.AddTask("dup-task", "Second")
	if err == nil {
		t.Fatal("should fail when adding duplicate task")
	}
}

func TestDirStore_SetStatus(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("status-task", "Test")

	// ready -> in_progress
	err := s.SetStatus("status-task", task.StatusInProgress)
	if err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["status-task"].Status != task.StatusInProgress {
		t.Errorf("expected in_progress, got %s", f.Tasks["status-task"].Status)
	}

	// in_progress -> done
	err = s.SetStatus("status-task", task.StatusDone)
	if err != nil {
		t.Fatalf("SetStatus to done failed: %v", err)
	}
}

func TestDirStore_SetDescription(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("desc-task", "Original")
	err := s.SetDescription("desc-task", "Updated")
	if err != nil {
		t.Fatalf("SetDescription failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["desc-task"].Description != "Updated" {
		t.Errorf("expected 'Updated', got %q", f.Tasks["desc-task"].Description)
	}
}

func TestDirStore_BlockUnblock(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("blocker", "Blocker task")
	s.AddTask("blocked", "Blocked task")

	err := s.BlockTask("blocked", []string{"blocker"})
	if err != nil {
		t.Fatalf("BlockTask failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["blocked"].Status != task.StatusBlocked {
		t.Errorf("expected blocked status")
	}
	if len(f.Tasks["blocked"].BlockedBy) != 1 {
		t.Errorf("expected 1 blocker")
	}

	err = s.UnblockTask("blocked")
	if err != nil {
		t.Fatalf("UnblockTask failed: %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["blocked"].Status != task.StatusReady {
		t.Errorf("expected ready status after unblock")
	}
}

func TestDirStore_BlockNonexistentBlocker(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("task", "Task")
	err := s.BlockTask("task", []string{"nonexistent"})
	if err == nil {
		t.Fatal("should fail when blocker doesn't exist")
	}
}

func TestDirStore_Tags(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("tag-task", "Task")

	// Add tag
	err := s.AddTag("tag-task", "important")
	if err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	f, _ := s.Read()
	if !f.Tasks["tag-task"].HasTag("important") {
		t.Error("task should have 'important' tag")
	}

	// Add duplicate tag (should be no-op)
	err = s.AddTag("tag-task", "important")
	if err != nil {
		t.Fatalf("AddTag duplicate failed: %v", err)
	}

	// Remove tag
	err = s.RemoveTag("tag-task", "important")
	if err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["tag-task"].HasTag("important") {
		t.Error("task should not have 'important' tag after removal")
	}
}

func TestDirStore_Fields(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("field-task", "Task")

	err := s.SetField("field-task", "estimate", "2h")
	if err != nil {
		t.Fatalf("SetField failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["field-task"].Fields["estimate"] != "2h" {
		t.Error("field should be set")
	}

	err = s.RemoveField("field-task", "estimate")
	if err != nil {
		t.Fatalf("RemoveField failed: %v", err)
	}

	f, _ = s.Read()
	if _, exists := f.Tasks["field-task"].Fields["estimate"]; exists {
		t.Error("field should be removed")
	}
}

func TestDirStore_Timer(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("timer-task", "Task")

	// Start timer
	err := s.StartTimer("timer-task")
	if err != nil {
		t.Fatalf("StartTimer failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["timer-task"].TimerStart == nil {
		t.Error("timer should be running")
	}

	// Start again should fail
	err = s.StartTimer("timer-task")
	if err == nil {
		t.Error("starting timer twice should fail")
	}

	// Wait a bit and stop
	time.Sleep(10 * time.Millisecond)
	elapsed, err := s.StopTimer("timer-task")
	if err != nil {
		t.Fatalf("StopTimer failed: %v", err)
	}

	if elapsed < 10*time.Millisecond {
		t.Errorf("elapsed time should be at least 10ms, got %v", elapsed)
	}

	f, _ = s.Read()
	if f.Tasks["timer-task"].TimerStart != nil {
		t.Error("timer should be stopped")
	}
	if f.Tasks["timer-task"].Duration == 0 {
		t.Error("duration should be recorded")
	}
}

func TestDirStore_ClaimRelease(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("claim-task", "Task")

	// Claim task
	err := s.ClaimTask("claim-task", "agent-1")
	if err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["claim-task"].Owner == nil || *f.Tasks["claim-task"].Owner != "agent-1" {
		t.Error("task should be claimed by agent-1")
	}

	// Another agent claiming should fail
	err = s.ClaimTask("claim-task", "agent-2")
	if err == nil {
		t.Error("claiming already claimed task should fail")
	}

	// Release
	err = s.ReleaseTask("claim-task")
	if err != nil {
		t.Fatalf("ReleaseTask failed: %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["claim-task"].Owner != nil {
		t.Error("task should be released")
	}
}

func TestDirStore_Learnings(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	// Add learning
	id, err := s.AddLearning("Always use dependency injection")
	if err != nil {
		t.Fatalf("AddLearning failed: %v", err)
	}

	f, _ := s.Read()
	if len(f.Context.Learnings) != 1 {
		t.Fatalf("expected 1 learning, got %d", len(f.Context.Learnings))
	}

	// Check rule detection
	if !f.Context.Learnings[0].IsRule {
		t.Error("learning starting with 'Always' should be detected as rule")
	}

	// Find by text
	found, err := s.FindLearningByText("dependency")
	if err != nil {
		t.Fatalf("FindLearningByText failed: %v", err)
	}
	if found.ID != id {
		t.Error("should find the learning by text")
	}

	// Remove learning
	_, err = s.RemoveLearning(id)
	if err != nil {
		t.Fatalf("RemoveLearning failed: %v", err)
	}

	f, _ = s.Read()
	if len(f.Context.Learnings) != 0 {
		t.Error("learning should be removed")
	}
}

func TestDirStore_Decisions(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	d := task.Decision{
		ID:        "db-choice",
		Chose:     "PostgreSQL",
		Over:      []string{"MySQL", "SQLite"},
		Because:   "Better JSON support",
		CreatedAt: time.Now().UTC(),
	}

	err := s.AddDecision(d)
	if err != nil {
		t.Fatalf("AddDecision failed: %v", err)
	}

	f, _ := s.Read()
	if len(f.Context.Decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(f.Context.Decisions))
	}
	if f.Context.Decisions[0].Chose != "PostgreSQL" {
		t.Error("decision should be recorded correctly")
	}
}

func TestDirStore_Notes(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("note-task", "Task")

	// Add note
	noteID, err := s.AddNote("note-task", "This is a note")
	if err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	f, _ := s.Read()
	if len(f.Context.Notes["note-task"]) != 1 {
		t.Error("note should be added")
	}

	// Remove note
	_, err = s.RemoveNote("note-task", noteID)
	if err != nil {
		t.Fatalf("RemoveNote failed: %v", err)
	}

	f, _ = s.Read()
	if len(f.Context.Notes["note-task"]) != 0 {
		t.Error("note should be removed")
	}
}

func TestDirStore_NoteForNonexistentTask(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	_, err := s.AddNote("nonexistent", "Note")
	if err == nil {
		t.Error("adding note to nonexistent task should fail")
	}
}

func TestDirStore_Archive(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("archive-task", "Task to archive")
	s.SetStatus("archive-task", task.StatusInProgress)
	s.SetStatus("archive-task", task.StatusDone)

	// Archive the task
	err := s.ArchiveTask("archive-task", "Completed successfully")
	if err != nil {
		t.Fatalf("ArchiveTask failed: %v", err)
	}

	// Task should be gone from active
	f, _ := s.Read()
	if _, exists := f.Tasks["archive-task"]; exists {
		t.Error("archived task should not be in active tasks")
	}

	// Task should be in archive
	if _, exists := f.Archive["archive-task"]; !exists {
		t.Error("task should be in archive")
	}

	// Verify archive file exists
	archivePath := filepath.Join(dir, ".tasuku", "archive", "archive-task.json")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archive file should exist: %v", err)
	}

	// Task file should be gone
	taskPath := filepath.Join(dir, ".tasuku", "tasks", "archive-task.json")
	if _, err := os.Stat(taskPath); !os.IsNotExist(err) {
		t.Error("task file should be deleted")
	}
}

func TestDirStore_ArchiveNonDoneTask(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("ready-task", "Not done yet")

	err := s.ArchiveTask("ready-task", "")
	if err == nil {
		t.Error("archiving non-done task should fail")
	}
}

func TestDirStore_RestoreTask(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("restore-task", "Task to restore")
	s.SetStatus("restore-task", task.StatusInProgress)
	s.SetStatus("restore-task", task.StatusDone)
	s.ArchiveTask("restore-task", "")

	// Restore the task
	err := s.RestoreTask("restore-task")
	if err != nil {
		t.Fatalf("RestoreTask failed: %v", err)
	}

	// Task should be back in active with ready status
	f, _ := s.Read()
	task, exists := f.Tasks["restore-task"]
	if !exists {
		t.Fatal("restored task should be in active tasks")
	}
	if task.Status != "ready" {
		t.Errorf("restored task should have ready status, got %s", task.Status)
	}

	// Task should be gone from archive
	if _, exists := f.Archive["restore-task"]; exists {
		t.Error("restored task should not be in archive")
	}
}

func TestDirStore_GetArchivedTasks(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	// Archive two tasks
	s.AddTask("task1", "Task 1")
	s.SetStatus("task1", task.StatusInProgress)
	s.SetStatus("task1", task.StatusDone)
	s.ArchiveTask("task1", "Summary 1")

	s.AddTask("task2", "Task 2")
	s.SetStatus("task2", task.StatusInProgress)
	s.SetStatus("task2", task.StatusDone)
	s.ArchiveTask("task2", "Summary 2")

	archived, err := s.GetArchivedTasks()
	if err != nil {
		t.Fatalf("GetArchivedTasks failed: %v", err)
	}

	if len(archived) != 2 {
		t.Errorf("expected 2 archived tasks, got %d", len(archived))
	}
}

func TestDirStore_ClearArchive(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("task", "Task")
	s.SetStatus("task", task.StatusInProgress)
	s.SetStatus("task", task.StatusDone)
	s.ArchiveTask("task", "")

	count, err := s.ClearArchive()
	if err != nil {
		t.Fatalf("ClearArchive failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected to clear 1 task, got %d", count)
	}

	archived, _ := s.GetArchivedTasks()
	if len(archived) != 0 {
		t.Error("archive should be empty")
	}
}

func TestDirStore_DeleteTask(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	s.AddTask("delete-me", "Task to delete")

	err := s.DeleteTask("delete-me")
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	f, _ := s.Read()
	if _, exists := f.Tasks["delete-me"]; exists {
		t.Error("deleted task should not exist")
	}

	// Delete nonexistent should fail
	err = s.DeleteTask("nonexistent")
	if err == nil {
		t.Error("deleting nonexistent task should fail")
	}
}

func TestDirStore_ParallelSafety(t *testing.T) {
	dir := t.TempDir()
	s := NewDirStore(filepath.Join(dir, ".tasuku"))
	s.Init()

	// Add tasks in parallel (each task is a different file - should not conflict)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			id := filepath.Base(t.Name()) + "-" + string(rune('a'+n))
			s.AddTask(id, "Parallel task")
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	f, _ := s.Read()
	if len(f.Tasks) != 10 {
		t.Errorf("expected 10 tasks, got %d", len(f.Tasks))
	}
}
