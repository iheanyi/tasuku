// Package hooks provides CLI commands for managing git hooks and AI integration.
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage git hooks and AI integration hooks",
		Long: `Manage hooks for git and AI tool integration with Tasuku.

Install/Uninstall:
  install    Install hooks (git and/or Claude Code)
  uninstall  Remove hooks (git and/or Claude Code)

Utility Commands:
  session       Display Tasuku context summary at session start
  stop-reminder Remind about running timers and in-progress tasks
  sync          Sync tasks from TodoWrite JSON input (uses nudge rule)
  plan-sync     Extract tasks from plan files (uses nudge rule)

Git hooks provide:
  - pre-commit: Validates Tasuku storage before commits
  - post-commit: Suggests task status updates based on commit messages

Claude Code hooks provide:
  - SessionStart: Shows project context summary when session begins
  - Stop: Reminds about running timers and in-progress tasks
  - ExitPlanMode: Prompts to sync plan to Tasuku tasks

The sync/plan-sync commands apply the nudge rule: only project-level tasks
are synced, session-level implementation steps are skipped.

Run 'tk hooks <subcommand> --help' for more details.`,
	}

	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(sessionCmd)
	cmd.AddCommand(stopReminderCmd)
	cmd.AddCommand(syncCmd)
	cmd.AddCommand(planSyncCmd)

	return cmd
}

// Cmd is the parent command for all hooks operations
var Cmd = newHooksCmd()

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Tasuku hooks",
		Long: `Install Tasuku hooks for git and/or Claude Code.

By default, installs all hooks. Use flags to install specific hooks only.

Git hooks (always local to .git/hooks/):
  - pre-commit: Validates Tasuku storage before commits
  - post-commit: Suggests task status updates based on commit messages

Claude Code hooks (global ~/.claude/ by default, or local ./.claude/ with --local):
  - SessionStart: Shows project context summary when session begins
  - Stop: Reminds about running timers and in-progress tasks
  - ExitPlanMode: Prompts to sync plan to Tasuku tasks

Examples:
  tk hooks install           # Git (local) + Claude (global)
  tk hooks install --local   # Git (local) + Claude (local to project)
  tk hooks install --git     # Install only git hooks
  tk hooks install --claude  # Install only Claude Code hooks (global)
  tk hooks install --claude --local  # Claude hooks local to project`,
		RunE: runInstall,
	}

	cmd.Flags().Bool("git", false, "Install git hooks only")
	cmd.Flags().Bool("claude", false, "Install Claude Code hooks only")
	cmd.Flags().Bool("force", false, "Overwrite existing hooks")
	cmd.Flags().Bool("local", false, "Install Claude hooks to project .claude/ instead of global")

	return cmd
}

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Tasuku hooks",
		Long: `Remove Tasuku hooks from git and/or Claude Code.

By default, removes all hooks. Use flags to remove specific hooks only.

Examples:
  tk hooks uninstall                  # Remove all hooks
  tk hooks uninstall --git            # Remove only git hooks
  tk hooks uninstall --claude         # Remove Claude Code hooks (global)
  tk hooks uninstall --claude --local # Remove Claude Code hooks (project)`,
		RunE: runUninstall,
	}

	cmd.Flags().Bool("git", false, "Remove git hooks only")
	cmd.Flags().Bool("claude", false, "Remove Claude Code hooks only")
	cmd.Flags().Bool("local", false, "Remove Claude hooks from project .claude/ instead of global")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	gitOnly, _ := cmd.Flags().GetBool("git")
	claudeOnly, _ := cmd.Flags().GetBool("claude")
	force, _ := cmd.Flags().GetBool("force")
	local, _ := cmd.Flags().GetBool("local")

	// If neither specified, install all
	installGit := !claudeOnly || gitOnly
	installClaude := !gitOnly || claudeOnly

	// If both flags given, install both
	if gitOnly && claudeOnly {
		installGit = true
		installClaude = true
	}

	var errors []string

	if installGit {
		if err := installGitHooks(); err != nil {
			errors = append(errors, fmt.Sprintf("git: %v", err))
		}
	}

	if installClaude {
		if err := installClaudeHooks(force, local); err != nil {
			errors = append(errors, fmt.Sprintf("claude: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some hooks failed to install:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	gitOnly, _ := cmd.Flags().GetBool("git")
	claudeOnly, _ := cmd.Flags().GetBool("claude")
	local, _ := cmd.Flags().GetBool("local")

	// If neither specified, uninstall all
	uninstallGit := !claudeOnly || gitOnly
	uninstallClaude := !gitOnly || claudeOnly

	// If both flags given, uninstall both
	if gitOnly && claudeOnly {
		uninstallGit = true
		uninstallClaude = true
	}

	var errors []string

	if uninstallGit {
		if err := uninstallGitHooks(); err != nil {
			errors = append(errors, fmt.Sprintf("git: %v", err))
		}
	}

	if uninstallClaude {
		if err := uninstallClaudeHooks(local); err != nil {
			errors = append(errors, fmt.Sprintf("claude: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some hooks failed to uninstall:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Display Tasuku context summary",
	Long: `Display a summary of Tasuku context for Claude Code session start.

Shows:
  - Task counts by status
  - Number of learnings and decisions
  - Suggested next task based on priority

Examples:
  tk hooks session               # Display context summary`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookSession()
	},
}

var stopReminderCmd = &cobra.Command{
	Use:   "stop-reminder",
	Short: "Remind about running timers and in-progress tasks",
	Long: `Display reminders about running timers and in-progress tasks when session ends.

This is called by the Claude Code Stop hook to remind the agent about:
  - Running timers that should be stopped
  - Tasks marked as in_progress that may need status updates
  - Any uncommitted learnings from the session

Examples:
  tk hooks stop-reminder   # Check for reminders`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookStopReminder()
	},
}

func hookStopReminder() error {
	s := store.DefaultStorageWithWarning()
	if !s.Exists() {
		return nil
	}

	f, err := s.Read()
	if err != nil {
		return err
	}

	hasReminders := false

	// Check for running timers
	var runningTimers []string
	for id, t := range f.Tasks {
		if t.IsTimerRunning() {
			runningTimers = append(runningTimers, id)
		}
	}
	if len(runningTimers) > 0 {
		if !hasReminders {
			fmt.Println("=== Tasuku Session Reminder ===")
			hasReminders = true
		}
		fmt.Printf("\n⏱️  Running timers (%d):\n", len(runningTimers))
		for _, id := range runningTimers {
			t := f.Tasks[id]
			elapsed := t.CurrentDuration()
			fmt.Printf("   - %s (running for %s)\n", id, formatDuration(elapsed))
		}
		fmt.Println("   Consider: tk timer stop <id>")
	}

	// Check for in-progress tasks
	var inProgressTasks []string
	for id, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			inProgressTasks = append(inProgressTasks, id)
		}
	}
	if len(inProgressTasks) > 0 {
		if !hasReminders {
			fmt.Println("=== Tasuku Session Reminder ===")
			hasReminders = true
		}
		fmt.Printf("\n🔄 Tasks still in progress (%d):\n", len(inProgressTasks))
		for _, id := range inProgressTasks {
			t := f.Tasks[id]
			desc := t.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Printf("   - %s: %s\n", id, desc)
		}
		fmt.Println("   Consider: tk task done <id> or tk task pause <id>")
	}

	if hasReminders {
		fmt.Println("\n================================")
	}

	return nil
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

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync tasks from TodoWrite JSON",
	Long: `Sync tasks from Claude Code's TodoWrite tool.

Reads JSON from stdin in TodoWrite format and applies the nudge rule:
- Project-level tasks (features, bugs, refactors) are synced to Tasuku
- Session-level tasks (fix type error, update file) stay in TodoWrite only

This prevents cluttering your task list with temporary implementation steps.

Examples:
  tk hooks sync < todos.json         # Sync from file
  echo '[...]' | tk hooks sync       # Sync from piped JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookSync()
	},
}

func hookSession() error {
	s := store.DefaultStorageWithWarning()
	if !s.Exists() {
		return nil
	}

	f, err := s.Read()
	if err != nil {
		return err
	}

	counts := map[task.Status]int{}
	for _, t := range f.Tasks {
		counts[t.Status]++
	}

	readyCount := 0
	var highestPriority *string
	highestPriorityVal := 999

	for id, t := range f.Tasks {
		if t.Status == task.StatusReady {
			blocked := false
			for _, blockerID := range t.BlockedBy {
				if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
					blocked = true
					break
				}
			}
			if !blocked {
				readyCount++
				if t.GetPriority() < highestPriorityVal {
					highestPriorityVal = t.GetPriority()
					idCopy := id
					highestPriority = &idCopy
				}
			}
		}
	}

	fmt.Println("=== Tasuku Context ===")
	fmt.Printf("Tasks: %d ready, %d in_progress, %d blocked, %d done\n",
		readyCount, counts[task.StatusInProgress], counts[task.StatusBlocked], counts[task.StatusDone])

	if len(f.Context.Learnings) > 0 {
		fmt.Printf("Learnings: %d recorded\n", len(f.Context.Learnings))
	}
	if len(f.Context.Decisions) > 0 {
		fmt.Printf("Decisions: %d recorded\n", len(f.Context.Decisions))
	}

	// Surface rules at session start (limit to avoid noise)
	var rules []task.Learning
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			rules = append(rules, l)
		}
	}
	const maxRulesToShow = 7
	if len(rules) > 0 && len(rules) <= maxRulesToShow {
		fmt.Printf("\nActive rules (%d):\n", len(rules))
		for _, r := range rules {
			text := r.Text
			if len(text) > 70 {
				text = text[:67] + "..."
			}
			fmt.Printf("  - %s\n", text)
		}
	} else if len(rules) > maxRulesToShow {
		fmt.Printf("\nActive rules: %d (run 'tk learning rules' to see all)\n", len(rules))
	}

	if highestPriority != nil {
		t := f.Tasks[*highestPriority]
		fmt.Printf("\nNext task: %s\n  %s\n", *highestPriority, t.Description)
	}

	fmt.Println("======================")
	return nil
}

func hookSync() error {
	s := store.DefaultStorageWithWarning()
	if !s.Exists() {
		return fmt.Errorf("no Tasuku storage found - run 'tk init' first")
	}

	var todos []struct {
		Content    string `json:"content"`
		Status     string `json:"status"`
		ActiveForm string `json:"activeForm"`
	}

	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&todos); err != nil {
		return fmt.Errorf("failed to parse TodoWrite JSON: %w", err)
	}

	if len(todos) == 0 {
		return nil
	}

	f, err := s.Read()
	if err != nil {
		return err
	}

	synced := 0
	skipped := 0
	for _, todo := range todos {
		id := generateID(todo.Content)
		if id == "" {
			continue
		}

		// Apply nudge rule: only sync project-level tasks
		// Session-level tasks stay in TodoWrite only
		if !shouldPersistTask(todo.Content) {
			// Task already exists? Update status. New task? Skip it.
			if _, exists := f.Tasks[id]; !exists {
				skipped++
				continue
			}
		}

		var status task.Status
		switch todo.Status {
		case "pending":
			status = task.StatusReady
		case "in_progress":
			status = task.StatusInProgress
		case "completed":
			status = task.StatusDone
		default:
			status = task.StatusReady
		}

		if existing, exists := f.Tasks[id]; exists {
			if existing.Status != status {
				s.SetStatus(id, status)
				synced++
			}
		} else {
			s.AddTask(id, todo.Content)
			if status != task.StatusReady {
				s.SetStatus(id, status)
			}
			synced++
		}
	}

	if synced > 0 {
		fmt.Printf("Synced %d tasks from TodoWrite\n", synced)
	}
	if skipped > 0 {
		fmt.Printf("Skipped %d session-level items (use 'tk suggest' to check)\n", skipped)
	}
	return nil
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

// Git hook constants
const (
	tasukuHookStart = "# --- TASUKU HOOK START ---"
	tasukuHookEnd   = "# --- TASUKU HOOK END ---"
)

// getTasukuHookContent returns the Tasuku-specific hook content for a given hook name
func getTasukuHookContent(hookName string) string {
	switch hookName {
	case "pre-commit":
		return `# Tasuku pre-commit hook: validate task storage
if [ -d .tasuku ] || [ -f .tasuku.json ]; then
    tk validate
    if [ $? -ne 0 ]; then
        echo "Tasuku validation failed. Please fix issues before committing."
        exit 1
    fi
fi`
	case "post-commit":
		return `# Tasuku post-commit hook: suggest task status updates
COMMIT_MSG=$(git log -1 --pretty=%B)

if [[ $COMMIT_MSG =~ \(#([a-zA-Z0-9-]+)\) ]]; then
    TASK_ID="${BASH_REMATCH[1]}"
    echo ""
    echo "Detected task reference: #$TASK_ID"
    echo "Consider: tk done $TASK_ID"
fi`
	default:
		return ""
	}
}

// wrapTasukuSection wraps content with Tasuku marker comments
func wrapTasukuSection(content string) string {
	return fmt.Sprintf("%s\n%s\n%s", tasukuHookStart, content, tasukuHookEnd)
}

// installHookWithMarkers installs a hook while preserving existing content
func installHookWithMarkers(hookPath, hookName string) error {
	tasukuContent := getTasukuHookContent(hookName)
	if tasukuContent == "" {
		return fmt.Errorf("unknown hook: %s", hookName)
	}

	wrappedContent := wrapTasukuSection(tasukuContent)

	// Check if hook file already exists
	existingContent, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing hook, create new one with shebang
			newContent := "#!/bin/bash\n\n" + wrappedContent + "\n"
			return os.WriteFile(hookPath, []byte(newContent), 0755)
		}
		return fmt.Errorf("failed to read existing hook: %w", err)
	}

	existingStr := string(existingContent)

	// Check if Tasuku section already exists
	startIdx := strings.Index(existingStr, tasukuHookStart)
	endIdx := strings.Index(existingStr, tasukuHookEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Replace existing Tasuku section
		before := existingStr[:startIdx]
		after := existingStr[endIdx+len(tasukuHookEnd):]
		newContent := before + wrappedContent + after
		return os.WriteFile(hookPath, []byte(newContent), 0755)
	}

	// Append Tasuku section to existing hook
	var newContent string
	if strings.HasSuffix(existingStr, "\n") {
		newContent = existingStr + "\n" + wrappedContent + "\n"
	} else {
		newContent = existingStr + "\n\n" + wrappedContent + "\n"
	}

	return os.WriteFile(hookPath, []byte(newContent), 0755)
}

// removeTasukuSection removes only the Tasuku section from a hook file
// Returns (fileDeleted, sectionFound, error)
func removeTasukuSection(hookPath string) (deleted bool, found bool, err error) {
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("failed to read hook: %w", err)
	}

	contentStr := string(content)

	// Find Tasuku section
	startIdx := strings.Index(contentStr, tasukuHookStart)
	endIdx := strings.Index(contentStr, tasukuHookEnd)

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		// No Tasuku section found
		return false, false, nil
	}

	// Remove the Tasuku section including surrounding whitespace
	before := contentStr[:startIdx]
	after := contentStr[endIdx+len(tasukuHookEnd):]

	// Clean up extra whitespace
	before = strings.TrimRight(before, " \t")
	after = strings.TrimLeft(after, " \t")

	// If before ends with newlines and after starts with newlines, normalize
	before = strings.TrimRight(before, "\n")
	after = strings.TrimLeft(after, "\n")

	var newContent string
	if before != "" && after != "" {
		newContent = before + "\n\n" + after
	} else if before != "" {
		newContent = before + "\n"
	} else if after != "" {
		newContent = after + "\n"
	} else {
		newContent = ""
	}

	// Check if the remaining content is essentially empty (just shebang or whitespace)
	trimmed := strings.TrimSpace(newContent)
	isEmptyOrShebangOnly := trimmed == "" ||
		trimmed == "#!/bin/bash" ||
		trimmed == "#!/bin/sh" ||
		trimmed == "#!/usr/bin/env bash" ||
		trimmed == "#!/usr/bin/env sh"

	if isEmptyOrShebangOnly {
		// Delete the file entirely
		if err := os.Remove(hookPath); err != nil {
			return false, true, fmt.Errorf("failed to delete empty hook: %w", err)
		}
		return true, true, nil
	}

	// Write the cleaned content back
	return false, true, os.WriteFile(hookPath, []byte(newContent), 0755)
}

func installGitHooks() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}

	hooksDir := ".git/hooks"

	hooks := []struct {
		name        string
		description string
	}{
		{"pre-commit", "validates Tasuku storage"},
		{"post-commit", "suggests task status updates"},
	}

	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook.name)
		if err := installHookWithMarkers(hookPath, hook.name); err != nil {
			return fmt.Errorf("failed to install %s hook: %w", hook.name, err)
		}
	}

	fmt.Println("Git hooks installed:")
	for _, hook := range hooks {
		fmt.Printf("  - %s: %s\n", hook.name, hook.description)
	}
	fmt.Println("\nNote: Existing hook content has been preserved.")

	return nil
}

func uninstallGitHooks() error {
	hooksDir := ".git/hooks"

	hooks := []string{"pre-commit", "post-commit"}
	removedCount := 0

	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook)
		deleted, found, err := removeTasukuSection(hookPath)
		if err != nil {
			return fmt.Errorf("failed to uninstall %s: %w", hook, err)
		}
		if !found {
			continue
		}
		if deleted {
			fmt.Printf("Removed: %s (file deleted - was empty)\n", hook)
		} else {
			fmt.Printf("Removed Tasuku section from: %s\n", hook)
		}
		removedCount++
	}

	if removedCount == 0 {
		fmt.Println("No Tasuku hooks found to uninstall")
	} else {
		fmt.Println("\nTasuku hooks uninstalled (other hook content preserved)")
	}

	return nil
}
