package decision

import (
	"strings"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestDecisionListEmpty(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("No decisions recorded")
}

func TestDecisionListWithDecisions(t *testing.T) {
	h := testutil.New(t)

	h.AddDecision("db-choice", "PostgreSQL", []string{"MySQL", "SQLite"}, "Better JSON support")
	h.AddDecision("framework", "Cobra", []string{"urfave/cli"}, "Industry standard")

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("db-choice")
	h.AssertOutputContains("PostgreSQL")
	h.AssertOutputContains("framework")
	h.AssertOutputContains("Cobra")
}

func TestDecisionListJSON(t *testing.T) {
	h := testutil.New(t)

	h.AddDecision("test-decision", "Option A", []string{"Option B"}, "Test reason")

	err := h.ExecuteWithFormat(Cmd, "json", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"id"`) {
		t.Errorf("expected JSON output, got:\n%s", output)
	}
	if !strings.Contains(output, `"test-decision"`) {
		t.Errorf("expected decision ID in output, got:\n%s", output)
	}
}

func TestDecisionListYAML(t *testing.T) {
	h := testutil.New(t)

	h.AddDecision("test-decision", "Option A", []string{"Option B"}, "Test reason")

	err := h.ExecuteWithFormat(Cmd, "yaml", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, "id:") {
		t.Errorf("expected YAML output, got:\n%s", output)
	}
}

func TestDecisionAdd(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add",
		"--id", "auth-method",
		"--chose", "JWT",
		"--over", "sessions,OAuth",
		"--because", "Stateless and scalable")
	h.AssertNoError(err)
	h.AssertOutputContains("Decision recorded: auth-method")

	// Verify it was added
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("auth-method")
	h.AssertOutputContains("JWT")
}

func TestDecisionAddNoOver(t *testing.T) {
	h := testutil.New(t)

	// Should work without --over
	err := h.Execute(Cmd, "add",
		"--id", "simple-decision",
		"--chose", "Simple option",
		"--because", "It was obvious")
	h.AssertNoError(err)
	h.AssertOutputContains("Decision recorded")
}

func TestDecisionAddMissingRequired(t *testing.T) {
	h := testutil.New(t)

	// Missing --chose
	err := h.Execute(Cmd, "add", "--id", "test", "--because", "reason")
	h.AssertError(err)

	// Missing --because
	err = h.Execute(Cmd, "add", "--id", "test", "--chose", "option")
	h.AssertError(err)

	// Missing --id
	err = h.Execute(Cmd, "add", "--chose", "option", "--because", "reason")
	h.AssertError(err)
}

func TestDecisionRemove(t *testing.T) {
	h := testutil.New(t)

	h.AddDecision("to-remove", "Option X", []string{"Option Y"}, "Will be removed")

	err := h.Execute(Cmd, "remove", "to-remove")
	h.AssertNoError(err)
	h.AssertOutputContains("Removed decision")

	// Verify it's gone
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputNotContains("to-remove")
}

func TestDecisionRemoveNotFound(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "remove", "nonexistent")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}
