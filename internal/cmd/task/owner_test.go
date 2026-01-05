package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestOwnerCmd_SetOwner(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "owner", "my-task", "agent-1")
	h.AssertNoError(err)
	h.AssertOutputContains("agent-1")

	tsk := h.MustGetTask("my-task")
	if tsk.Owner == nil || *tsk.Owner != "agent-1" {
		t.Error("expected owner to be agent-1")
	}
}

func TestOwnerCmd_ShowOwner(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().SetOwner("my-task", "agent-1")

	err := h.Execute(Cmd, "owner", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("agent-1")
}

func TestOwnerCmd_ShowNoOwner(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "owner", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("no owner")
}

func TestOwnerCmd_ClearOwner(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().SetOwner("my-task", "agent-1")

	err := h.Execute(Cmd, "owner", "my-task", "--clear")
	h.AssertNoError(err)

	tsk := h.MustGetTask("my-task")
	if tsk.Owner != nil {
		t.Errorf("expected owner to be cleared, got %v", tsk.Owner)
	}
}

func TestOwnerCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "owner", "nonexistent")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestOwnerCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "owner")
	h.AssertError(err)
}
