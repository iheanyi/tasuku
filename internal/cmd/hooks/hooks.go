// Package hooks provides CLI commands for managing git hooks and AI integration.
package hooks

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/task"
)

func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage git hooks and AI integration hooks",
		Long: `Manage hooks for git and AI tool integration with Tasuku.

Install/Uninstall:
  install    Install hooks (git, Claude Code, Codex, OpenCode, Copilot CLI, Cursor)
  uninstall  Remove hooks (git, Claude Code, Codex, OpenCode, Copilot CLI, Cursor)

Utility Commands:
  session        Display Tasuku context summary at session start
  stop-reminder  Remind about running timers and in-progress tasks
  codex-notify   Handle Codex notify callback
  sync           Sync tasks from TodoWrite JSON input (uses nudge rule)
  plan-sync      Extract tasks from plan files (uses nudge rule)

Git hooks provide:
  - pre-commit: Validates Tasuku storage before commits
  - post-commit: Suggests task status updates based on commit messages

Claude Code hooks provide:
  - SessionStart: Shows project context summary when session begins
  - Stop: Reminds about running timers and in-progress tasks
  - ExitPlanMode: Prompts to sync plan to Tasuku tasks

Codex hooks provide:
  - notify: Called on agent turn completion

OpenCode hooks provide (via plugin):
  - session.created: Shows context summary at session start
  - session.idle: Reminds about running timers
  - todo.updated: Checks for project-level tasks

Cursor hooks provide:
  - sessionStart: Shows project context summary when session begins
  - stop: Reminds about running timers and in-progress tasks
  - preCompact: Shows context summary before compaction
  - postToolUse: Checks if TodoWrite items should persist
  - beforeSubmitPrompt: Detects task intent in prompts

The sync/plan-sync commands apply the nudge rule: only project-level tasks
are synced, session-level implementation steps are skipped.

Run 'tk hooks <subcommand> --help' for more details.`,
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(sessionCmd)
	cmd.AddCommand(stopReminderCmd)
	cmd.AddCommand(preCompactCmd)
	cmd.AddCommand(syncCmd)
	cmd.AddCommand(newPlanSyncCmd())
	cmd.AddCommand(newTodoCheckCmd())
	cmd.AddCommand(subagentDoneCmd)
	cmd.AddCommand(newPromptCheckCmd())
	cmd.AddCommand(codexNotifyCmd)

	return cmd
}

// Cmd is the parent command for all hooks operations
var Cmd = newHooksCmd()

// featureConfig holds enabled/disabled state for each feature
type featureConfig map[string]bool

// parseDisabledFeatures parses comma-separated feature names
func parseDisabledFeatures(s string) map[string]bool {
	disabled := make(map[string]bool)
	if s == "" {
		return disabled
	}
	for _, f := range strings.Split(s, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			disabled[f] = true
		}
	}
	return disabled
}

// buildFeatureConfig creates a feature config based on quiet mode and disabled features
func buildFeatureConfig(allFeatures []string, quietFeatures map[string]bool, quiet bool, disabled map[string]bool) featureConfig {
	config := make(featureConfig)
	for _, f := range allFeatures {
		enabled := true
		if quiet {
			enabled = quietFeatures[f]
		}
		if disabled[f] {
			enabled = false
		}
		config[f] = enabled
	}
	return config
}

// taskWithID pairs a task with its ID for sorting/display
type taskWithID struct {
	id   string
	task task.Task
}

// isBugFixTask checks if a task description indicates bug fix work
func isBugFixTask(description string) bool {
	desc := strings.ToLower(description)

	bugKeywords := []string{
		"fix", "bug", "debug", "resolve", "repair",
		"patch", "hotfix", "error", "issue", "problem",
		"broken", "crash", "failing", "failed",
	}

	for _, kw := range bugKeywords {
		if strings.Contains(desc, kw) {
			return true
		}
	}
	return false
}

// getStatusIcon returns an icon for the task status
func getStatusIcon(status task.Status) string {
	switch status {
	case task.StatusReady:
		return "○"
	case task.StatusInProgress:
		return "●"
	case task.StatusBlocked:
		return "⊘"
	case task.StatusDone:
		return "✓"
	default:
		return "?"
	}
}

// extractKeywords extracts significant words from text (skip common words)
func extractKeywords(text string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "as": true,
		"into": true, "through": true, "during": true, "before": true,
		"after": true, "above": true, "below": true, "between": true,
		"and": true, "but": true, "or": true, "nor": true, "so": true,
		"yet": true, "both": true, "either": true, "neither": true,
		"not": true, "no": true, "yes": true, "this": true, "that": true,
		"these": true, "those": true, "it": true, "its": true,
	}

	words := strings.Fields(text)
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?\"'()[]{}*_~`<>")
		w = strings.ToLower(w)
		if len(w) > 2 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}

// countOverlap counts how many words from a appear in b
func countOverlap(a, b []string) int {
	bSet := make(map[string]bool)
	for _, w := range b {
		bSet[w] = true
	}

	count := 0
	for _, w := range a {
		if bSet[w] {
			count++
		}
	}
	return count
}

// escapeForShell escapes a string for safe use in shell commands
func escapeForShell(s string) string {
	// Replace double quotes with escaped double quotes
	return strings.ReplaceAll(s, "\"", "\\\"")
}

// execCommand runs a command and returns its output
func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	return string(out), err
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// generateID creates a kebab-case ID from a description
func generateID(description string) string {
	// Simple kebab-case conversion
	id := strings.ToLower(description)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove non-alphanumeric except dashes
	var result []rune
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	id = string(result)
	// Collapse multiple dashes
	for strings.Contains(id, "--") {
		id = strings.ReplaceAll(id, "--", "-")
	}
	// Trim leading/trailing dashes
	id = strings.Trim(id, "-")
	// Truncate if too long
	if len(id) > 50 {
		id = id[:50]
		id = strings.TrimRight(id, "-")
	}
	return id
}

// shouldPersistTask checks if a task description indicates a project-level task
// that should be persisted to tk. Returns false for session-level implementation steps.
func shouldPersistTask(description string) bool {
	desc := strings.ToLower(description)

	// Keywords that indicate project-level tasks (should persist)
	projectKeywords := []string{
		"implement", "add feature", "build", "create", "develop",
		"fix bug", "bugfix", "hotfix", "patch",
		"refactor", "rewrite", "redesign", "rearchitect",
		"migrate", "upgrade", "update dependency",
		"integrate", "connect", "setup", "configure",
		"support", "enable", "add support",
		"milestone", "epic", "feature", "story",
		"api endpoint", "database", "schema",
		"authentication", "authorization", "security",
		"performance", "optimize", "cache",
		"deploy", "release", "ship",
	}

	// Keywords that indicate session-level tasks (should NOT persist)
	sessionKeywords := []string{
		"fix type error", "fix typo", "fix lint",
		"update file", "edit file", "modify file",
		"read file", "check file", "review file",
		"run test", "run build", "run script",
		"verify", "check", "confirm", "ensure",
		"debug", "investigate", "look into",
		"format", "cleanup", "tidy",
		"add comment", "add docstring", "add import",
		"remove unused", "delete unused",
		"rename variable", "rename function",
	}

	shouldPersist := false

	// Check for project keywords
	for _, kw := range projectKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = true
			break
		}
	}

	// Session keywords override
	for _, kw := range sessionKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = false
			break
		}
	}

	return shouldPersist
}
