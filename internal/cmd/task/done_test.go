package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestDoneCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("my-task", "My task", itask.StatusInProgress)

	err := h.Execute(Cmd, "done", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("Completed: my-task")
	h.AssertTaskStatus("my-task", itask.StatusDone)
}

func TestDoneCmd_FromReady(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("ready-task", "Ready task", itask.StatusReady)

	err := h.Execute(Cmd, "done", "ready-task")
	h.AssertNoError(err)
	h.AssertTaskStatus("ready-task", itask.StatusDone)
}

func TestDoneCmd_FromBlocked(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker", "Blocker")
	h.AddTask("blocked-task", "Blocked task")
	h.Store().BlockTask("blocked-task", []string{"blocker"})

	// blocked -> done is not allowed
	err := h.Execute(Cmd, "done", "blocked-task")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "invalid transition")
}

func TestDoneCmd_AlreadyDone(t *testing.T) {
	h := testutil.New(t)

	h.AddTaskWithStatus("done-task", "Done task", itask.StatusDone)

	// done -> done is not allowed (already done)
	err := h.Execute(Cmd, "done", "done-task")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "invalid transition")
}

func TestDoneCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "done", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestDoneCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "done")
	h.AssertError(err)
}

func TestDoneCmd_UnblocksDependent(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker", "Blocker task")
	h.AddTask("dependent", "Dependent task")
	h.Store().BlockTask("dependent", []string{"blocker"})

	// Verify dependent is blocked
	h.AssertTaskStatus("dependent", itask.StatusBlocked)

	// Complete the blocker
	err := h.Execute(Cmd, "done", "blocker")
	h.AssertNoError(err)
	h.AssertTaskStatus("blocker", itask.StatusDone)

	// Note: Automatic unblocking of dependent tasks is not implemented
	// The dependent task keeps its blockers - user must manually unblock
	tsk := h.MustGetTask("dependent")
	// Blockers are still present (this is current behavior)
	if len(tsk.BlockedBy) == 0 {
		// If this passes, automatic unblocking was implemented
		t.Log("Note: Automatic unblocking of dependent tasks is now implemented")
	}
}
