package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestListCmd_Empty(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("No tasks found")
}

func TestListCmd_WithTasks(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddTask("task-2", "Second task")

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("task-1")
	h.AssertOutputContains("task-2")
	h.AssertOutputContains("First task")
	h.AssertOutputContains("Second task")
}

func TestListCmd_StatusFilter(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("ready-task", "Ready task", itask.StatusReady)
	h.AddTaskWithStatus("done-task", "Done task", itask.StatusDone)
	h.AddTaskWithStatus("progress-task", "In progress task", itask.StatusInProgress)

	// Filter by ready status
	err := h.Execute(Cmd, "list", "--status", "ready")
	h.AssertNoError(err)
	h.AssertOutputContains("ready-task")
	h.AssertOutputNotContains("done-task")
	h.AssertOutputNotContains("progress-task")

	// Filter by done status
	err = h.Execute(Cmd, "list", "--status", "done")
	h.AssertNoError(err)
	h.AssertOutputContains("done-task")
	h.AssertOutputNotContains("ready-task")
}

func TestListCmd_TagFilter(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("frontend-task", "Frontend work")
	h.Store().AddTag("frontend-task", "frontend")
	h.AddTask("backend-task", "Backend work")
	h.Store().AddTag("backend-task", "backend")

	// Filter by frontend tag
	err := h.Execute(Cmd, "list", "--tag", "frontend")
	h.AssertNoError(err)
	h.AssertOutputContains("frontend-task")
	h.AssertOutputNotContains("backend-task")
}

func TestListCmd_JSONFormat(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")

	err := h.ExecuteWithFormat(Cmd, "json", "list")
	h.AssertNoError(err)
	h.AssertOutputContains(`"task-1"`)
	h.AssertOutputContains(`"description": "First task"`)
}

func TestListCmd_YAMLFormat(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")

	err := h.ExecuteWithFormat(Cmd, "yaml", "list")
	h.AssertNoError(err)
	h.AssertOutputContains("task-1:")
	h.AssertOutputContains("description: First task")
}

func TestListCmd_SortOrder(t *testing.T) {
	h := testutil.New(t)

	// Add tasks in random order
	h.AddTaskWithStatus("done-task", "Done", itask.StatusDone)
	h.AddTaskWithStatus("ready-task", "Ready", itask.StatusReady)
	h.AddTaskWithStatus("progress-task", "In progress", itask.StatusInProgress)
	h.AddTaskWithStatus("blocked-task", "Blocked", itask.StatusBlocked)

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)

	// Check that output contains all tasks
	output := h.Stdout()
	h.AssertOutputContains("progress-task")
	h.AssertOutputContains("ready-task")
	h.AssertOutputContains("blocked-task")
	h.AssertOutputContains("done-task")

	// Verify in_progress comes before ready (based on position in output)
	progressPos := indexOf(output, "progress-task")
	readyPos := indexOf(output, "ready-task")
	blockedPos := indexOf(output, "blocked-task")
	donePos := indexOf(output, "done-task")

	if progressPos > readyPos {
		t.Errorf("in_progress tasks should appear before ready tasks")
	}
	if readyPos > blockedPos {
		t.Errorf("ready tasks should appear before blocked tasks")
	}
	if blockedPos > donePos {
		t.Errorf("blocked tasks should appear before done tasks")
	}
}

func TestListCmd_TreeView(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("parent-task", "Parent task")
	h.Store().AddSubtask("child-task", "Child task", "parent-task")

	err := h.Execute(Cmd, "list", "--tree")
	h.AssertNoError(err)
	h.AssertOutputContains("parent-task")
	h.AssertOutputContains("child-task")
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestListCmd_ArchivedStatus(t *testing.T) {
	h := testutil.New(t)

	// Add a task, mark it done, then archive it
	h.AddTaskWithStatus("archived-task", "Archived task", itask.StatusDone)
	h.AddTaskWithStatus("active-task", "Active task", itask.StatusReady)

	// Archive the done task
	err := h.Store().ArchiveTask("archived-task", "")
	if err != nil {
		t.Fatalf("failed to archive task: %v", err)
	}

	// List with --status archived should show only archived tasks
	err = h.Execute(Cmd, "list", "--status", "archived")
	h.AssertNoError(err)
	h.AssertOutputContains("archived-task")
	h.AssertOutputContains("⌂") // archived icon
	h.AssertOutputNotContains("active-task")

	// Regular list should not show archived tasks
	h.ResetOutput()
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("active-task")
	h.AssertOutputNotContains("archived-task")
}
