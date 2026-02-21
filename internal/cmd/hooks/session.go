package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

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
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		// No usable storage — nothing to remind about. Degrade gracefully.
		return nil
	}
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
	fmt.Println("   - Did you make any architectural decisions? → /tasuku:decide")
	fmt.Println("   - Did you discover any gotchas or insights? → /tasuku:learn")
	fmt.Println("   - Any 'never do X' or 'always do Y' patterns? → /tasuku:learn (auto-flagged as rule)")

	if hasReminders {
		fmt.Println("\n================================")
	}

	return nil
}

func hookPreCompact() error {
	fmt.Println("=== Pre-Compaction Checkpoint ===")
	fmt.Println("Context is about to be summarized. Capture important insights NOW!")
	fmt.Println()

	s, _ := store.DefaultStorageWithWarning()
	// Ignore storage errors — git activity and reflection prompts still run without Tasuku.

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
	if s != nil && s.Exists() {
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
	fmt.Println("   → /tasuku:decide (guided decision recording)")
	fmt.Println()
	fmt.Println("   LEARNINGS - Did you discover gotchas, patterns, or behaviors?")
	fmt.Println("   → /tasuku:learn \"insight here\"")
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

func hookSession() error {
	// Check for hook version mismatch (check both local and global)
	checkHookVersionAndWarn()

	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		// Print to stdout so Claude sees it (SessionStart stdout → Claude's context).
		// Don't return the error — a non-zero exit shows "hook error" in the banner
		// but the system message to the LLM says "success", leaving the AI blind.
		fmt.Printf("Tasuku: %v\n", err)
		return nil
	}
	if !s.Exists() {
		return nil
	}

	f, err := s.Read()
	if err != nil {
		fmt.Printf("Tasuku: failed to read task store: %v\n", err)
		return nil
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

	// Check for stale timers (running 4+ hours)
	const staleTimerThreshold = 4 * time.Hour
	var staleTimers []string
	for id, t := range f.Tasks {
		if t.TimerStart != nil {
			elapsed := time.Since(*t.TimerStart)
			if elapsed >= staleTimerThreshold {
				staleTimers = append(staleTimers, fmt.Sprintf("%s (%s)", id, elapsed.Round(time.Minute)))
			}
		}
	}
	if len(staleTimers) > 0 {
		fmt.Printf("\n⚠️  Stale timers (running %s+):\n", staleTimerThreshold)
		for _, s := range staleTimers {
			fmt.Printf("   - %s\n", s)
		}
		fmt.Println("   Stop with: tk task timer stop <id>")
	}

	if highestPriority != nil {
		t := f.Tasks[*highestPriority]
		fmt.Printf("\nNext task: %s\n  %s\n", *highestPriority, t.Description)
	}

	fmt.Println("======================")
	return nil
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

func hookSync() error {
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
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

// checkHookVersionAndWarn checks if installed hooks are outdated and warns the user.
// Checks both local (./.claude/) and global (~/.claude/) installations.
func checkHookVersionAndWarn() {
	// Check local first, then global
	for _, local := range []bool{true, false} {
		installed, current, needsUpdate := CheckHookVersion(local)
		if needsUpdate {
			location := "global"
			flag := ""
			if local {
				location = "project"
				flag = " --local"
			}
			fmt.Printf("⬆️  Hooks outdated (%s): %s → %s\n", location, installed, current)
			fmt.Printf("   Run: tk hooks install --force%s\n", flag)
			fmt.Println()
			return // Only show one warning
		}
	}
}
