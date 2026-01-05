package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestBlockCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker", "Blocker task")
	h.AddTask("blocked", "Task to block")

	err := h.Execute(Cmd, "block", "blocked", "--by=blocker")
	h.AssertNoError(err)
	h.AssertOutputContains("blocked")
	h.AssertTaskStatus("blocked", itask.StatusBlocked)

	tsk := h.MustGetTask("blocked")
	if len(tsk.BlockedBy) != 1 || tsk.BlockedBy[0] != "blocker" {
		t.Errorf("expected blocker to be 'blocker', got %v", tsk.BlockedBy)
	}
}

func TestBlockCmd_MultipleBlockers(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker-1", "First blocker")
	h.AddTask("blocker-2", "Second blocker")
	h.AddTask("blocked", "Task to block")

	err := h.Execute(Cmd, "block", "blocked", "--by=blocker-1,blocker-2")
	h.AssertNoError(err)
	h.AssertTaskStatus("blocked", itask.StatusBlocked)

	tsk := h.MustGetTask("blocked")
	if len(tsk.BlockedBy) != 2 {
		t.Errorf("expected 2 blockers, got %d", len(tsk.BlockedBy))
	}
}

func TestBlockCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("blocker", "Blocker")

	err := h.Execute(Cmd, "block", "nonexistent", "--by", "blocker")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestBlockCmd_MissingByFlag(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "block", "my-task")
	h.AssertError(err)
}

func TestBlockCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "block")
	h.AssertError(err)
}
