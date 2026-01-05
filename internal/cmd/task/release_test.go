package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestReleaseCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().ClaimTask("my-task", "agent-1")
	h.Store().SetStatus("my-task", itask.StatusInProgress)

	err := h.Execute(Cmd, "release", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("released")

	tsk := h.MustGetTask("my-task")
	if tsk.Owner != nil {
		t.Errorf("expected owner to be cleared, got %v", tsk.Owner)
	}
}

func TestReleaseCmd_SetsToReady(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().ClaimTask("my-task", "agent-1")
	h.Store().SetStatus("my-task", itask.StatusInProgress)

	err := h.Execute(Cmd, "release", "my-task")
	h.AssertNoError(err)

	// Task should be back to ready status
	h.AssertTaskStatus("my-task", itask.StatusReady)
}

func TestReleaseCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "release", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestReleaseCmd_NotClaimed(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	// Releasing an unclaimed task should succeed (no-op)
	err := h.Execute(Cmd, "release", "my-task")
	h.AssertNoError(err)
}

func TestReleaseCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "release")
	h.AssertError(err)
}
