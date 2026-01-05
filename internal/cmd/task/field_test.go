package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestFieldCmd_Set(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "field", "set", "my-task", "assignee", "john")
	h.AssertNoError(err)
	h.AssertOutputContains("assignee")
	h.AssertOutputContains("john")

	tsk := h.MustGetTask("my-task")
	if tsk.Fields["assignee"] != "john" {
		t.Errorf("expected field assignee=john, got %v", tsk.Fields)
	}
}

func TestFieldCmd_SetMultiple(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	h.Execute(Cmd, "field", "set", "my-task", "assignee", "john")
	h.Execute(Cmd, "field", "set", "my-task", "team", "backend")

	tsk := h.MustGetTask("my-task")
	if len(tsk.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(tsk.Fields))
	}
	if tsk.Fields["assignee"] != "john" || tsk.Fields["team"] != "backend" {
		t.Errorf("unexpected fields: %v", tsk.Fields)
	}
}

func TestFieldCmd_Remove(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().SetField("my-task", "assignee", "john")
	h.Store().SetField("my-task", "team", "backend")

	err := h.Execute(Cmd, "field", "remove", "my-task", "assignee")
	h.AssertNoError(err)

	tsk := h.MustGetTask("my-task")
	if _, ok := tsk.Fields["assignee"]; ok {
		t.Error("expected assignee field to be removed")
	}
	if tsk.Fields["team"] != "backend" {
		t.Error("expected team field to remain")
	}
}

func TestFieldCmd_List(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")
	h.Store().SetField("my-task", "assignee", "john")
	h.Store().SetField("my-task", "team", "backend")

	err := h.Execute(Cmd, "field", "list", "my-task")
	h.AssertNoError(err)
	h.AssertOutputContains("assignee")
	h.AssertOutputContains("john")
	h.AssertOutputContains("team")
	h.AssertOutputContains("backend")
}

func TestFieldCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "field", "set", "nonexistent", "key", "value")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestFieldCmd_MissingArgs(t *testing.T) {
	h := testutil.New(t)

	// field without subcommand shows help (not an error)
	err := h.Execute(Cmd, "field")
	// Just check it doesn't crash

	h.AddTask("my-task", "My task")
	// field set without value should error
	err = h.Execute(Cmd, "field", "set", "my-task", "key")
	h.AssertError(err)
	_ = err
}
