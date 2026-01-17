package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestArchiveCmd_SingleTask(t *testing.T) {
	h := testutil.New(t)

	// Add a done task
	h.AddTaskWithStatus("done-task", "Completed task", itask.StatusDone)

	// Archive it
	err := h.Execute(Cmd, "archive", "done-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Archived task done-task")

	// Verify task is archived
	archived, err := h.Store().GetArchivedTasks()
	if err != nil {
		t.Fatalf("failed to get archived tasks: %v", err)
	}
	if _, exists := archived["done-task"]; !exists {
		t.Error("expected task to be in archive")
	}

	// Verify task is no longer in active tasks
	if h.TaskExists("done-task") {
		t.Error("expected task to be removed from active tasks")
	}
}

func TestArchiveCmd_WithSummary(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("done-task", "Completed task", itask.StatusDone)

	err := h.Execute(Cmd, "archive", "done-task", "--summary", "Implemented feature X")
	h.AssertNoError(err)
	h.AssertOutputContains("Archived task done-task with summary: Implemented feature X")

	// Verify summary is stored
	archived, _ := h.Store().GetArchivedTask("done-task")
	if archived.Summary != "Implemented feature X" {
		t.Errorf("expected summary 'Implemented feature X', got '%s'", archived.Summary)
	}
}

func TestArchiveCmd_NotDoneTask(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("ready-task", "Ready task", itask.StatusReady)

	err := h.Execute(Cmd, "archive", "ready-task")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "done")
}

func TestArchiveCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "archive", "nonexistent")
	h.AssertError(err)
}

func TestArchiveCmd_BulkOlderThan(t *testing.T) {
	h := testutil.New(t)

	// Add done tasks
	h.AddTaskWithStatus("old-task-1", "Old task 1", itask.StatusDone)
	h.AddTaskWithStatus("old-task-2", "Old task 2", itask.StatusDone)
	h.AddTaskWithStatus("ready-task", "Ready task", itask.StatusReady)

	// Archive all done tasks (using 0 seconds to archive everything)
	err := h.Execute(Cmd, "archive", "--older-than", "0h")
	h.AssertNoError(err)
	h.AssertOutputContains("Archived 2 tasks")

	// Verify tasks are archived
	archived, _ := h.Store().GetArchivedTasks()
	if len(archived) != 2 {
		t.Errorf("expected 2 archived tasks, got %d", len(archived))
	}

	// Verify ready task is still active
	if !h.TaskExists("ready-task") {
		t.Error("ready task should not be archived")
	}
}

func TestArchiveCmd_BulkNothingToArchive(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("ready-task", "Ready task", itask.StatusReady)

	err := h.Execute(Cmd, "archive", "--older-than", "7d")
	h.AssertNoError(err)
	h.AssertOutputContains("No done tasks older than 7d to archive")
}

func TestArchiveCmd_ConflictingArgs(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("done-task", "Done task", itask.StatusDone)

	// Can't use both task ID and --older-than
	err := h.Execute(Cmd, "archive", "done-task", "--older-than", "7d")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "cannot use both")
}

func TestArchiveCmd_MissingArgs(t *testing.T) {
	h := testutil.New(t)

	// Must provide either task ID or --older-than
	err := h.Execute(Cmd, "archive")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "provide a task ID")
}
