package pr

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestPRCmdStructure(t *testing.T) {
	if Cmd.Use != "pr" {
		t.Errorf("expected Use to be 'pr', got %s", Cmd.Use)
	}

	// Check subcommands exist
	subcommands := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subcommands[sub.Use] = true
	}

	if !subcommands["create"] {
		t.Error("expected 'create' subcommand")
	}
	if !subcommands["list"] {
		t.Error("expected 'list' subcommand")
	}
}

func TestCreateCmdFlags(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Use == "create" {
			taskFlag := sub.Flags().Lookup("task")
			if taskFlag == nil {
				t.Error("expected --task flag on create command")
			}

			doneFlag := sub.Flags().Lookup("done")
			if doneFlag == nil {
				t.Error("expected --done flag on create command")
			}
			break
		}
	}
}

func TestHasGhCLI(t *testing.T) {
	// This test just verifies the function doesn't panic
	// The actual result depends on whether gh is installed
	_ = hasGhCLI()
}

func TestBuildTaskContext(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("test-task", "Test task description")
	h.AddNote("test-task", "Implementation note")

	f, _ := h.Store().Read()
	task := f.Tasks["test-task"]

	context := buildTaskContext("test-task", task, f)

	if context == "" {
		t.Error("expected non-empty task context")
	}

	// Check expected content
	if !contains(context, "test-task") {
		t.Error("expected task ID in context")
	}
	if !contains(context, "Test task description") {
		t.Error("expected task description in context")
	}
	if !contains(context, "Implementation note") {
		t.Error("expected note in context")
	}
	if !contains(context, "Tasuku") {
		t.Error("expected Tasuku attribution")
	}
}

func TestBuildTaskContextWithPriority(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithPriority("high-priority", "High priority task", 0)

	f, _ := h.Store().Read()
	task := f.Tasks["high-priority"]

	context := buildTaskContext("high-priority", task, f)

	if !contains(context, "Priority") {
		t.Error("expected priority in context for prioritized task")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestBuildTaskContextWithTags(t *testing.T) {
	h := testutil.New(t)

	// Add a task with tags
	h.Store().AddTaskWithTags("tagged-task", "Task with tags", nil, []string{"bug", "urgent"})

	f, _ := h.Store().Read()
	task := f.Tasks["tagged-task"]

	context := buildTaskContext("tagged-task", task, f)

	if !contains(context, "Tags") {
		t.Error("expected tags in context for tagged task")
	}
	if !contains(context, "bug") {
		t.Error("expected 'bug' tag in context")
	}
	if !contains(context, "urgent") {
		t.Error("expected 'urgent' tag in context")
	}
}

func TestBuildTaskContextWithoutNotes(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("no-notes-task", "Task without notes")

	f, _ := h.Store().Read()
	task := f.Tasks["no-notes-task"]

	context := buildTaskContext("no-notes-task", task, f)

	// Should not contain Notes section
	if contains(context, "### Notes") {
		t.Error("did not expect Notes section for task without notes")
	}
}

func TestBuildTaskContextContainsTaskSection(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("test-task", "Test description")

	f, _ := h.Store().Read()
	task := f.Tasks["test-task"]

	context := buildTaskContext("test-task", task, f)

	if !contains(context, "## Task") {
		t.Error("expected Task section header")
	}
	if !contains(context, "**ID:**") {
		t.Error("expected ID field")
	}
	if !contains(context, "**Description:**") {
		t.Error("expected Description field")
	}
	if !contains(context, "**Status:**") {
		t.Error("expected Status field")
	}
}

func TestCreateCmdTaskNotFound(t *testing.T) {
	h := testutil.New(t)

	// Try to create PR with non-existent task (only if gh is available)
	if !hasGhCLI() {
		t.Skip("gh CLI not installed")
	}

	err := h.Execute(Cmd, "create", "--task", "non-existent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "task not found")
}

func TestListCmdExists(t *testing.T) {
	// Verify list subcommand exists
	subcommands := make(map[string]bool)
	for _, sub := range Cmd.Commands() {
		subcommands[sub.Use] = true
	}

	if !subcommands["list"] {
		t.Error("expected 'list' subcommand")
	}
}

func TestBuildTaskContextWithMultipleNotes(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("multi-note-task", "Task with multiple notes")
	h.AddNote("multi-note-task", "First note")
	h.AddNote("multi-note-task", "Second note")

	f, _ := h.Store().Read()
	task := f.Tasks["multi-note-task"]

	context := buildTaskContext("multi-note-task", task, f)

	if !contains(context, "First note") {
		t.Error("expected first note in context")
	}
	if !contains(context, "Second note") {
		t.Error("expected second note in context")
	}
}
