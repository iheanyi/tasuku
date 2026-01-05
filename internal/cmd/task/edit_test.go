package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestEditCmd_Basic(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "Original description")

	err := h.Execute(Cmd, "edit", "my-task", "Updated description")
	h.AssertNoError(err)
	h.AssertOutputContains("Updated")

	h.AssertTaskDescription("my-task", "Updated description")
}

func TestEditCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "edit", "nonexistent", "New description")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestEditCmd_MissingDescription(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "Original")

	err := h.Execute(Cmd, "edit", "my-task")
	h.AssertError(err)
}

func TestEditCmd_MissingArg(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "edit")
	h.AssertError(err)
}

func TestEditCmd_PreservesOtherFields(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "Original description")
	h.Store().AddTag("my-task", "important")
	h.Store().SetOwner("my-task", "agent-1")

	err := h.Execute(Cmd, "edit", "my-task", "New description")
	h.AssertNoError(err)

	tsk := h.MustGetTask("my-task")
	if tsk.Description != "New description" {
		t.Errorf("expected description to be updated")
	}
	if len(tsk.Tags) == 0 || tsk.Tags[0] != "important" {
		t.Error("expected tags to be preserved")
	}
	if tsk.Owner == nil || *tsk.Owner != "agent-1" {
		t.Error("expected owner to be preserved")
	}
}
