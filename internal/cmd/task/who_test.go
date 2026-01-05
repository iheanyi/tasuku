package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestWhoCmd_Empty(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "who")
	h.AssertNoError(err)
	h.AssertOutputContains("No tasks are currently claimed")
}

func TestWhoCmd_WithClaims(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddTask("task-2", "Second task")
	h.Store().ClaimTask("task-1", "agent-1")
	h.Store().ClaimTask("task-2", "agent-2")

	err := h.Execute(Cmd, "who")
	h.AssertNoError(err)
	h.AssertOutputContains("agent-1")
	h.AssertOutputContains("agent-2")
	h.AssertOutputContains("task-1")
	h.AssertOutputContains("task-2")
}

func TestWhoCmd_FilterByAgent(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.AddTask("task-2", "Second task")
	h.Store().ClaimTask("task-1", "agent-1")
	h.Store().ClaimTask("task-2", "agent-2")

	err := h.Execute(Cmd, "who", "agent-1")
	h.AssertNoError(err)
	h.AssertOutputContains("agent-1")
	h.AssertOutputContains("task-1")
	h.AssertOutputNotContains("agent-2")
	h.AssertOutputNotContains("task-2")
}

func TestWhoCmd_NoTasksForAgent(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.Store().ClaimTask("task-1", "agent-1")

	err := h.Execute(Cmd, "who", "agent-2")
	h.AssertNoError(err)
	h.AssertOutputContains("No tasks claimed by agent-2")
}

func TestWhoCmd_JSONFormat(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("task-1", "First task")
	h.Store().ClaimTask("task-1", "agent-1")

	err := h.ExecuteWithFormat(Cmd, "json", "who")
	h.AssertNoError(err)
	h.AssertOutputContains(`"owner"`)
	h.AssertOutputContains(`"task_id"`)
}
