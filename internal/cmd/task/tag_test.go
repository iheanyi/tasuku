package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestTagCmd_Add(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "tag", "add", "my-task", "frontend")
	h.AssertNoError(err)
	h.AssertOutputContains("frontend")

	tsk := h.MustGetTask("my-task")
	if len(tsk.Tags) != 1 || tsk.Tags[0] != "frontend" {
		t.Errorf("expected tag 'frontend', got %v", tsk.Tags)
	}
}

func TestTagCmd_AddMultiple(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	h.Execute(Cmd, "tag", "add", "my-task", "frontend")
	h.Execute(Cmd, "tag", "add", "my-task", "urgent")

	tsk := h.MustGetTask("my-task")
	if len(tsk.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tsk.Tags))
	}
}

func TestTagCmd_Remove(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().AddTag("my-task", "frontend")
	h.Store().AddTag("my-task", "urgent")

	err := h.Execute(Cmd, "tag", "remove", "my-task", "frontend")
	h.AssertNoError(err)

	tsk := h.MustGetTask("my-task")
	if len(tsk.Tags) != 1 || tsk.Tags[0] != "urgent" {
		t.Errorf("expected only 'urgent' tag, got %v", tsk.Tags)
	}
}

func TestTagCmd_List(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().AddTag("my-task", "frontend")
	h.Store().AddTag("my-task", "urgent")

	err := h.Execute(Cmd, "tag", "list", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("frontend")
	h.AssertOutputContains("urgent")
}

func TestTagCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "tag", "add", "nonexistent", "test")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestTagCmd_MissingArgs(t *testing.T) {
	h := testutil.New(t)

	// tag without subcommand shows help (not an error)
	err := h.Execute(Cmd, "tag")
	// Just check it doesn't crash

	h.AddTask("my-task", "My task")
	// tag add without tag name should error
	err = h.Execute(Cmd, "tag", "add", "my-task")
	h.AssertError(err)
	_ = err
}
