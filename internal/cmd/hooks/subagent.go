package hooks

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var subagentDoneCmd = &cobra.Command{
	Use:   "subagent-done",
	Short: "Capture insights after subagent (Task tool) completes",
	Long: `Called by Claude Code SubagentStop hook when a Task tool subagent completes.

Subagents often do deep exploration work (code searches, complex analysis,
multi-step implementations). When they complete, prompt for:
  - Learnings discovered during exploration
  - Decisions made about implementation approaches
  - Patterns or gotchas worth documenting

This helps capture valuable insights that might otherwise be lost
when subagent context is merged back.

Examples:
  tk hooks subagent-done   # Prompt for subagent insights`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookSubagentDone()
	},
}

type subagentStopInput struct {
	SubagentType         string `json:"subagent_type"`
	LastAssistantMessage string `json:"last_assistant_message"`
}

// hookSubagentDone prompts for insights after subagent completion
func hookSubagentDone() error {
	var input subagentStopInput
	// SubagentStop delivers JSON on stdin like all other hook events.
	// Ignore decode errors — if stdin is empty or malformed, proceed with zero values.
	_ = json.NewDecoder(os.Stdin).Decode(&input)

	agentType := input.SubagentType

	// Only prompt for exploration-type subagents that do significant work.
	// Skip trivial agents like quick "haiku" lookups.
	significantAgents := map[string]bool{
		"Explore":          true,
		"general-purpose":  true,
		"Plan":             true,
		"code-reviewer":    true,
		"database-design":  true,
		"issue-summarizer": true,
	}

	if agentType != "" && !significantAgents[agentType] {
		// Not a significant agent type, skip
		return nil
	}

	// Check if there's ongoing Tasuku work
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return nil
	}
	if !s.Exists() {
		return nil
	}

	f, err := s.Read()
	if err != nil {
		return nil
	}

	// Only prompt if there are in-progress tasks (active work session)
	hasInProgress := false
	for _, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			hasInProgress = true
			break
		}
	}

	if !hasInProgress {
		return nil
	}

	// Output insight prompt
	fmt.Println("🔍 Subagent exploration completed.")
	fmt.Println()
	fmt.Println("Did the exploration reveal:")
	fmt.Println("   - Patterns or conventions? → /tasuku:learn")
	fmt.Println("   - Gotchas or unexpected behaviors? → /tasuku:learn")
	fmt.Println("   - Design decisions to document? → /tasuku:decide")
	fmt.Println()
	fmt.Println("   Or use /tasuku:reflect for guided extraction.")
	fmt.Println()

	return nil
}
