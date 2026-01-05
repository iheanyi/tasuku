package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestUnblockCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker", "Blocker task")
	h.AddTask("blocked", "Blocked task")
	h.Store().BlockTask("blocked", []string{"blocker"})

	err := h.Execute(Cmd, "unblock", "blocked")
	h.AssertNoError(err)
	h.AssertOutputContains("Unblocked")

	tsk := h.MustGetTask("blocked")
	if len(tsk.BlockedBy) > 0 {
		t.Errorf("expected no blockers, got %v", tsk.BlockedBy)
	}
	// Status should change from blocked to ready
	h.AssertTaskStatus("blocked", itask.StatusReady)
}

func TestUnblockCmd_RemoveSpecificBlocker(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker-1", "First blocker")
	h.AddTask("blocker-2", "Second blocker")
	h.AddTask("blocked", "Blocked task")
	h.Store().BlockTask("blocked", []string{"blocker-1", "blocker-2"})

	// Remove only blocker-1
	err := h.Execute(Cmd, "unblock", "blocked", "--from", "blocker-1")
	h.AssertNoError(err)

	tsk := h.MustGetTask("blocked")
	if len(tsk.BlockedBy) != 1 {
		t.Errorf("expected 1 blocker remaining, got %d", len(tsk.BlockedBy))
	}
	if tsk.BlockedBy[0] != "blocker-2" {
		t.Errorf("expected blocker-2 to remain, got %v", tsk.BlockedBy)
	}
}

func TestUnblockCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "unblock", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestUnblockCmd_NotBlocked(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "Not blocked")

	// Unblocking a non-blocked task should succeed (no-op)
	err := h.Execute(Cmd, "unblock", "my-task")
	h.AssertNoError(err)
}

func TestUnblockCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "unblock")
	h.AssertError(err)
}
