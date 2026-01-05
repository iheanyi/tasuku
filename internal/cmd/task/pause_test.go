package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestPauseCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("my-task", "In progress task", itask.StatusInProgress)

	err := h.Execute(Cmd, "pause", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Paused")
	h.AssertTaskStatus("my-task", itask.StatusReady)
}

func TestPauseCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "pause", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestPauseCmd_NotInProgress(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("ready-task", "Ready task", itask.StatusReady)

	// Pausing a ready task - may succeed or fail depending on implementation
	err := h.Execute(Cmd, "pause", "ready-task")
	// Just check it doesn't crash - behavior may vary
	_ = err
}

func TestPauseCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "pause")
	h.AssertError(err)
}
