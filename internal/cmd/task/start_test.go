package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestStartCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "start", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Started: my-task")
	h.AssertTaskStatus("my-task", itask.StatusInProgress)
}

func TestStartCmd_FromReady(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("ready-task", "Ready task", itask.StatusReady)

	err := h.Execute(Cmd, "start", "ready-task")
	h.AssertNoError(err)
	h.AssertTaskStatus("ready-task", itask.StatusInProgress)
}

func TestStartCmd_FromDone(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("done-task", "Done task", itask.StatusDone)

	// done -> in_progress is not allowed (need to reopen first)
	err := h.Execute(Cmd, "start", "done-task")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "invalid transition")
}

func TestStartCmd_AlreadyInProgress(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("progress-task", "Progress task", itask.StatusInProgress)

	// in_progress -> in_progress is not allowed (already in progress)
	err := h.Execute(Cmd, "start", "progress-task")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "invalid transition")
}

func TestStartCmd_BlockedTask(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker", "Blocker task")
	h.AddTask("blocked-task", "Blocked task")
	h.Store().BlockTask("blocked-task", []string{"blocker"})

	// Starting a blocked task without --unblock should fail
	err := h.Execute(Cmd, "start", "blocked-task")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "invalid transition")
}

func TestStartCmd_WithUnblock(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker", "Blocker task")
	h.AddTask("blocked-task", "Blocked task")
	h.Store().BlockTask("blocked-task", []string{"blocker"})

	// Starting with --unblock should clear blockers
	err := h.Execute(Cmd, "start", "blocked-task", "--unblock")
	h.AssertNoError(err)
	h.AssertTaskStatus("blocked-task", itask.StatusInProgress)

	// Blockers should be cleared
	tsk := h.MustGetTask("blocked-task")
	if len(tsk.BlockedBy) > 0 {
		t.Errorf("expected blockers to be cleared, got %v", tsk.BlockedBy)
	}
}

func TestStartCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "start", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestStartCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "start")
	h.AssertError(err)
}
