package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestDeleteCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "Task to delete")

	err := h.Execute(Cmd, "delete", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Deleted")

	if h.TaskExists("my-task") {
		t.Error("expected task to be deleted")
	}
}

func TestDeleteCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "delete", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestDeleteCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "delete")
	h.AssertError(err)
}

func TestDeleteCmd_MultipleDeletes(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddTask("task-2", "Second task")
	h.AddTask("task-3", "Third task")

	// Delete one by one
	h.Execute(Cmd, "delete", "task-1")
	h.Execute(Cmd, "delete", "task-3")

	if h.TaskExists("task-1") || h.TaskExists("task-3") {
		t.Error("expected tasks to be deleted")
	}
	if !h.TaskExists("task-2") {
		t.Error("expected task-2 to still exist")
	}
}
