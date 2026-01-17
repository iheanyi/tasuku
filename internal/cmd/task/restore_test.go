package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestRestoreCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	// Add a task, mark done, archive it
	h.AddTaskWithStatus("archived-task", "Archived task", itask.StatusDone)
	err := h.Store().ArchiveTask("archived-task", "")
	if err != nil {
		t.Fatalf("failed to archive task: %v", err)
	}

	// Restore it
	err = h.Execute(Cmd, "restore", "archived-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Restored task archived-task")
	h.AssertOutputContains("status: ready")

	// Verify task is back in active tasks
	if !h.TaskExists("archived-task") {
		t.Error("expected task to be restored to active tasks")
	}

	// Verify task has ready status
	h.AssertTaskStatus("archived-task", itask.StatusReady)

	// Verify task is no longer in archive
	archived, _ := h.Store().GetArchivedTasks()
	if _, exists := archived["archived-task"]; exists {
		t.Error("expected task to be removed from archive")
	}
}

func TestRestoreCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "restore", "nonexistent")
	h.AssertError(err)
}

func TestRestoreCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "restore")
	h.AssertError(err)
}

func TestRestoreCmd_ActiveTaskNotArchived(t *testing.T) {
	h := testutil.New(t)

	// Add an active (non-archived) task
	h.AddTaskWithStatus("active-task", "Active task", itask.StatusReady)

	// Try to restore it (should fail because it's not archived)
	err := h.Execute(Cmd, "restore", "active-task")
	h.AssertError(err)
}
