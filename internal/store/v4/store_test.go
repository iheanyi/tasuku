package v4

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestNewStore(t *testing.T) {
	s := New("/tmp/test-tasuku")
	if s.Path() != "/tmp/test-tasuku" {
		t.Errorf("Path() = %q, want /tmp/test-tasuku", s.Path())
	}
}

func TestStoreInit(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".tasuku")

	s := New(root)

	// Should not exist before init
	if s.Exists() {
		t.Error("Exists() should be false before Init")
	}

	// Init
	if err := s.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Should exist after init
	if !s.Exists() {
		t.Error("Exists() should be true after Init")
	}

	// Check directory structure
	for _, subdir := range []string{TasksDir, ArchiveDir, ContextDir} {
		path := filepath.Join(root, subdir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("Expected directory %s to exist", subdir)
		}
	}

	// Check config
	configPath := filepath.Join(root, ConfigFileName)
	if _, err := os.Stat(configPath); err != nil {
		t.Error("config.json should exist")
	}

	// Check index
	indexPath := filepath.Join(root, IndexFileName)
	if _, err := os.Stat(indexPath); err != nil {
		t.Error("index.json should exist")
	}

	// Check learnings.md
	learningsPath := filepath.Join(root, ContextDir, "learnings.md")
	if _, err := os.Stat(learningsPath); err != nil {
		t.Error("learnings.md should exist")
	}

	// Check decisions.md
	decisionsPath := filepath.Join(root, ContextDir, "decisions.md")
	if _, err := os.Stat(decisionsPath); err != nil {
		t.Error("decisions.md should exist")
	}

	// Init again should fail
	if err := s.Init(); err == nil {
		t.Error("Second Init() should fail")
	}
}

func TestStoreAddTask(t *testing.T) {
	s := setupTestStore(t)

	if err := s.AddTask("test-task", "Test task description"); err != nil {
		t.Fatalf("AddTask() error = %v", err)
	}

	// Verify task file exists
	taskPath := filepath.Join(s.root, TasksDir, "test-task.md")
	if _, err := os.Stat(taskPath); err != nil {
		t.Error("Task file should exist")
	}

	// Read it back
	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	task, exists := f.Tasks["test-task"]
	if !exists {
		t.Fatal("Task should exist")
	}
	if task.Description != "Test task description" {
		t.Errorf("Description = %q, want 'Test task description'", task.Description)
	}
	if task.Status != "ready" {
		t.Errorf("Status = %q, want 'ready'", task.Status)
	}

	// Add duplicate should fail
	if err := s.AddTask("test-task", "Duplicate"); err == nil {
		t.Error("Adding duplicate task should fail")
	}
}

func TestStoreAddTaskWithPriority(t *testing.T) {
	s := setupTestStore(t)

	priority := 1
	if err := s.AddTaskWithPriority("high-task", "High priority task", &priority); err != nil {
		t.Fatalf("AddTaskWithPriority() error = %v", err)
	}

	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	task := f.Tasks["high-task"]
	if task.Priority == nil || *task.Priority != 1 {
		t.Errorf("Priority = %v, want 1", task.Priority)
	}
}

func TestStoreAddTaskWithTags(t *testing.T) {
	s := setupTestStore(t)

	priority := 2
	tags := []string{"backend", "api"}
	if err := s.AddTaskWithTags("tagged-task", "Tagged task", &priority, tags); err != nil {
		t.Fatalf("AddTaskWithTags() error = %v", err)
	}

	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	task := f.Tasks["tagged-task"]
	if len(task.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(task.Tags))
	}
}

func TestStoreAddSubtask(t *testing.T) {
	s := setupTestStore(t)

	// Create parent first
	if err := s.AddTask("parent", "Parent task"); err != nil {
		t.Fatalf("AddTask() error = %v", err)
	}

	// Create subtask
	if err := s.AddSubtask("child", "Child task", "parent"); err != nil {
		t.Fatalf("AddSubtask() error = %v", err)
	}

	f, err := s.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	child := f.Tasks["child"]
	if child.ParentID == nil || *child.ParentID != "parent" {
		t.Error("Subtask should have parent")
	}

	// Subtask with missing parent should fail
	if err := s.AddSubtask("orphan", "Orphan", "missing-parent"); err == nil {
		t.Error("Subtask with missing parent should fail")
	}
}

func TestStoreSetStatus(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Ready -> InProgress
	if err := s.SetStatus("task-1", task.StatusInProgress); err != nil {
		t.Fatalf("SetStatus to in_progress error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].Status != task.StatusInProgress {
		t.Errorf("Status = %s, want in_progress", f.Tasks["task-1"].Status)
	}

	// InProgress -> Done
	if err := s.SetStatus("task-1", task.StatusDone); err != nil {
		t.Fatalf("SetStatus to done error = %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["task-1"].Status != task.StatusDone {
		t.Errorf("Status = %s, want done", f.Tasks["task-1"].Status)
	}
}

func TestStoreMarkDoneAndUnblock(t *testing.T) {
	s := setupTestStore(t)

	// Create blocker and blocked task
	s.AddTask("blocker", "Blocker task")
	s.SetStatus("blocker", task.StatusInProgress)

	s.AddTask("blocked", "Blocked task")
	s.BlockTask("blocked", []string{"blocker"})

	// Verify blocked
	f, _ := s.Read()
	if f.Tasks["blocked"].Status != task.StatusBlocked {
		t.Fatal("Task should be blocked")
	}

	// Mark blocker as done
	unblocked, err := s.MarkDoneAndUnblock("blocker")
	if err != nil {
		t.Fatalf("MarkDoneAndUnblock error = %v", err)
	}

	if len(unblocked) != 1 || unblocked[0] != "blocked" {
		t.Errorf("Unblocked = %v, want [blocked]", unblocked)
	}

	f, _ = s.Read()
	if f.Tasks["blocked"].Status != task.StatusReady {
		t.Errorf("Previously blocked task should be ready, got %s", f.Tasks["blocked"].Status)
	}
}

func TestStoreBlockAndUnblock(t *testing.T) {
	s := setupTestStore(t)

	s.AddTask("task-1", "Task 1")
	s.AddTask("blocker-1", "Blocker 1")
	s.AddTask("blocker-2", "Blocker 2")

	// Block
	if err := s.BlockTask("task-1", []string{"blocker-1", "blocker-2"}); err != nil {
		t.Fatalf("BlockTask error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].Status != task.StatusBlocked {
		t.Error("Task should be blocked")
	}
	if len(f.Tasks["task-1"].BlockedBy) != 2 {
		t.Errorf("BlockedBy count = %d, want 2", len(f.Tasks["task-1"].BlockedBy))
	}

	// Remove one blocker
	if err := s.RemoveBlocker("task-1", "blocker-1"); err != nil {
		t.Fatalf("RemoveBlocker error = %v", err)
	}

	f, _ = s.Read()
	if len(f.Tasks["task-1"].BlockedBy) != 1 {
		t.Errorf("BlockedBy count = %d, want 1", len(f.Tasks["task-1"].BlockedBy))
	}

	// Unblock completely
	if err := s.UnblockTask("task-1"); err != nil {
		t.Fatalf("UnblockTask error = %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["task-1"].Status != task.StatusReady {
		t.Error("Task should be ready after unblock")
	}
}

func TestStorePriority(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	if err := s.SetPriority("task-1", 0); err != nil {
		t.Fatalf("SetPriority error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].Priority == nil || *f.Tasks["task-1"].Priority != 0 {
		t.Error("Priority should be 0 (critical)")
	}
}

func TestStoreTags(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Add tag
	if err := s.AddTag("task-1", "backend"); err != nil {
		t.Fatalf("AddTag error = %v", err)
	}

	f, _ := s.Read()
	if len(f.Tasks["task-1"].Tags) != 1 {
		t.Errorf("Tags count = %d, want 1", len(f.Tasks["task-1"].Tags))
	}

	// Add same tag again (should be no-op)
	s.AddTag("task-1", "backend")
	f, _ = s.Read()
	if len(f.Tasks["task-1"].Tags) != 1 {
		t.Error("Duplicate tag should not be added")
	}

	// Remove tag
	if err := s.RemoveTag("task-1", "backend"); err != nil {
		t.Fatalf("RemoveTag error = %v", err)
	}

	f, _ = s.Read()
	if len(f.Tasks["task-1"].Tags) != 0 {
		t.Error("Tag should be removed")
	}
}

func TestStoreFields(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Set field
	if err := s.SetField("task-1", "estimate", "2h"); err != nil {
		t.Fatalf("SetField error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].Fields["estimate"] != "2h" {
		t.Error("Field should be set")
	}

	// Remove field
	if err := s.RemoveField("task-1", "estimate"); err != nil {
		t.Fatalf("RemoveField error = %v", err)
	}

	f, _ = s.Read()
	if _, exists := f.Tasks["task-1"].Fields["estimate"]; exists {
		t.Error("Field should be removed")
	}
}

func TestStoreTimer(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Start timer
	if err := s.StartTimer("task-1"); err != nil {
		t.Fatalf("StartTimer error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].TimerStart == nil {
		t.Error("Timer should be started")
	}

	// Starting again should fail
	if err := s.StartTimer("task-1"); err == nil {
		t.Error("Starting timer twice should fail")
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Stop timer
	elapsed, err := s.StopTimer("task-1")
	if err != nil {
		t.Fatalf("StopTimer error = %v", err)
	}

	if elapsed < 10*time.Millisecond {
		t.Errorf("Elapsed = %v, expected >= 10ms", elapsed)
	}

	f, _ = s.Read()
	if f.Tasks["task-1"].TimerStart != nil {
		t.Error("Timer should be stopped")
	}
	if f.Tasks["task-1"].Duration < task.Duration(10*time.Millisecond) {
		t.Error("Duration should be accumulated")
	}
}

func TestStoreOwnership(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Set owner
	if err := s.SetOwner("task-1", "alice"); err != nil {
		t.Fatalf("SetOwner error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].Owner == nil || *f.Tasks["task-1"].Owner != "alice" {
		t.Error("Owner should be alice")
	}

	// Clear owner
	if err := s.ClearOwner("task-1"); err != nil {
		t.Fatalf("ClearOwner error = %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["task-1"].Owner != nil {
		t.Error("Owner should be cleared")
	}
}

func TestStoreClaimAndRelease(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Claim
	if err := s.ClaimTask("task-1", "agent-1"); err != nil {
		t.Fatalf("ClaimTask error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].Owner == nil || *f.Tasks["task-1"].Owner != "agent-1" {
		t.Error("Owner should be agent-1")
	}
	if f.Tasks["task-1"].ClaimedAt == nil {
		t.Error("ClaimedAt should be set")
	}

	// Release
	if err := s.ReleaseTask("task-1"); err != nil {
		t.Fatalf("ReleaseTask error = %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["task-1"].Owner != nil {
		t.Error("Owner should be cleared after release")
	}
	if f.Tasks["task-1"].ClaimedAt != nil {
		t.Error("ClaimedAt should be cleared after release")
	}
}

func TestStoreNotes(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Add note
	noteID, err := s.AddNote("task-1", "This is a note")
	if err != nil {
		t.Fatalf("AddNote error = %v", err)
	}

	if noteID == "" {
		t.Error("NoteID should not be empty")
	}

	f, _ := s.Read()
	notes := f.Context.Notes["task-1"]
	if len(notes) != 1 {
		t.Fatalf("Notes count = %d, want 1", len(notes))
	}
	if notes[0].Text != "This is a note" {
		t.Errorf("Note text = %q, want 'This is a note'", notes[0].Text)
	}

	// Remove note
	text, err := s.RemoveNote("task-1", noteID)
	if err != nil {
		t.Fatalf("RemoveNote error = %v", err)
	}
	if text != "This is a note" {
		t.Errorf("Removed text = %q, want 'This is a note'", text)
	}

	f, _ = s.Read()
	if len(f.Context.Notes["task-1"]) != 0 {
		t.Error("Note should be removed")
	}
}

func TestStoreLearnings(t *testing.T) {
	s := setupTestStore(t)

	// Add learning
	id, err := s.AddLearning("Test insight")
	if err != nil {
		t.Fatalf("AddLearning error = %v", err)
	}

	if id == "" {
		t.Error("Learning ID should not be empty")
	}

	f, _ := s.Read()
	if len(f.Context.Learnings) != 1 {
		t.Fatalf("Learnings count = %d, want 1", len(f.Context.Learnings))
	}
	if f.Context.Learnings[0].Text != "Test insight" {
		t.Errorf("Learning text = %q, want 'Test insight'", f.Context.Learnings[0].Text)
	}

	// Add rule learning
	ruleID, isRule, err := s.AddLearningWithRule("Never do this", nil)
	if err != nil {
		t.Fatalf("AddLearningWithRule error = %v", err)
	}
	if !isRule {
		t.Error("Should be detected as rule (starts with 'Never')")
	}

	// Remove learning
	text, err := s.RemoveLearning(ruleID)
	if err != nil {
		t.Fatalf("RemoveLearning error = %v", err)
	}
	if text != "Never do this" {
		t.Errorf("Removed text = %q, want 'Never do this'", text)
	}
}

func TestStoreDecisions(t *testing.T) {
	s := setupTestStore(t)

	d := task.Decision{
		ID:        "test-decision",
		Chose:     "Option A",
		Over:      []string{"Option B", "Option C"},
		Because:   "It's better",
		CreatedAt: time.Now().UTC(),
	}

	if err := s.AddDecision(d); err != nil {
		t.Fatalf("AddDecision error = %v", err)
	}

	f, _ := s.Read()
	if len(f.Context.Decisions) != 1 {
		t.Fatalf("Decisions count = %d, want 1", len(f.Context.Decisions))
	}
	if f.Context.Decisions[0].Chose != "Option A" {
		t.Errorf("Decision chose = %q, want 'Option A'", f.Context.Decisions[0].Chose)
	}
}

func TestStoreArchive(t *testing.T) {
	s := setupTestStore(t)

	// Create and complete a task
	s.AddTask("task-1", "Task 1")
	s.SetStatus("task-1", task.StatusInProgress)
	s.SetStatus("task-1", task.StatusDone)

	// Archive
	if err := s.ArchiveTask("task-1", "Completed successfully"); err != nil {
		t.Fatalf("ArchiveTask error = %v", err)
	}

	// Should not be in active tasks
	f, _ := s.Read()
	if _, exists := f.Tasks["task-1"]; exists {
		t.Error("Task should not be in active tasks after archive")
	}

	// Should be in archive
	archived, err := s.GetArchivedTasks()
	if err != nil {
		t.Fatalf("GetArchivedTasks error = %v", err)
	}
	if _, exists := archived["task-1"]; !exists {
		t.Error("Task should be in archive")
	}

	// Restore
	if err := s.RestoreTask("task-1"); err != nil {
		t.Fatalf("RestoreTask error = %v", err)
	}

	f, _ = s.Read()
	if _, exists := f.Tasks["task-1"]; !exists {
		t.Error("Task should be back in active tasks after restore")
	}
	if f.Tasks["task-1"].Status != task.StatusReady {
		t.Errorf("Restored task should be ready, got %s", f.Tasks["task-1"].Status)
	}
}

func TestStoreDeleteTask(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	if err := s.DeleteTask("task-1"); err != nil {
		t.Fatalf("DeleteTask error = %v", err)
	}

	f, _ := s.Read()
	if _, exists := f.Tasks["task-1"]; exists {
		t.Error("Task should be deleted")
	}

	// Delete non-existent should fail
	if err := s.DeleteTask("nonexistent"); err == nil {
		t.Error("Deleting non-existent task should fail")
	}
}

func TestStoreIndexRegeneration(t *testing.T) {
	s := setupTestStore(t)

	// Add tasks
	s.AddTask("task-1", "Task 1")
	s.AddTask("task-2", "Task 2")

	// Add learning
	s.AddLearning("Test learning")

	// Add decision
	s.AddDecision(task.Decision{
		ID:      "test",
		Chose:   "A",
		Over:    []string{"B"},
		Because: "Why not",
	})

	// Archive a task
	s.SetStatus("task-1", task.StatusInProgress)
	s.SetStatus("task-1", task.StatusDone)
	s.ArchiveTask("task-1", "Done")

	// Read index
	idxPath := filepath.Join(s.root, IndexFileName)
	data, err := os.ReadFile(idxPath)
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	idx, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("Failed to parse index: %v", err)
	}

	if len(idx.Tasks) != 1 {
		t.Errorf("Index tasks count = %d, want 1", len(idx.Tasks))
	}
	if idx.ArchivedCount != 1 {
		t.Errorf("ArchivedCount = %d, want 1", idx.ArchivedCount)
	}
	if idx.LearningsCount != 1 {
		t.Errorf("LearningsCount = %d, want 1", idx.LearningsCount)
	}
	if idx.DecisionsCount != 1 {
		t.Errorf("DecisionsCount = %d, want 1", idx.DecisionsCount)
	}
}

func TestStoreSetParent(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("parent", "Parent task")
	s.AddTask("child", "Child task")

	// Set parent
	parent := "parent"
	if err := s.SetParent("child", &parent); err != nil {
		t.Fatalf("SetParent error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["child"].ParentID == nil || *f.Tasks["child"].ParentID != "parent" {
		t.Error("Child should have parent")
	}

	// Clear parent
	if err := s.SetParent("child", nil); err != nil {
		t.Fatalf("SetParent(nil) error = %v", err)
	}

	f, _ = s.Read()
	if f.Tasks["child"].ParentID != nil {
		t.Error("Parent should be cleared")
	}

	// Self-reference should fail
	self := "child"
	if err := s.SetParent("child", &self); err == nil {
		t.Error("Self-referencing parent should fail")
	}
}

func TestStoreGetSubtasks(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("parent", "Parent task")
	s.AddSubtask("child-1", "Child 1", "parent")
	s.AddSubtask("child-2", "Child 2", "parent")
	s.AddTask("unrelated", "Unrelated task")

	subtasks, err := s.GetSubtasks("parent")
	if err != nil {
		t.Fatalf("GetSubtasks error = %v", err)
	}

	if len(subtasks) != 2 {
		t.Errorf("Subtasks count = %d, want 2", len(subtasks))
	}

	if _, exists := subtasks["child-1"]; !exists {
		t.Error("child-1 should be in subtasks")
	}
	if _, exists := subtasks["child-2"]; !exists {
		t.Error("child-2 should be in subtasks")
	}
}

func TestStoreEditTask(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Original description")

	if err := s.EditTask("task-1", "Updated description"); err != nil {
		t.Fatalf("EditTask error = %v", err)
	}

	f, _ := s.Read()
	if f.Tasks["task-1"].Description != "Updated description" {
		t.Errorf("Description = %q, want 'Updated description'", f.Tasks["task-1"].Description)
	}
}

func TestStoreFindLearningByText(t *testing.T) {
	s := setupTestStore(t)
	s.AddLearning("Always test your code")
	s.AddLearning("Never skip validation")

	l, err := s.FindLearningByText("test")
	if err != nil {
		t.Fatalf("FindLearningByText error = %v", err)
	}
	if l == nil {
		t.Fatal("Should find learning")
	}
	if l.Text != "Always test your code" {
		t.Errorf("Found = %q, want 'Always test your code'", l.Text)
	}

	// Not found
	_, err = s.FindLearningByText("nonexistent")
	if err == nil {
		t.Error("Should return error for not found")
	}
}

func TestStoreGetActiveTimers(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")
	s.AddTask("task-2", "Task 2")

	s.StartTimer("task-1")

	timers, err := s.GetActiveTimers()
	if err != nil {
		t.Fatalf("GetActiveTimers error = %v", err)
	}

	if len(timers) != 1 {
		t.Errorf("Active timers count = %d, want 1", len(timers))
	}
	if _, exists := timers["task-1"]; !exists {
		t.Error("task-1 should have active timer")
	}
}

func TestStoreStopTimerIfRunning(t *testing.T) {
	s := setupTestStore(t)
	s.AddTask("task-1", "Task 1")

	// Stop when no timer running
	_, wasRunning, err := s.StopTimerIfRunning("task-1")
	if err != nil {
		t.Fatalf("StopTimerIfRunning error = %v", err)
	}
	if wasRunning {
		t.Error("Should not be running")
	}

	// Start and stop
	s.StartTimer("task-1")
	elapsed, wasRunning, err := s.StopTimerIfRunning("task-1")
	if err != nil {
		t.Fatalf("StopTimerIfRunning error = %v", err)
	}
	if !wasRunning {
		t.Error("Should have been running")
	}
	if elapsed == 0 {
		t.Error("Should have some elapsed time")
	}
}

func TestStoreArchiveDoneTasks(t *testing.T) {
	s := setupTestStore(t)

	// Create and complete tasks
	s.AddTask("old-task", "Old task")
	s.SetStatus("old-task", task.StatusInProgress)
	s.SetStatus("old-task", task.StatusDone)

	s.AddTask("new-task", "New task")
	s.SetStatus("new-task", task.StatusInProgress)
	s.SetStatus("new-task", task.StatusDone)

	// Archive tasks older than 0 (all done tasks)
	archived, err := s.ArchiveDoneTasks(0)
	if err != nil {
		t.Fatalf("ArchiveDoneTasks error = %v", err)
	}

	if len(archived) != 2 {
		t.Errorf("Archived count = %d, want 2", len(archived))
	}
}

func TestStoreClearArchive(t *testing.T) {
	s := setupTestStore(t)

	// Archive some tasks
	s.AddTask("task-1", "Task 1")
	s.SetStatus("task-1", task.StatusInProgress)
	s.SetStatus("task-1", task.StatusDone)
	s.ArchiveTask("task-1", "")

	// Clear
	count, err := s.ClearArchive()
	if err != nil {
		t.Fatalf("ClearArchive error = %v", err)
	}
	if count != 1 {
		t.Errorf("Cleared count = %d, want 1", count)
	}

	archived, _ := s.GetArchivedTasks()
	if len(archived) != 0 {
		t.Error("Archive should be empty")
	}
}

// Helper function to set up a test store
func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, ".tasuku")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	return s
}
