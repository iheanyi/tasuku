package learning

import (
	"strings"
	"testing"

	"github.com/iheanyi/tasuku/internal/cmd/testutil"
)

func TestLearningListEmpty(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("No learnings recorded")
}

func TestLearningListWithLearnings(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Redis connection pooling improves latency")
	h.AddLearning("Auth middleware must run before rate limiting")

	err := h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("Redis connection pooling")
	h.AssertOutputContains("Auth middleware")
}

func TestLearningListJSON(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Test learning")

	err := h.ExecuteWithFormat(Cmd, "json", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"text"`) {
		t.Errorf("expected JSON output, got:\n%s", output)
	}
}

func TestLearningListYAML(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Test learning")

	err := h.ExecuteWithFormat(Cmd, "yaml", "list")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, "text:") {
		t.Errorf("expected YAML output, got:\n%s", output)
	}
}

func TestLearningAdd(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "New insight about the codebase")
	h.AssertNoError(err)
	h.AssertOutputContains("Learning added")

	// Verify it was added
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputContains("New insight")
}

func TestLearningAddRule(t *testing.T) {
	h := testutil.New(t)

	// Learning starting with "Never" should be detected as a rule
	err := h.Execute(Cmd, "add", "Never use raw SQL queries")
	h.AssertNoError(err)
	h.AssertOutputContains("[RULE]")

	// Verify it shows in rules list
	err = h.Execute(Cmd, "rules")
	h.AssertNoError(err)
	h.AssertOutputContains("Never use raw SQL")
}

func TestLearningAddWithRuleFlag(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "add", "--rule", "Custom rule that should be marked")
	h.AssertNoError(err)
	h.AssertOutputContains("[RULE]")
}

func TestLearningRemoveByID(t *testing.T) {
	h := testutil.New(t)

	id, _ := h.AddLearning("Learning to remove")

	err := h.Execute(Cmd, "remove", id)
	h.AssertNoError(err)
	h.AssertOutputContains("Removed learning")

	// Verify it's gone
	err = h.Execute(Cmd, "list")
	h.AssertNoError(err)
	h.AssertOutputNotContains("Learning to remove")
}

func TestLearningRemoveByText(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Unique learning about Redis")

	err := h.Execute(Cmd, "remove", "Redis")
	h.AssertNoError(err)
	h.AssertOutputContains("Removed learning")
}

func TestLearningRemoveNotFound(t *testing.T) {
	h := testutil.New(t)

	err := h.Execute(Cmd, "remove", "nonexistent-id")
	h.AssertError(err)
	if !strings.Contains(err.Error(), "no learning found") {
		t.Errorf("expected 'no learning found' error, got: %v", err)
	}
}

func TestLearningRulesEmpty(t *testing.T) {
	h := testutil.New(t)

	// Add non-rule learning
	h.AddLearning("Regular insight")

	err := h.Execute(Cmd, "rules")
	h.AssertNoError(err)
	h.AssertOutputContains("No rule learnings")
}

func TestLearningRulesWithRules(t *testing.T) {
	h := testutil.New(t)

	h.AddLearning("Regular learning")
	h.Execute(Cmd, "add", "Never commit secrets")
	h.Execute(Cmd, "add", "Always validate input")

	err := h.Execute(Cmd, "rules")
	h.AssertNoError(err)
	h.AssertOutputContains("Never commit secrets")
	h.AssertOutputContains("Always validate input")
	h.AssertOutputNotContains("Regular learning")
}

func TestLearningRulesJSON(t *testing.T) {
	h := testutil.New(t)

	h.Execute(Cmd, "add", "Never do this")

	err := h.ExecuteWithFormat(Cmd, "json", "rules")
	h.AssertNoError(err)

	output := h.Stdout()
	if !strings.Contains(output, `"is_rule"`) {
		t.Errorf("expected JSON output with is_rule, got:\n%s", output)
	}
}
