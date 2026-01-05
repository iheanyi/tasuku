package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestClaimCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "claim", "my-task", "agent-1")
	h.AssertNoError(err)
	h.AssertOutputContains("claimed")
	h.AssertOutputContains("agent-1")

	tsk := h.MustGetTask("my-task")
	if tsk.Owner == nil || *tsk.Owner != "agent-1" {
		t.Error("expected task to be owned by agent-1")
	}
	h.AssertTaskStatus("my-task", itask.StatusInProgress)
}

func TestClaimCmd_AutoDetectAgent(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	// Without specifying agent name, should auto-detect
	err := h.Execute(Cmd, "claim", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("claimed")

	tsk := h.MustGetTask("my-task")
	if tsk.Owner == nil {
		t.Error("expected task to have an owner")
	}
}

func TestClaimCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "claim", "nonexistent", "agent-1")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestClaimCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "claim")
	h.AssertError(err)
}
