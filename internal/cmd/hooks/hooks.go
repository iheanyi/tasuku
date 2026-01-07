// Package hooks provides CLI commands for managing git hooks and AI integration.
package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
  install    Install hooks (git, Claude Code, Codex, OpenCode)
  uninstall  Remove hooks (git, Claude Code, Codex, OpenCode)

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
	cmd.AddCommand(planSyncCmd)
	cmd.AddCommand(todoCheckCmd)
	cmd.AddCommand(subagentDoneCmd)
	cmd.AddCommand(promptCheckCmd)
	cmd.AddCommand(codexNotifyCmd)

	return cmd
}

// Cmd is the parent command for all hooks operations
var Cmd = newHooksCmd()

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Tasuku hooks",
		Long: `Install Tasuku hooks for git and AI tools (Claude Code, Codex, OpenCode).

By default, installs all hooks. Use flags to install specific hooks only.

Git hooks (always local to .git/hooks/):
  - pre-commit: Validates Tasuku storage before commits
  - post-commit: Suggests task status updates based on commit messages

Claude Code hooks (global ~/.claude/ by default, or local ./.claude/ with --local):
  - SessionStart: Shows project context summary when session begins
  - Stop: Reminds about running timers and in-progress tasks
  - ExitPlanMode: Prompts to sync plan to Tasuku tasks

Codex hooks (global ~/.codex/config.toml):
  - notify: Called on agent turn completion

OpenCode hooks (via plugin in ~/.config/opencode/plugin/ or .opencode/plugin/):
  - session.created: Shows context summary at session start
  - session.idle: Reminds about running timers
  - todo.updated: Checks for project-level tasks

Examples:
  tk hooks install              # Git + Claude + Codex + OpenCode (global)
  tk hooks install --local      # Git + Claude + OpenCode (local to project)
  tk hooks install --git        # Install only git hooks
  tk hooks install --claude     # Install only Claude Code hooks (global)
  tk hooks install --codex      # Install only Codex hooks
  tk hooks install --opencode   # Install only OpenCode hooks (global)
  tk hooks install --opencode --local  # OpenCode hooks local to project`,
		RunE: runInstall,
	}

	cmd.Flags().Bool("git", false, "Install git hooks only")
	cmd.Flags().Bool("claude", false, "Install Claude Code hooks only")
	cmd.Flags().Bool("codex", false, "Install Codex hooks only")
	cmd.Flags().Bool("opencode", false, "Install OpenCode hooks only")
	cmd.Flags().Bool("force", false, "Overwrite existing hooks")
	cmd.Flags().Bool("local", false, "Install hooks to project instead of global")

	return cmd
}

func newUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove Tasuku hooks",
		Long: `Remove Tasuku hooks from git and AI tools (Claude Code, Codex, OpenCode).

By default, removes all hooks. Use flags to remove specific hooks only.

Examples:
  tk hooks uninstall                     # Remove all hooks
  tk hooks uninstall --git               # Remove only git hooks
  tk hooks uninstall --claude            # Remove Claude Code hooks (global)
  tk hooks uninstall --claude --local    # Remove Claude Code hooks (project)
  tk hooks uninstall --codex             # Remove Codex hooks
  tk hooks uninstall --opencode          # Remove OpenCode hooks (global)
  tk hooks uninstall --opencode --local  # Remove OpenCode hooks (project)`,
		RunE: runUninstall,
	}

	cmd.Flags().Bool("git", false, "Remove git hooks only")
	cmd.Flags().Bool("claude", false, "Remove Claude Code hooks only")
	cmd.Flags().Bool("codex", false, "Remove Codex hooks only")
	cmd.Flags().Bool("opencode", false, "Remove OpenCode hooks only")
	cmd.Flags().Bool("local", false, "Remove hooks from project instead of global")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	gitOnly, _ := cmd.Flags().GetBool("git")
	claudeOnly, _ := cmd.Flags().GetBool("claude")
	codexOnly, _ := cmd.Flags().GetBool("codex")
	opencodeOnly, _ := cmd.Flags().GetBool("opencode")
	force, _ := cmd.Flags().GetBool("force")
	local, _ := cmd.Flags().GetBool("local")

	// Determine what to install based on flags
	anySpecific := gitOnly || claudeOnly || codexOnly || opencodeOnly
	installGit := !anySpecific || gitOnly
	installClaude := !anySpecific || claudeOnly
	installCodex := !anySpecific || codexOnly
	installOpenCode := !anySpecific || opencodeOnly

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

	if installCodex {
		// Codex hooks are always global (no local option)
		if err := installCodexHooks(force); err != nil {
			errors = append(errors, fmt.Sprintf("codex: %v", err))
		}
	}

	if installOpenCode {
		if err := installOpenCodeHooks(force, local); err != nil {
			errors = append(errors, fmt.Sprintf("opencode: %v", err))
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
	codexOnly, _ := cmd.Flags().GetBool("codex")
	opencodeOnly, _ := cmd.Flags().GetBool("opencode")
	local, _ := cmd.Flags().GetBool("local")

	// Determine what to uninstall based on flags
	anySpecific := gitOnly || claudeOnly || codexOnly || opencodeOnly
	uninstallGit := !anySpecific || gitOnly
	uninstallClaude := !anySpecific || claudeOnly
	uninstallCodex := !anySpecific || codexOnly
	uninstallOpenCode := !anySpecific || opencodeOnly

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

	if uninstallCodex {
		if err := uninstallCodexHooks(); err != nil {
			errors = append(errors, fmt.Sprintf("codex: %v", err))
		}
	}

	if uninstallOpenCode {
		if err := uninstallOpenCodeHooks(local); err != nil {
			errors = append(errors, fmt.Sprintf("opencode: %v", err))
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
  - Decisions or learnings that should be recorded

Examples:
  tk hooks stop-reminder   # Check for reminders`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookStopReminder()
	},
}

var preCompactCmd = &cobra.Command{
	Use:   "pre-compact",
	Short: "Capture decisions and learnings before context compaction",
	Long: `Called by Claude Code PreCompact hook before context window is summarized.

This is a critical checkpoint to capture insights before they're lost:
  - Prompts for architectural decisions made during the session
  - Prompts for learnings or gotchas discovered
  - Lists any in-progress tasks that need status updates
  - Shows recent git activity that might warrant documentation

Examples:
  tk hooks pre-compact   # Run pre-compaction checklist`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookPreCompact()
	},
}

var todoCheckCmd = &cobra.Command{
	Use:   "todo-check",
	Short: "Check if TodoWrite items should persist to Tasuku",
	Long: `Called by Claude Code PostToolUse hook after TodoWrite is used.

Analyzes the todos that were just written and suggests persisting
project-level tasks to Tasuku:
  - Features, bugs, refactors → suggest adding to tk
  - Implementation steps → keep in TodoWrite only

This bridges session-level tracking (TodoWrite) with project-level
persistence (Tasuku) automatically.

The hook receives TodoWrite JSON via TOOL_INPUT environment variable.

Examples:
  tk hooks todo-check   # Analyze TodoWrite output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookTodoCheck()
	},
}

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

var promptCheckCmd = &cobra.Command{
	Use:   "prompt-check",
	Short: "Detect task-related intent in user prompts",
	Long: `Called by Claude Code UserPromptSubmit hook when user sends a message.

Analyzes the user's prompt to detect task-related intent:
  - "implement X" / "fix bug Y" → suggest creating a task if none exists
  - References to existing task IDs → show task context
  - Work that should be tracked → gentle reminder about task creation

This helps ensure significant work gets tracked in Tasuku from the start.

The hook receives the user prompt via USER_PROMPT environment variable.

Examples:
  tk hooks prompt-check   # Analyze user prompt for task intent`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookPromptCheck()
	},
}

var codexNotifyCmd = &cobra.Command{
	Use:   "codex-notify [event-json]",
	Short: "Handle Codex notify callback",
	Long: `Called by Codex when an agent turn completes.

Codex sends a JSON payload as the first argument with:
  - type: "agent-turn-complete"
  - thread-id: session identifier
  - turn-id: turn identifier
  - cwd: working directory
  - input-messages: user messages
  - last-assistant-message: assistant response text

This hook displays the same reminders as stop-reminder for session end.

Examples:
  tk hooks codex-notify '{"type":"agent-turn-complete",...}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookCodexNotify(args)
	},
}

func hookCodexNotify(args []string) error {
	// Codex passes JSON as first argument
	// For now, we just run the stop reminder functionality
	// Future: parse the JSON to extract context-specific info

	if len(args) > 0 {
		var event struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(args[0]), &event); err == nil {
			// Only run on agent-turn-complete events
			if event.Type != "agent-turn-complete" {
				return nil
			}
		}
	}

	// Run the standard stop reminder
	return hookStopReminder()
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

	// Always prompt for reflection at session end
	if !hasReminders {
		fmt.Println("=== Tasuku Session Reminder ===")
		hasReminders = true
	}
	fmt.Println("\n💡 Before ending this session, reflect:")
	fmt.Println("   - Did you make any architectural decisions? → tk decide")
	fmt.Println("   - Did you discover any gotchas or insights? → tk learn")
	fmt.Println("   - Any 'never do X' or 'always do Y' patterns? → tk learn (auto-flagged as rule)")

	if hasReminders {
		fmt.Println("\n================================")
	}

	return nil
}

func hookPreCompact() error {
	fmt.Println("=== Pre-Compaction Checkpoint ===")
	fmt.Println("Context is about to be summarized. Capture important insights NOW!")
	fmt.Println()

	s := store.DefaultStorageWithWarning()

	// Check for recent git activity
	hasGitActivity := false
	if out, err := execCommand("git", "log", "--oneline", "-5", "--since=1 hour ago"); err == nil && len(out) > 0 {
		hasGitActivity = true
		fmt.Println("📝 Recent commits (last hour):")
		lines := strings.Split(strings.TrimSpace(out), "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("   %s\n", line)
			}
		}
		fmt.Println()
	}

	// Check for uncommitted changes
	if out, err := execCommand("git", "status", "--porcelain"); err == nil && len(out) > 0 {
		lineCount := len(strings.Split(strings.TrimSpace(out), "\n"))
		fmt.Printf("📁 Uncommitted changes: %d file(s) modified\n\n", lineCount)
	}

	// Check Tasuku state
	if s.Exists() {
		f, err := s.Read()
		if err == nil {
			// In-progress tasks
			var inProgress []string
			for id, t := range f.Tasks {
				if t.Status == task.StatusInProgress {
					inProgress = append(inProgress, id)
				}
			}
			if len(inProgress) > 0 {
				fmt.Printf("🔄 Tasks in progress (%d):\n", len(inProgress))
				for _, id := range inProgress {
					t := f.Tasks[id]
					desc := t.Description
					if len(desc) > 50 {
						desc = desc[:47] + "..."
					}
					fmt.Printf("   - %s: %s\n", id, desc)
				}
				fmt.Println()
			}

			// Running timers
			var timers []string
			for id, t := range f.Tasks {
				if t.IsTimerRunning() {
					timers = append(timers, id)
				}
			}
			if len(timers) > 0 {
				fmt.Printf("⏱️  Running timers: %v\n\n", timers)
			}
		}
	}

	// The key reflection prompts
	fmt.Println("🧠 CAPTURE BEFORE CONTEXT IS LOST:")
	fmt.Println()
	fmt.Println("   DECISIONS - Did you choose between approaches?")
	fmt.Println("   → tk decide --id <name> --chose \"X\" --over \"Y,Z\" --because \"reason\"")
	fmt.Println()
	fmt.Println("   LEARNINGS - Did you discover gotchas, patterns, or behaviors?")
	fmt.Println("   → tk learn \"insight here\"")
	fmt.Println()

	if hasGitActivity {
		fmt.Println("   💡 Your recent commits suggest significant work was done.")
		fmt.Println("      Consider what decisions/learnings should persist!")
		fmt.Println()
	}

	fmt.Println("These persist across sessions and help future agents!")
	fmt.Println("==================================")

	return nil
}

// hookTodoCheck analyzes TodoWrite output and suggests persisting project-level items
// Also detects completed bug fixes and prompts for learnings
func hookTodoCheck() error {
	// Get TodoWrite input from environment (set by Claude Code hook)
	toolInput := os.Getenv("TOOL_INPUT")
	if toolInput == "" {
		// No input, nothing to check
		return nil
	}

	// Parse the TodoWrite JSON
	var input struct {
		Todos []struct {
			Content    string `json:"content"`
			Status     string `json:"status"`
			ActiveForm string `json:"activeForm"`
		} `json:"todos"`
	}

	if err := json.Unmarshal([]byte(toolInput), &input); err != nil {
		// Not valid JSON or wrong format, skip silently
		return nil
	}

	if len(input.Todos) == 0 {
		return nil
	}

	// Check if Tasuku is initialized
	s := store.DefaultStorageWithWarning()
	var existingTasks map[string]bool
	if s.Exists() {
		f, err := s.Read()
		if err == nil {
			existingTasks = make(map[string]bool)
			for id := range f.Tasks {
				existingTasks[id] = true
			}
		}
	}

	// Track completed bug fixes for learning prompt
	var completedBugFixes []string

	// Analyze each todo for project-level indicators
	var suggestions []string
	for _, todo := range input.Todos {
		// Check for completed bug fixes - prompt for learnings
		if todo.Status == "completed" && isBugFixTask(todo.Content) {
			completedBugFixes = append(completedBugFixes, todo.Content)
		}

		if shouldPersistTask(todo.Content) {
			// Check if similar task already exists
			id := generateID(todo.Content)
			if existingTasks != nil && existingTasks[id] {
				continue // Already tracked
			}

			suggestions = append(suggestions, todo.Content)
		}
	}

	// PRIORITY: Prompt for learnings after bug fixes
	// This is the most important prompt - capture insights immediately!
	if len(completedBugFixes) > 0 {
		fmt.Println("🎯 BUG FIX COMPLETED - RECORD YOUR LEARNINGS NOW!")
		fmt.Println()
		for _, fix := range completedBugFixes {
			desc := fix
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Printf("   ✓ %s\n", desc)
		}
		fmt.Println()
		fmt.Println("📝 MANDATORY: Document what you learned:")
		fmt.Println("   → What was the root cause? tk learn \"cause\"")
		fmt.Println("   → What rule prevents this? tk learn \"Never X\" or \"Always Y\"")
		fmt.Println("   → Any gotchas discovered? tk learn \"insight\"")
		fmt.Println()
		fmt.Println("⚠️  If you skip this, the same bug WILL happen again!")
		fmt.Println()
	}

	// Secondary: suggest persisting project-level tasks
	if len(suggestions) > 0 {
		fmt.Println("💡 Some TodoWrite items look project-level:")
		for _, s := range suggestions {
			desc := s
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Printf("   → %s\n", desc)
		}
		fmt.Println()
		fmt.Println("Consider persisting to Tasuku:")
		fmt.Println("   tk task add \"description\" --priority high")
		fmt.Println()
		fmt.Println("Project-level tasks survive across sessions and help future agents!")
	}

	return nil
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

// hookSubagentDone prompts for insights after subagent completion
func hookSubagentDone() error {
	// Get subagent info from environment (set by Claude Code hook)
	agentType := os.Getenv("SUBAGENT_TYPE")
	// duration := os.Getenv("SUBAGENT_DURATION") // Future: filter by duration

	// Only prompt for exploration-type subagents that do significant work
	// Skip trivial agents like "haiku" quick lookups
	significantAgents := map[string]bool{
		"Explore":           true,
		"general-purpose":   true,
		"Plan":              true,
		"code-reviewer":     true,
		"database-design":   true,
		"issue-summarizer":  true,
	}

	if agentType != "" && !significantAgents[agentType] {
		// Not a significant agent type, skip
		return nil
	}

	// Check if there's ongoing Tasuku work
	s := store.DefaultStorageWithWarning()
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
	fmt.Println("   - Patterns or conventions in the codebase? → tk learn")
	fmt.Println("   - Gotchas or unexpected behaviors? → tk learn")
	fmt.Println("   - Design decisions to document? → tk decide")
	fmt.Println()

	return nil
}

// hookPromptCheck analyzes user prompts for task-related intent and rule patterns
func hookPromptCheck() error {
	// Get user prompt from environment (set by Claude Code hook)
	userPrompt := os.Getenv("USER_PROMPT")
	if userPrompt == "" {
		return nil
	}

	// Skip very short prompts (likely follow-ups or confirmations)
	if len(userPrompt) < 20 {
		return nil
	}

	promptLower := strings.ToLower(userPrompt)

	// Check if Tasuku is initialized
	s := store.DefaultStorageWithWarning()
	if !s.Exists() {
		return nil
	}

	f, err := s.Read()
	if err != nil {
		return nil
	}

	// PRIORITY: Check for rule patterns in user prompts
	// Users often give instructions like "always do X" or "never do Y"
	if task.IsRuleLearning(userPrompt) && looksLikeInstruction(userPrompt) {
		// Extract the rule-like portion of the message
		rulePortion := extractRulePortion(userPrompt)
		if rulePortion != "" {
			// Check if we already have a similar learning
			if !hasSimilarLearning(f.Context.Learnings, rulePortion) {
				fmt.Println("📝 RULE DETECTED in your message:")
				displayRulePortion := rulePortion
				if len(displayRulePortion) > 80 {
					displayRulePortion = displayRulePortion[:77] + "..."
				}
				fmt.Printf("   \"%s\"\n", displayRulePortion)
				fmt.Println()
				fmt.Println("   Record this as a project rule:")
				fmt.Printf("   → tk learn \"%s\"\n", escapeForShell(rulePortion))
				fmt.Println()
			}
		}
	}

	// Check for explicit task ID references (e.g., "work on fix-auth-bug")
	for id, t := range f.Tasks {
		if strings.Contains(promptLower, strings.ToLower(id)) {
			// Found task reference - show context
			fmt.Printf("📋 Task referenced: %s\n", id)
			fmt.Printf("   Status: %s\n", t.Status)
			desc := t.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Printf("   %s\n", desc)
			if t.Status == task.StatusReady {
				fmt.Printf("   → Consider: tk task start %s\n", id)
			}
			fmt.Println()
			return nil
		}
	}

	// Keywords that suggest significant work that should be tracked
	workKeywords := []string{
		"implement", "add feature", "build", "create new",
		"fix bug", "fix the", "debug", "resolve issue",
		"refactor", "rewrite", "migrate",
		"set up", "configure", "integrate",
		"add support for", "enable",
	}

	// Check if prompt suggests significant work
	suggestsWork := false
	for _, kw := range workKeywords {
		if strings.Contains(promptLower, kw) {
			suggestsWork = true
			break
		}
	}

	if !suggestsWork {
		return nil
	}

	// Check if there's already an in-progress task
	hasInProgress := false
	for _, t := range f.Tasks {
		if t.Status == task.StatusInProgress {
			hasInProgress = true
			break
		}
	}

	// If significant work requested and no task in progress, suggest creating one
	if !hasInProgress {
		fmt.Println("💡 This looks like significant work.")
		fmt.Println("   Consider creating a task to track it:")
		fmt.Println("   → tk task add \"description\" --priority high")
		fmt.Println()
	}

	return nil
}

// looksLikeInstruction checks if a message looks like an instruction to an agent
// rather than casual conversation
func looksLikeInstruction(text string) bool {
	lower := strings.ToLower(text)

	// Instruction indicators - imperative tone or directive language
	instructionIndicators := []string{
		// Imperative verbs
		"make sure", "ensure", "always", "never", "don't", "do not",
		"use ", "avoid", "prefer", "remember to", "be sure to",
		// Directive language
		"you should", "you must", "you need to", "please ",
		"when you", "if you", "try to",
		// Code/development context
		"the code", "the api", "the function", "the class",
		"sdk", "library", "framework", "codebase",
		"bug", "fix", "implement", "refactor",
	}

	for _, indicator := range instructionIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// extractRulePortion extracts the rule-like sentence from a longer message
func extractRulePortion(text string) string {
	// Split by sentence boundaries
	sentences := splitSentences(text)

	// Find the sentence containing the rule pattern
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if task.IsRuleLearning(sentence) && len(sentence) > 10 {
			return sentence
		}
	}

	// If no single sentence matches, return empty (full text might be too noisy)
	return ""
}

// splitSentences splits text into sentences
func splitSentences(text string) []string {
	// Simple sentence splitting on common terminators
	// Replace multiple terminators with single marker
	text = strings.ReplaceAll(text, "...", ".")
	text = strings.ReplaceAll(text, ". ", ".|")
	text = strings.ReplaceAll(text, "! ", "!|")
	text = strings.ReplaceAll(text, "? ", "?|")
	text = strings.ReplaceAll(text, ".\n", ".|\n")
	text = strings.ReplaceAll(text, "!\n", "!|\n")
	text = strings.ReplaceAll(text, "?\n", "?|\n")

	parts := strings.Split(text, "|")
	var sentences []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			sentences = append(sentences, p)
		}
	}
	return sentences
}

// hasSimilarLearning checks if we already have a similar learning recorded
func hasSimilarLearning(learnings []task.Learning, newText string) bool {
	newLower := strings.ToLower(newText)
	newWords := extractKeywords(newLower)

	for _, l := range learnings {
		existingLower := strings.ToLower(l.Text)

		// Exact substring match
		if strings.Contains(existingLower, newLower) || strings.Contains(newLower, existingLower) {
			return true
		}

		// Keyword overlap check (if >60% keywords match, consider similar)
		existingWords := extractKeywords(existingLower)
		overlap := countOverlap(newWords, existingWords)
		if len(newWords) > 0 && float64(overlap)/float64(len(newWords)) > 0.6 {
			return true
		}
	}

	return false
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
		return `# Tasuku post-commit hook: suggest task status updates and reflection
COMMIT_MSG=$(git log -1 --pretty=%B)

if [[ $COMMIT_MSG =~ \(#([a-zA-Z0-9-]+)\) ]]; then
    TASK_ID="${BASH_REMATCH[1]}"
    echo ""
    echo "Detected task reference: #$TASK_ID"
    echo "Consider: tk done $TASK_ID"
fi

# Prompt for reflection after significant commits
echo ""
echo "💡 Post-commit reflection:"
echo "   - Made an architectural decision? → tk decide"
echo "   - Discovered a gotcha or insight? → tk learn"`
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
