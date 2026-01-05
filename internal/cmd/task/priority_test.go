package task

import (
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
	itask "github.com/iheanyi/tasuku/internal/task"
)

func TestPriorityCmd_SetCritical(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "priority", "my-task", "0")
	h.AssertNoError(err)
	h.AssertOutputContains("critical")

	tsk := h.MustGetTask("my-task")
	if tsk.GetPriority() != itask.PriorityCritical {
		t.Errorf("expected critical priority, got %d", tsk.GetPriority())
	}
}

func TestPriorityCmd_SetByName(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "priority", "my-task", "high")
	h.AssertNoError(err)

	tsk := h.MustGetTask("my-task")
	if tsk.GetPriority() != itask.PriorityHigh {
		t.Errorf("expected high priority, got %d", tsk.GetPriority())
	}
}

func TestPriorityCmd_AllLevels(t *testing.T) {
	h := testutil.New(t)

	tests := []struct {
		name     string
		level    string
		expected int
	}{
		{"critical-num", "0", itask.PriorityCritical},
		{"critical-name", "critical", itask.PriorityCritical},
		{"high-num", "1", itask.PriorityHigh},
		{"high-name", "high", itask.PriorityHigh},
		{"normal-num", "2", itask.PriorityNormal},
		{"normal-name", "normal", itask.PriorityNormal},
		{"low-num", "3", itask.PriorityLow},
		{"low-name", "low", itask.PriorityLow},
		{"backlog-num", "4", itask.PriorityBacklog},
		{"backlog-name", "backlog", itask.PriorityBacklog},
	}

	for _, tc := range tests {
		h.AddTask(tc.name, "Test task")
		err := h.Execute(Cmd, "priority", tc.name, tc.level)
		h.AssertNoError(err)

		tsk := h.MustGetTask(tc.name)
		if tsk.GetPriority() != tc.expected {
			t.Errorf("%s: expected priority %d, got %d", tc.name, tc.expected, tsk.GetPriority())
		}
	}
}

func TestPriorityCmd_InvalidPriority(t *testing.T) {
	h := testutil.New(t)

	h.AddTask("my-task", "My task")

	err := h.Execute(Cmd, "priority", "my-task", "invalid")
	h.AssertError(err)
}

func TestPriorityCmd_NonexistentTask(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "priority", "nonexistent", "high")
	h.AssertError(err)
	h.AssertErrorContainsMsg(err, "not found")
}

func TestPriorityCmd_MissingArgs(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "priority")
	h.AssertError(err)

	h.AddTask("my-task", "My task")
	err = h.Execute(Cmd, "priority", "my-task")
	h.AssertError(err)
}
