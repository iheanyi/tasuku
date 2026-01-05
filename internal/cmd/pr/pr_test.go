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
