package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	tkhttp "github.com/iheanyi/tasuku/internal/http"
	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

const version = "0.2.0"

// Global flags
var (
	outputFormat string // json, yaml, toml, table
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "tk",
	Short: "Tasuku - agent-first task management",
	Long: `tk is an agent-first task management system designed for AI agents
working on codebases. It prioritizes pull over push, parallel-safety,
minimal context, and human-readable JSON storage.`,
	Version: version,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "table", "Output format: table, json, yaml")

	// Add all commands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(blockCmd)
	rootCmd.AddCommand(unblockCmd)
	rootCmd.AddCommand(readyCmd)
	rootCmd.AddCommand(findCmd)
	rootCmd.AddCommand(priorityCmd)
	rootCmd.AddCommand(learnCmd)
	rootCmd.AddCommand(learningsCmd)
	rootCmd.AddCommand(unlearnCmd)
	rootCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(decideCmd)
	rootCmd.AddCommand(decisionsCmd)
	rootCmd.AddCommand(undecideCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(notesCmd)
	rootCmd.AddCommand(unnoteCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(hooksCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(ownerCmd)
}

// =============================================================================
// Task Commands
// =============================================================================

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create .tasuku.json in current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		if s.Exists() {
			return fmt.Errorf(".tasuku.json already exists")
		}
		if err := s.Init(); err != nil {
			return err
		}
		fmt.Println("Created .tasuku.json")
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")

		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		var tasks []taskEntry

		for id, t := range f.Tasks {
			if status != "" && string(t.Status) != status {
				continue
			}
			tasks = append(tasks, taskEntry{ID: id, Task: t})
		}

		// Sort by status priority, then by task priority, then by ID
		statusOrder := map[task.Status]int{
			task.StatusInProgress: 0,
			task.StatusReady:      1,
			task.StatusBlocked:    2,
			task.StatusDone:       3,
		}
		sort.Slice(tasks, func(i, j int) bool {
			if statusOrder[tasks[i].Task.Status] != statusOrder[tasks[j].Task.Status] {
				return statusOrder[tasks[i].Task.Status] < statusOrder[tasks[j].Task.Status]
			}
			pi, pj := tasks[i].Task.GetPriority(), tasks[j].Task.GetPriority()
			if pi != pj {
				return pi < pj
			}
			return tasks[i].ID < tasks[j].ID
		})

		return outputTasks(tasks)
	},
}

func init() {
	listCmd.Flags().StringP("status", "s", "", "Filter by status: ready, in_progress, blocked, done")
}

var addCmd = &cobra.Command{
	Use:   "add [description]",
	Short: "Add a new task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := args[0]
		id, _ := cmd.Flags().GetString("id")
		priority, _ := cmd.Flags().GetInt("priority")

		if id == "" {
			id = generateID(description)
		}

		s := store.Default()

		var priorityPtr *int
		if priority >= 0 && priority <= 4 {
			priorityPtr = &priority
		}

		if err := s.AddTaskWithPriority(id, description, priorityPtr); err != nil {
			return err
		}

		priorityStr := ""
		if priorityPtr != nil {
			priorityStr = fmt.Sprintf(" (priority: %s)", task.PriorityName(*priorityPtr))
		}
		fmt.Printf("Created task: %s%s\n", id, priorityStr)
		return nil
	},
}

func init() {
	addCmd.Flags().String("id", "", "Task ID (auto-generated if not provided)")
	addCmd.Flags().IntP("priority", "p", -1, "Priority: 0=critical, 1=high, 2=normal, 3=low, 4=backlog")
}

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		t, exists := f.Tasks[taskID]
		if !exists {
			return fmt.Errorf("task not found: %s", taskID)
		}

		notes := f.Context.Notes[taskID]

		return outputTaskDetail(taskID, t, notes)
	},
}

var startCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Mark task as in_progress",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		if err := s.SetStatus(taskID, task.StatusInProgress); err != nil {
			return err
		}

		fmt.Printf("Started: %s\n", taskID)
		return nil
	},
}

var doneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Mark task as done",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		if err := s.SetStatus(taskID, task.StatusDone); err != nil {
			return err
		}

		fmt.Printf("Completed: %s\n", taskID)
		return nil
	},
}

var blockCmd = &cobra.Command{
	Use:   "block [id]",
	Short: "Mark task as blocked",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		by, _ := cmd.Flags().GetString("by")

		blockers := strings.Split(by, ",")
		for i := range blockers {
			blockers[i] = strings.TrimSpace(blockers[i])
		}

		s := store.Default()
		if err := s.BlockTask(taskID, blockers); err != nil {
			return err
		}

		fmt.Printf("Blocked: %s (by: %s)\n", taskID, strings.Join(blockers, ", "))
		return nil
	},
}

func init() {
	blockCmd.Flags().String("by", "", "Comma-separated list of blocking task IDs")
	blockCmd.MarkFlagRequired("by")
}

var unblockCmd = &cobra.Command{
	Use:   "unblock [id]",
	Short: "Remove blockers, set to ready",
	Long: `Remove blockers from a task.

By default, removes ALL blockers and sets the task to ready status.
Use --from to remove only a specific blocker while keeping others.

Examples:
  tk unblock task-1              # Remove all blockers from task-1
  tk unblock task-1 --from task-2  # Remove only task-2 from blockers`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		fromBlocker, _ := cmd.Flags().GetString("from")
		s := store.Default()

		if fromBlocker == "" {
			// Clear all blockers (original behavior)
			if err := s.UnblockTask(taskID); err != nil {
				return err
			}
			fmt.Printf("Unblocked: %s (removed all blockers)\n", taskID)
			return nil
		}

		// Partial unblock: remove only the specified blocker
		if err := s.Update(func(f *task.File) error {
			t, exists := f.Tasks[taskID]
			if !exists {
				return fmt.Errorf("task %q not found", taskID)
			}

			// Find and remove the specific blocker
			found := false
			newBlockers := []string{}
			for _, b := range t.BlockedBy {
				if b == fromBlocker {
					found = true
				} else {
					newBlockers = append(newBlockers, b)
				}
			}

			if !found {
				return fmt.Errorf("task %q is not blocked by %q", taskID, fromBlocker)
			}

			t.BlockedBy = newBlockers
			// If no more blockers, set to ready
			if len(newBlockers) == 0 {
				t.Status = task.StatusReady
			}
			t.UpdatedAt = time.Now().UTC()
			f.Tasks[taskID] = t
			return nil
		}); err != nil {
			return err
		}

		fmt.Printf("Unblocked: %s (removed blocker: %s)\n", taskID, fromBlocker)
		return nil
	},
}

func init() {
	unblockCmd.Flags().String("from", "", "Remove only this specific blocker (partial unblock)")
}

var deleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a task permanently",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		return s.Update(func(f *task.File) error {
			if _, exists := f.Tasks[taskID]; !exists {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// Delete the task
			delete(f.Tasks, taskID)

			// Remove notes for this task
			delete(f.Context.Notes, taskID)

			// Remove this task from any blocked_by arrays in other tasks
			for id, t := range f.Tasks {
				newBlockedBy := []string{}
				for _, blockerID := range t.BlockedBy {
					if blockerID != taskID {
						newBlockedBy = append(newBlockedBy, blockerID)
					}
				}
				if len(newBlockedBy) != len(t.BlockedBy) {
					t.BlockedBy = newBlockedBy
					t.UpdatedAt = time.Now().UTC()
					f.Tasks[id] = t
				}
			}

			fmt.Printf("Deleted: %s\n", taskID)
			return nil
		})
	},
}

var editCmd = &cobra.Command{
	Use:   "edit [id] [new-description]",
	Short: "Update task description",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		newDescription := args[1]
		s := store.Default()

		return s.Update(func(f *task.File) error {
			t, exists := f.Tasks[taskID]
			if !exists {
				return fmt.Errorf("task not found: %s", taskID)
			}

			t.Description = newDescription
			t.UpdatedAt = time.Now().UTC()
			f.Tasks[taskID] = t

			fmt.Printf("Updated: %s\n", taskID)
			return nil
		})
	},
}

var pauseCmd = &cobra.Command{
	Use:   "pause [id]",
	Short: "Revert in_progress task back to ready",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		return s.Update(func(f *task.File) error {
			t, exists := f.Tasks[taskID]
			if !exists {
				return fmt.Errorf("task not found: %s", taskID)
			}

			if t.Status != task.StatusInProgress {
				return fmt.Errorf("task %s is not in_progress (current status: %s)", taskID, t.Status)
			}

			t.Status = task.StatusReady
			t.Owner = nil
			t.UpdatedAt = time.Now().UTC()
			f.Tasks[taskID] = t

			fmt.Printf("Paused: %s (now ready)\n", taskID)
			return nil
		})
	},
}

var ownerCmd = &cobra.Command{
	Use:   "owner [id] [owner-name]",
	Short: "Manage task owner",
	Long: `Manage the owner of a task.

If owner-name is provided, set the owner.
If --clear flag is used, clear the owner.
Otherwise, show the current owner.

Examples:
  tk owner my-task agent-1      # Set owner to agent-1
  tk owner my-task --clear      # Clear owner
  tk owner my-task              # Show current owner`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		clearFlag, _ := cmd.Flags().GetBool("clear")
		s := store.Default()

		// If owner-name is provided, set owner
		if len(args) == 2 {
			ownerName := args[1]
			if err := s.SetOwner(taskID, ownerName); err != nil {
				return err
			}
			fmt.Printf("Set owner of %s to: %s\n", taskID, ownerName)
			return nil
		}

		// If --clear flag, clear owner
		if clearFlag {
			if err := s.ClearOwner(taskID); err != nil {
				return err
			}
			fmt.Printf("Cleared owner of: %s\n", taskID)
			return nil
		}

		// Otherwise, show current owner
		f, err := s.Read()
		if err != nil {
			return err
		}

		t, exists := f.Tasks[taskID]
		if !exists {
			return fmt.Errorf("task not found: %s", taskID)
		}

		if t.Owner == nil {
			fmt.Printf("Task %s has no owner\n", taskID)
		} else {
			fmt.Printf("Owner of %s: %s\n", taskID, *t.Owner)
		}
		return nil
	},
}

func init() {
	ownerCmd.Flags().Bool("clear", false, "Clear the task owner")
}

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show unblocked tasks sorted by priority",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		var tasks []taskEntry

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
					tasks = append(tasks, taskEntry{ID: id, Task: t})
				}
			}
		}

		sort.Slice(tasks, func(i, j int) bool {
			pi, pj := tasks[i].Task.GetPriority(), tasks[j].Task.GetPriority()
			if pi != pj {
				return pi < pj
			}
			return tasks[i].ID < tasks[j].ID
		})

		if outputFormat != "table" {
			return outputTasks(tasks)
		}

		if len(tasks) == 0 {
			fmt.Println("No ready tasks")
			return nil
		}

		fmt.Println("Ready tasks (sorted by priority):")
		for _, t := range tasks {
			priority := task.PriorityName(t.Task.GetPriority())
			desc := t.Task.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Printf("  [%s] %s: %s\n", priority, t.ID, desc)
		}

		return nil
	},
}

var findCmd = &cobra.Command{
	Use:   "find [query]",
	Short: "Search tasks, notes, and learnings",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(args[0])

		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		var results []searchResult

		// Search tasks
		for id, t := range f.Tasks {
			if strings.Contains(strings.ToLower(t.Description), query) ||
				strings.Contains(strings.ToLower(id), query) {
				results = append(results, searchResult{
					Type:    "task",
					ID:      id,
					Content: t.Description,
				})
			}
		}

		// Search notes
		for taskID, notes := range f.Context.Notes {
			for _, note := range notes {
				if strings.Contains(strings.ToLower(note), query) {
					results = append(results, searchResult{
						Type:    "note",
						ID:      taskID,
						Content: note,
					})
				}
			}
		}

		// Search learnings
		for _, learning := range f.Context.Learnings {
			if strings.Contains(strings.ToLower(learning), query) {
				results = append(results, searchResult{
					Type:    "learning",
					Content: learning,
				})
			}
		}

		// Search decisions
		for _, d := range f.Context.Decisions {
			if strings.Contains(strings.ToLower(d.Chose), query) ||
				strings.Contains(strings.ToLower(d.Because), query) ||
				strings.Contains(strings.ToLower(d.ID), query) {
				results = append(results, searchResult{
					Type:    "decision",
					ID:      d.ID,
					Content: fmt.Sprintf("Chose %s because %s", d.Chose, d.Because),
				})
			}
		}

		return outputSearchResults(results, query)
	},
}

var priorityCmd = &cobra.Command{
	Use:   "priority [id] [0-4]",
	Short: "Set task priority",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		priorityStr := args[1]

		var priority int
		switch priorityStr {
		case "0", "critical":
			priority = task.PriorityCritical
		case "1", "high":
			priority = task.PriorityHigh
		case "2", "normal":
			priority = task.PriorityNormal
		case "3", "low":
			priority = task.PriorityLow
		case "4", "backlog":
			priority = task.PriorityBacklog
		default:
			return fmt.Errorf("invalid priority: %s (use 0-4 or critical/high/normal/low/backlog)", priorityStr)
		}

		s := store.Default()
		if err := s.SetPriority(taskID, priority); err != nil {
			return err
		}

		fmt.Printf("Set priority of %s to %s\n", taskID, task.PriorityName(priority))
		return nil
	},
}

// =============================================================================
// Context Commands
// =============================================================================

var learnCmd = &cobra.Command{
	Use:   "learn \"insight\"",
	Short: "Record an insight or knowledge discovered during work",
	Long: `Record an insight, discovery, or piece of knowledge learned while working.
Learnings are stored in the context section and help build project knowledge.

Use --permanent to also append the learning to CLAUDE.md for persistent documentation.

Examples:
  tk learn "Redis connection pooling significantly improves API latency"
  tk learn "The auth middleware must run before rate limiting" --permanent
  tk learn "Users expect the save button in the top-right corner"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		learning := args[0]
		permanent, _ := cmd.Flags().GetBool("permanent")
		s := store.Default()

		if err := s.AddLearning(learning); err != nil {
			return err
		}

		if permanent {
			if err := appendToCLAUDEmd(learning, "learning"); err != nil {
				fmt.Printf("Warning: could not append to CLAUDE.md: %v\n", err)
			} else {
				fmt.Println("Learning added (also appended to CLAUDE.md)")
				return nil
			}
		}

		fmt.Println("Learning added")
		return nil
	},
}

func init() {
	learnCmd.Flags().Bool("permanent", false, "Also append learning to CLAUDE.md")
}

func appendToCLAUDEmd(content, contentType string) error {
	// Read existing CLAUDE.md or create header
	claudePath := "CLAUDE.md"
	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Look for existing Learnings section or append at end
	text := string(existing)
	var section string
	if contentType == "learning" {
		section = "\n\n## Learnings\n\n"
	} else {
		section = "\n\n## Notes\n\n"
	}

	entry := fmt.Sprintf("- %s\n", content)

	if strings.Contains(text, "## Learnings") {
		// Append to existing section
		idx := strings.Index(text, "## Learnings")
		endOfLine := strings.Index(text[idx:], "\n") + idx + 1
		// Find next section or end of file
		nextSection := strings.Index(text[endOfLine:], "\n## ")
		if nextSection == -1 {
			text = text + entry
		} else {
			insertAt := endOfLine + nextSection
			text = text[:insertAt] + entry + text[insertAt:]
		}
	} else {
		text = text + section + entry
	}

	return os.WriteFile(claudePath, []byte(text), 0644)
}

var learningsCmd = &cobra.Command{
	Use:   "learnings",
	Short: "List all recorded learnings",
	Long: `Display all learnings recorded in the project context.

Examples:
  tk learnings              # List all learnings
  tk learnings --format json  # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		learnings := f.Context.Learnings
		if len(learnings) == 0 {
			fmt.Println("No learnings recorded yet.")
			fmt.Println("Use: tk learn \"your insight here\"")
			return nil
		}

		switch outputFormat {
		case "json":
			data, _ := json.MarshalIndent(learnings, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(learnings)
			fmt.Print(string(data))
		default:
			fmt.Printf("Learnings (%d):\n\n", len(learnings))
			for i, l := range learnings {
				fmt.Printf("  %d. %s\n", i+1, l)
			}
		}
		return nil
	},
}

var unlearnCmd = &cobra.Command{
	Use:   "unlearn [index or text]",
	Short: "Remove a learning by index or partial match",
	Long: `Remove a learning from the project context.

You can specify either:
- A number (1-based index from 'tk learnings' output)
- A partial text match (case-insensitive)

Examples:
  tk unlearn 3                    # Remove learning #3
  tk unlearn "redis"              # Remove first learning containing "redis"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		query := args[0]

		// Try to parse as number first
		var idx int
		if _, err := fmt.Sscanf(query, "%d", &idx); err == nil && idx > 0 {
			return s.Update(func(f *task.File) error {
				if idx > len(f.Context.Learnings) {
					return fmt.Errorf("index %d out of range (have %d learnings)", idx, len(f.Context.Learnings))
				}
				removed := f.Context.Learnings[idx-1]
				f.Context.Learnings = append(f.Context.Learnings[:idx-1], f.Context.Learnings[idx:]...)
				fmt.Printf("Removed: %s\n", removed)
				return nil
			})
		}

		// Otherwise, search by text
		return s.Update(func(f *task.File) error {
			lowerQuery := strings.ToLower(query)
			for i, l := range f.Context.Learnings {
				if strings.Contains(strings.ToLower(l), lowerQuery) {
					removed := f.Context.Learnings[i]
					f.Context.Learnings = append(f.Context.Learnings[:i], f.Context.Learnings[i+1:]...)
					fmt.Printf("Removed: %s\n", removed)
					return nil
				}
			}
			return fmt.Errorf("no learning found matching %q", query)
		})
	},
}

// detectContextFile finds the appropriate context file for permanent documentation
// based on which AI tools are being used in the project.
func detectContextFile() string {
	// Priority order based on specificity
	contextFiles := []struct {
		path        string
		description string
	}{
		{"CLAUDE.md", "Claude Code"},
		{".cursorrules", "Cursor"},
		{".github/copilot-instructions.md", "GitHub Copilot"},
		{"AGENTS.md", "Generic AI agents"},
		{"AI.md", "Generic AI documentation"},
	}

	for _, cf := range contextFiles {
		if _, err := os.Stat(cf.path); err == nil {
			return cf.path
		}
	}

	// Default to CLAUDE.md if nothing exists
	return "CLAUDE.md"
}

var promoteCmd = &cobra.Command{
	Use:   "promote [index or text]",
	Short: "Promote a learning to permanent documentation",
	Long: `Move a learning from .tasuku.json to your AI context file.

Tasuku auto-detects which context file to use based on your project:
- CLAUDE.md (Claude Code)
- .cursorrules (Cursor)
- .github/copilot-instructions.md (GitHub Copilot)
- AGENTS.md (Generic)

Use --to to specify a custom target file.

Examples:
  tk promote 1                     # Promote learning #1 to auto-detected file
  tk promote "redis"               # Promote learning containing "redis"
  tk promote 1 --to AGENTS.md      # Promote to specific file
  tk promote 1 --keep              # Keep in learnings after promoting`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		query := args[0]
		targetFile, _ := cmd.Flags().GetString("to")
		keep, _ := cmd.Flags().GetBool("keep")

		if targetFile == "" {
			targetFile = detectContextFile()
		}

		f, err := s.Read()
		if err != nil {
			return err
		}

		var learning string
		var learningIdx int = -1

		// Try to parse as number first
		var idx int
		if _, err := fmt.Sscanf(query, "%d", &idx); err == nil && idx > 0 {
			if idx > len(f.Context.Learnings) {
				return fmt.Errorf("index %d out of range (have %d learnings)", idx, len(f.Context.Learnings))
			}
			learning = f.Context.Learnings[idx-1]
			learningIdx = idx - 1
		} else {
			// Search by text
			lowerQuery := strings.ToLower(query)
			for i, l := range f.Context.Learnings {
				if strings.Contains(strings.ToLower(l), lowerQuery) {
					learning = l
					learningIdx = i
					break
				}
			}
		}

		if learning == "" {
			return fmt.Errorf("no learning found matching %q", query)
		}

		// Append to context file
		if err := appendToContextFile(targetFile, learning); err != nil {
			return fmt.Errorf("failed to write to %s: %w", targetFile, err)
		}

		// Remove from learnings unless --keep
		if !keep {
			if err := s.Update(func(f *task.File) error {
				f.Context.Learnings = append(
					f.Context.Learnings[:learningIdx],
					f.Context.Learnings[learningIdx+1:]...,
				)
				return nil
			}); err != nil {
				return err
			}
		}

		if keep {
			fmt.Printf("Promoted to %s (kept in learnings): %s\n", targetFile, learning)
		} else {
			fmt.Printf("Promoted to %s: %s\n", targetFile, learning)
		}
		return nil
	},
}

func init() {
	promoteCmd.Flags().String("to", "", "Target context file (auto-detected if not specified)")
	promoteCmd.Flags().Bool("keep", false, "Keep the learning in .tasuku.json after promoting")
}

func appendToContextFile(filePath, learning string) error {
	existing, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(existing)
	entry := fmt.Sprintf("- %s\n", learning)

	// Find or create Learnings section
	if strings.Contains(text, "## Learnings") {
		// Append after the Learnings header
		idx := strings.Index(text, "## Learnings")
		endOfLine := strings.Index(text[idx:], "\n") + idx + 1

		// Find next section or end of file
		nextSection := strings.Index(text[endOfLine:], "\n## ")
		if nextSection == -1 {
			// No next section, append at end
			text = text + entry
		} else {
			// Insert before next section
			insertAt := endOfLine + nextSection
			text = text[:insertAt] + entry + text[insertAt:]
		}
	} else {
		// Create new Learnings section at end
		if len(text) > 0 && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "\n## Learnings\n\n" + entry
	}

	return os.WriteFile(filePath, []byte(text), 0644)
}

var decideCmd = &cobra.Command{
	Use:   "decide",
	Short: "Record a decision",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		chose, _ := cmd.Flags().GetString("chose")
		over, _ := cmd.Flags().GetString("over")
		because, _ := cmd.Flags().GetString("because")

		if id == "" || chose == "" || because == "" {
			return fmt.Errorf("usage: tk decide --id <id> --chose <choice> --over <options> --because <reason>")
		}

		alternatives := strings.Split(over, ",")
		for i := range alternatives {
			alternatives[i] = strings.TrimSpace(alternatives[i])
		}

		d := task.Decision{
			ID:      id,
			Chose:   chose,
			Over:    alternatives,
			Because: because,
		}

		s := store.Default()
		if err := s.AddDecision(d); err != nil {
			return err
		}

		fmt.Printf("Decision recorded: %s\n", id)
		return nil
	},
}

func init() {
	decideCmd.Flags().String("id", "", "Decision ID")
	decideCmd.Flags().String("chose", "", "The option chosen")
	decideCmd.Flags().String("over", "", "Comma-separated alternatives considered")
	decideCmd.Flags().String("because", "", "Reasoning")
	decideCmd.MarkFlagRequired("id")
	decideCmd.MarkFlagRequired("chose")
	decideCmd.MarkFlagRequired("because")
}

var noteCmd = &cobra.Command{
	Use:   "note [task-id] [text]",
	Short: "Add a note to a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		note := args[1]

		s := store.Default()
		if err := s.AddNote(taskID, note); err != nil {
			return err
		}

		fmt.Printf("Note added to: %s\n", taskID)
		return nil
	},
}

var decisionsCmd = &cobra.Command{
	Use:   "decisions",
	Short: "List all recorded decisions",
	Long: `Display all decisions recorded in the project context.

Examples:
  tk decisions              # List all decisions
  tk decisions --format json  # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		decisions := f.Context.Decisions
		if len(decisions) == 0 {
			fmt.Println("No decisions recorded yet.")
			fmt.Println("Use: tk decide --id <id> --chose <choice> --over <alternatives> --because <reason>")
			return nil
		}

		switch outputFormat {
		case "json":
			data, _ := json.MarshalIndent(decisions, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(decisions)
			fmt.Print(string(data))
		default:
			fmt.Printf("Decisions (%d):\n\n", len(decisions))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, d := range decisions {
				overStr := strings.Join(d.Over, ", ")
				if len(overStr) > 30 {
					overStr = overStr[:27] + "..."
				}
				becauseStr := d.Because
				if len(becauseStr) > 40 {
					becauseStr = becauseStr[:37] + "..."
				}
				fmt.Fprintf(w, "  %s\tChose: %s\tOver: %s\n", d.ID, d.Chose, overStr)
				fmt.Fprintf(w, "  \tBecause: %s\n\n", becauseStr)
			}
			w.Flush()
		}
		return nil
	},
}

var undecideCmd = &cobra.Command{
	Use:   "undecide [id]",
	Short: "Remove a decision by ID",
	Long: `Remove a decision from the project context by its ID.

Examples:
  tk undecide json-format          # Remove decision with ID "json-format"
  tk undecide use-cobra            # Remove decision with ID "use-cobra"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		decisionID := args[0]
		s := store.Default()

		return s.Update(func(f *task.File) error {
			for i, d := range f.Context.Decisions {
				if d.ID == decisionID {
					removed := f.Context.Decisions[i]
					f.Context.Decisions = append(f.Context.Decisions[:i], f.Context.Decisions[i+1:]...)
					fmt.Printf("Removed decision: %s (chose %s)\n", removed.ID, removed.Chose)
					return nil
				}
			}
			return fmt.Errorf("decision not found: %s", decisionID)
		})
	},
}

var notesCmd = &cobra.Command{
	Use:   "notes [task-id]",
	Short: "List notes for a task or all notes",
	Long: `Display notes recorded in the project context.

If task-id is provided, show notes for that specific task.
If no task-id is provided, show all notes grouped by task.

Examples:
  tk notes                    # List all notes grouped by task
  tk notes my-task            # List notes for "my-task"
  tk notes --format json      # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		notes := f.Context.Notes
		if len(notes) == 0 {
			fmt.Println("No notes recorded yet.")
			fmt.Println("Use: tk note <task-id> \"your note here\"")
			return nil
		}

		// If task-id provided, show only that task's notes
		if len(args) == 1 {
			taskID := args[0]
			taskNotes, exists := notes[taskID]
			if !exists || len(taskNotes) == 0 {
				return fmt.Errorf("no notes found for task: %s", taskID)
			}

			switch outputFormat {
			case "json":
				data, _ := json.MarshalIndent(map[string][]string{taskID: taskNotes}, "", "  ")
				fmt.Println(string(data))
			case "yaml":
				data, _ := yaml.Marshal(map[string][]string{taskID: taskNotes})
				fmt.Print(string(data))
			default:
				fmt.Printf("Notes for %s (%d):\n\n", taskID, len(taskNotes))
				for i, note := range taskNotes {
					fmt.Printf("  %d. %s\n", i+1, note)
				}
			}
			return nil
		}

		// Show all notes grouped by task
		switch outputFormat {
		case "json":
			data, _ := json.MarshalIndent(notes, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(notes)
			fmt.Print(string(data))
		default:
			// Sort task IDs for consistent output
			var taskIDs []string
			for taskID := range notes {
				taskIDs = append(taskIDs, taskID)
			}
			sort.Strings(taskIDs)

			totalNotes := 0
			for _, taskNotes := range notes {
				totalNotes += len(taskNotes)
			}

			fmt.Printf("Notes (%d total across %d tasks):\n\n", totalNotes, len(notes))
			for _, taskID := range taskIDs {
				taskNotes := notes[taskID]
				fmt.Printf("  [%s]\n", taskID)
				for i, note := range taskNotes {
					fmt.Printf("    %d. %s\n", i+1, note)
				}
				fmt.Println()
			}
		}
		return nil
	},
}

var unnoteCmd = &cobra.Command{
	Use:   "unnote [task-id] [index]",
	Short: "Remove a note from a task",
	Long: `Remove a note from a task by its 1-based index.

Use 'tk notes <task-id>' to see available notes and their indices.

Examples:
  tk unnote my-task 1         # Remove first note from "my-task"
  tk unnote my-task 3         # Remove third note from "my-task"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		indexStr := args[1]

		var idx int
		if _, err := fmt.Sscanf(indexStr, "%d", &idx); err != nil || idx < 1 {
			return fmt.Errorf("invalid index: %s (must be a positive number)", indexStr)
		}

		s := store.Default()

		return s.Update(func(f *task.File) error {
			taskNotes, exists := f.Context.Notes[taskID]
			if !exists || len(taskNotes) == 0 {
				return fmt.Errorf("no notes found for task: %s", taskID)
			}

			if idx > len(taskNotes) {
				return fmt.Errorf("index %d out of range (task has %d notes)", idx, len(taskNotes))
			}

			removed := taskNotes[idx-1]
			f.Context.Notes[taskID] = append(taskNotes[:idx-1], taskNotes[idx:]...)

			// If task has no more notes, remove the key from the map
			if len(f.Context.Notes[taskID]) == 0 {
				delete(f.Context.Notes, taskID)
			}

			fmt.Printf("Removed note from %s: %s\n", taskID, removed)
			return nil
		})
	},
}

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Output full context as JSON/YAML",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		// Context always outputs structured data
		if outputFormat == "yaml" {
			data, _ := yaml.Marshal(f)
			fmt.Print(string(data))
		} else {
			data, _ := json.MarshalIndent(f, "", "  ")
			fmt.Println(string(data))
		}
		return nil
	},
}

// =============================================================================
// Server Commands
// =============================================================================

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP or HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		httpAddr, _ := cmd.Flags().GetString("http")

		s := store.Default()

		// HTTP mode via --http
		if httpAddr != "" {
			httpServer := tkhttp.New(s)
			return httpServer.Run(httpAddr)
		}

		// HTTP mode via --port (legacy)
		if port > 0 {
			httpServer := tkhttp.New(s)
			return httpServer.Run(fmt.Sprintf(":%d", port))
		}

		// Default: MCP stdio mode
		mcpServer := mcp.New(s)
		return mcpServer.Run()
	},
}

func init() {
	serveCmd.Flags().Int("port", 0, "HTTP port (deprecated, use --http)")
	serveCmd.Flags().String("http", "", "HTTP address (e.g., :3000 or localhost:8080)")
}

// =============================================================================
// Migration Commands
// =============================================================================

var migrateCmd = &cobra.Command{
	Use:   "migrate [source]",
	Short: "Migrate from another task system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		switch source {
		case "beads":
			return migrateFromBeads(dryRun)
		default:
			return fmt.Errorf("unknown migration source: %s\nSupported: beads", source)
		}
	},
}

func init() {
	migrateCmd.Flags().Bool("dry-run", false, "Preview migration without making changes")
}

// =============================================================================
// Hook Commands
// =============================================================================

var hooksCmd = &cobra.Command{
	Use:   "hooks [action]",
	Short: "Manage git hooks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		action := args[0]
		switch action {
		case "install":
			return installHooks()
		case "uninstall":
			return uninstallHooks()
		default:
			return fmt.Errorf("unknown action: %s (use install or uninstall)", action)
		}
	},
}

var hookCmd = &cobra.Command{
	Use:   "hook [name]",
	Short: "Run Claude Code integration hooks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		action := args[0]
		switch action {
		case "session":
			return hookSession()
		case "sync":
			return hookSync()
		default:
			return fmt.Errorf("unknown hook: %s (use session or sync)", action)
		}
	},
}

// =============================================================================
// MCP Commands
// =============================================================================

var mcpCmd = &cobra.Command{
	Use:   "mcp [action]",
	Short: "Manage MCP server for Claude Code",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		action := args[0]
		switch action {
		case "install":
			return mcpInstall()
		case "uninstall":
			return mcpUninstall()
		case "config":
			return mcpConfig()
		default:
			return fmt.Errorf("unknown action: %s (use install, uninstall, or config)", action)
		}
	},
}

// =============================================================================
// Utility Commands
// =============================================================================

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate .tasuku.json",
	Long: `Validate the .tasuku.json file for correctness.

Checks performed:
- Version is supported (must be 1)
- All tasks have non-empty descriptions
- All tasks have valid statuses
- No circular dependencies in blocked_by relationships

Examples:
  tk validate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		if f.Version != 1 {
			return fmt.Errorf("unsupported version: %d", f.Version)
		}

		for id, t := range f.Tasks {
			if t.Description == "" {
				return fmt.Errorf("task %s has empty description", id)
			}
			switch t.Status {
			case task.StatusReady, task.StatusInProgress, task.StatusBlocked, task.StatusDone:
				// Valid
			default:
				return fmt.Errorf("task %s has invalid status: %s", id, t.Status)
			}
		}

		// Detect circular dependencies
		cycles := detectCircularDependencies(f.Tasks)
		if len(cycles) > 0 {
			fmt.Println("Circular dependencies detected:")
			for _, cycle := range cycles {
				fmt.Printf("  %s\n", strings.Join(cycle, " -> "))
			}
			return fmt.Errorf("found %d circular dependency chain(s)", len(cycles))
		}

		fmt.Println("Validation passed")
		return nil
	},
}

// detectCircularDependencies finds all circular dependency chains in blocked_by relationships.
// Uses DFS with path tracking to detect cycles.
func detectCircularDependencies(tasks map[string]task.Task) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	// Track which cycles we've already reported to avoid duplicates
	reportedCycles := make(map[string]bool)

	var dfs func(taskID string, path []string) bool
	dfs = func(taskID string, path []string) bool {
		if inStack[taskID] {
			// Found a cycle - extract the cycle portion
			cycleStart := -1
			for i, id := range path {
				if id == taskID {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := append(path[cycleStart:], taskID)
				// Create a normalized key for this cycle to avoid duplicates
				cycleKey := normalizeCycle(cycle)
				if !reportedCycles[cycleKey] {
					reportedCycles[cycleKey] = true
					cycles = append(cycles, cycle)
				}
			}
			return true
		}

		if visited[taskID] {
			return false
		}

		visited[taskID] = true
		inStack[taskID] = true

		t, exists := tasks[taskID]
		if exists {
			for _, blockerID := range t.BlockedBy {
				dfs(blockerID, append(path, taskID))
			}
		}

		inStack[taskID] = false
		return false
	}

	// Run DFS from each task
	for taskID := range tasks {
		if !visited[taskID] {
			dfs(taskID, []string{})
		}
	}

	return cycles
}

// normalizeCycle creates a canonical string representation of a cycle
// to help detect duplicate cycles reported from different starting points.
func normalizeCycle(cycle []string) string {
	if len(cycle) <= 1 {
		return strings.Join(cycle, ",")
	}
	// Remove the repeated last element (cycle closes the loop)
	nodes := cycle[:len(cycle)-1]
	// Find the lexicographically smallest rotation
	minIdx := 0
	for i := 1; i < len(nodes); i++ {
		if nodes[i] < nodes[minIdx] {
			minIdx = i
		}
	}
	// Rotate to start from the smallest element
	rotated := append(nodes[minIdx:], nodes[:minIdx]...)
	return strings.Join(rotated, ",")
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Output JSON Schema for .tasuku.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		schema := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/iheanyi/tasuku/schema.json",
  "title": "Tasuku File",
  "description": "Schema for .tasuku.json task management file",
  "type": "object",
  "required": ["version", "tasks", "context"],
  "properties": {
    "version": { "type": "integer", "const": 1 },
    "tasks": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "required": ["status", "description", "blocked_by", "created_at", "updated_at"],
        "properties": {
          "status": { "type": "string", "enum": ["ready", "in_progress", "blocked", "done"] },
          "description": { "type": "string" },
          "priority": { "type": "integer", "minimum": 0, "maximum": 4 },
          "blocked_by": { "type": "array", "items": { "type": "string" } },
          "owner": { "type": ["string", "null"] },
          "created_at": { "type": "string", "format": "date-time" },
          "updated_at": { "type": "string", "format": "date-time" }
        }
      }
    },
    "context": {
      "type": "object",
      "required": ["learnings", "decisions", "notes"],
      "properties": {
        "learnings": { "type": "array", "items": { "type": "string" } },
        "decisions": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["id", "chose", "over", "because"],
            "properties": {
              "id": { "type": "string" },
              "chose": { "type": "string" },
              "over": { "type": "array", "items": { "type": "string" } },
              "because": { "type": "string" }
            }
          }
        },
        "notes": {
          "type": "object",
          "additionalProperties": { "type": "array", "items": { "type": "string" } }
        }
      }
    }
  }
}`
		fmt.Println(schema)
		return nil
	},
}

// =============================================================================
// Output Types
// =============================================================================

type taskEntry struct {
	ID   string    `json:"id" yaml:"id"`
	Task task.Task `json:"task" yaml:"task"`
}

type searchResult struct {
	Type    string `json:"type" yaml:"type"`
	ID      string `json:"id,omitempty" yaml:"id,omitempty"`
	Content string `json:"content" yaml:"content"`
}

// =============================================================================
// Output Helpers
// =============================================================================

func outputTasks(tasks []taskEntry) error {
	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(tasks, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(tasks)
		fmt.Print(string(data))
	default: // table
		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, t := range tasks {
			statusIcon := statusToIcon(t.Task.Status)
			desc := t.Task.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			blocked := ""
			if len(t.Task.BlockedBy) > 0 {
				blocked = fmt.Sprintf("(blocked by: %s)", strings.Join(t.Task.BlockedBy, ", "))
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", statusIcon, t.ID, desc, blocked)
		}
		w.Flush()
	}
	return nil
}

func outputTaskDetail(id string, t task.Task, notes []string) error {
	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(map[string]interface{}{
			"id":    id,
			"task":  t,
			"notes": notes,
		}, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(map[string]interface{}{
			"id":    id,
			"task":  t,
			"notes": notes,
		})
		fmt.Print(string(data))
	default: // table
		fmt.Printf("ID:          %s\n", id)
		fmt.Printf("Status:      %s\n", t.Status)
		fmt.Printf("Description: %s\n", t.Description)
		if t.Priority != nil {
			fmt.Printf("Priority:    %s\n", task.PriorityName(*t.Priority))
		}
		if len(t.BlockedBy) > 0 {
			fmt.Printf("Blocked by:  %s\n", strings.Join(t.BlockedBy, ", "))
		}
		if t.Owner != nil {
			fmt.Printf("Owner:       %s\n", *t.Owner)
		}
		fmt.Printf("Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))

		if len(notes) > 0 {
			fmt.Printf("\nNotes:\n")
			for _, note := range notes {
				fmt.Printf("  - %s\n", note)
			}
		}
	}
	return nil
}

func outputSearchResults(results []searchResult, query string) error {
	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(results)
		fmt.Print(string(data))
	default: // table
		if len(results) == 0 {
			fmt.Printf("No results for: %s\n", query)
			return nil
		}

		fmt.Printf("Found %d results for \"%s\":\n", len(results), query)
		for _, r := range results {
			switch r.Type {
			case "task":
				fmt.Printf("  [task] %s: %s\n", r.ID, r.Content)
			case "note":
				fmt.Printf("  [note on %s] %s\n", r.ID, r.Content)
			case "learning":
				fmt.Printf("  [learning] %s\n", r.Content)
			case "decision":
				fmt.Printf("  [decision %s] %s\n", r.ID, r.Content)
			}
		}
	}
	return nil
}

func statusToIcon(s task.Status) string {
	switch s {
	case task.StatusReady:
		return "[ ]"
	case task.StatusInProgress:
		return "[>]"
	case task.StatusBlocked:
		return "[!]"
	case task.StatusDone:
		return "[x]"
	default:
		return "[?]"
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// generateID creates a kebab-case ID from description.
func generateID(desc string) string {
	result := ""
	for _, r := range desc {
		if r >= 'a' && r <= 'z' {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else if r == ' ' && len(result) > 0 && result[len(result)-1] != '-' {
			result += "-"
		}
	}
	result = strings.TrimSuffix(result, "-")
	if len(result) > 32 {
		result = result[:32]
	}
	return result
}

// =============================================================================
// Hook Implementations
// =============================================================================

func hookSession() error {
	s := store.Default()
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

	if highestPriority != nil {
		t := f.Tasks[*highestPriority]
		fmt.Printf("\nNext task: %s\n  %s\n", *highestPriority, t.Description)
	}

	fmt.Println("======================")
	return nil
}

func hookSync() error {
	s := store.Default()
	if !s.Exists() {
		return fmt.Errorf("no .tasuku.json found - run 'tk init' first")
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
	for _, todo := range todos {
		id := generateID(todo.Content)
		if id == "" {
			continue
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
	return nil
}

// =============================================================================
// Migration Implementation
// =============================================================================

// BeadsIssue represents a Beads issue from issues.jsonl
type BeadsIssue struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	Status       string   `json:"status,omitempty"`
	Priority     int      `json:"priority"`
	IssueType    string   `json:"issue_type,omitempty"`
	Assignee     string   `json:"assignee,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	ClosedAt     string   `json:"closed_at,omitempty"`
	CloseReason  string   `json:"close_reason,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Dependencies []struct {
		Type     string `json:"type"`
		TargetID string `json:"target_id"`
	} `json:"dependencies,omitempty"`
}

func migrateFromBeads(dryRun bool) error {
	if _, err := os.Stat(".beads"); os.IsNotExist(err) {
		return fmt.Errorf(".beads directory not found")
	}

	fmt.Println("Migrating from Beads...")

	issuesFile := ".beads/issues.jsonl"
	content, err := os.ReadFile(issuesFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", issuesFile, err)
	}

	lines := strings.Split(string(content), "\n")
	var issues []BeadsIssue

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var issue BeadsIssue
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			fmt.Printf("  Warning: failed to parse line: %v\n", err)
			continue
		}
		issues = append(issues, issue)
	}

	if len(issues) == 0 {
		return fmt.Errorf("no issues found in %s", issuesFile)
	}

	if dryRun {
		fmt.Printf("Found %d issues to migrate:\n", len(issues))
		for _, issue := range issues {
			fmt.Printf("  - %s: %s (%s)\n", issue.ID, issue.Title, issue.Status)
		}
		fmt.Println("\nRun without --dry-run to perform migration")
		return nil
	}

	s := store.Default()
	if !s.Exists() {
		if err := s.Init(); err != nil {
			return err
		}
	}

	migrated := 0
	for _, issue := range issues {
		id := strings.ToLower(issue.ID)
		id = strings.ReplaceAll(id, " ", "-")

		status := task.StatusReady
		switch strings.ToLower(issue.Status) {
		case "open":
			status = task.StatusReady
		case "in_progress", "in-progress", "active":
			status = task.StatusInProgress
		case "blocked", "deferred":
			status = task.StatusBlocked
		case "closed", "done":
			status = task.StatusDone
		}

		createdAt := time.Now().UTC()
		if issue.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, issue.CreatedAt); err == nil {
				createdAt = t
			}
		}
		updatedAt := createdAt
		if issue.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, issue.UpdatedAt); err == nil {
				updatedAt = t
			}
		}

		var priority *int
		if issue.Priority >= 0 && issue.Priority <= 4 {
			priority = &issue.Priority
		}

		var blockedBy []string
		for _, dep := range issue.Dependencies {
			if dep.Type == "blocks" || dep.Type == "blocked_by" {
				blockedBy = append(blockedBy, strings.ToLower(dep.TargetID))
			}
		}

		if err := s.Update(func(f *task.File) error {
			f.Tasks[id] = task.Task{
				Status:      status,
				Description: issue.Title,
				Priority:    priority,
				BlockedBy:   blockedBy,
				Owner:       nil,
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
			}

			if f.Context.Notes == nil {
				f.Context.Notes = make(map[string][]string)
			}
			if issue.Description != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], "Description: "+issue.Description)
			}
			if issue.Notes != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], issue.Notes)
			}
			if issue.CloseReason != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], "Close reason: "+issue.CloseReason)
			}

			return nil
		}); err != nil {
			fmt.Printf("  Warning: failed to add %s: %v\n", id, err)
			continue
		}

		priorityStr := ""
		if priority != nil {
			priorityStr = fmt.Sprintf(" [P%d]", *priority)
		}
		fmt.Printf("  Migrated: %s -> %s (%s%s)\n", issue.ID, id, status, priorityStr)
		migrated++
	}

	fmt.Printf("\nMigration complete: %d tasks imported\n", migrated)
	fmt.Println("Original .beads/ preserved (delete manually if desired)")

	return nil
}

// =============================================================================
// Git Hooks Implementation
// =============================================================================

const (
	tasukuHookStart = "# --- TASUKU HOOK START ---"
	tasukuHookEnd   = "# --- TASUKU HOOK END ---"
)

// getTasukuHookContent returns the Tasuku-specific hook content for a given hook name
func getTasukuHookContent(hookName string) string {
	switch hookName {
	case "pre-commit":
		return `# Tasuku pre-commit hook: validate .tasuku.json
if [ -f .tasuku.json ]; then
    tk validate
    if [ $? -ne 0 ]; then
        echo "Tasuku validation failed. Please fix .tasuku.json before committing."
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

func installHooks() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}

	hooksDir := ".git/hooks"

	hooks := []struct {
		name        string
		description string
	}{
		{"pre-commit", "validates .tasuku.json"},
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

func uninstallHooks() error {
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

// =============================================================================
// MCP Implementation
// =============================================================================

func mcpConfig() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	config := map[string]interface{}{
		"tasuku": map[string]interface{}{
			"command": executable,
			"args":    []string{"serve"},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println("Add this to your Claude Code settings.json under 'mcpServers':")
	fmt.Println()
	fmt.Println(string(data))
	return nil
}

// AITool represents a supported AI tool configuration
type AITool struct {
	Name         string
	SettingsPath string
	MCPKey       string
}

func getSupportedAITools() []AITool {
	home, _ := os.UserHomeDir()
	return []AITool{
		{"Claude Code", home + "/.claude/settings.json", "mcpServers"},
		{"Claude Code (local)", home + "/.claude/settings.local.json", "mcpServers"},
		{"Cursor", home + "/.cursor/mcp.json", "mcpServers"},
		{"Cursor (alt)", home + "/Library/Application Support/Cursor/User/globalStorage/mcp.json", "mcpServers"},
	}
}

func getClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	settingsPath := home + "/.claude/settings.json"
	if _, err := os.Stat(settingsPath); err == nil {
		return settingsPath, nil
	}

	localPath := home + "/.claude/settings.local.json"
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	return settingsPath, nil
}

func mcpInstall() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	tools := getSupportedAITools()
	installedTo := []string{}
	alreadyInstalled := []string{}

	for _, tool := range tools {
		// Check if settings file exists
		if _, err := os.Stat(tool.SettingsPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(tool.SettingsPath)
		if err != nil {
			continue
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			// Try to create directory and empty settings for Cursor
			if strings.Contains(tool.Name, "Cursor") {
				settings = make(map[string]interface{})
			} else {
				continue
			}
		}

		mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
		if !ok {
			mcpServers = make(map[string]interface{})
		}

		if _, exists := mcpServers["tasuku"]; exists {
			alreadyInstalled = append(alreadyInstalled, tool.Name)
			continue
		}

		mcpServers["tasuku"] = map[string]interface{}{
			"command": executable,
			"args":    []string{"serve"},
		}
		settings[tool.MCPKey] = mcpServers

		newData, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			continue
		}

		realPath := tool.SettingsPath
		if info, err := os.Lstat(tool.SettingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
			realPath, _ = os.Readlink(tool.SettingsPath)
		}

		if err := os.WriteFile(realPath, newData, 0644); err != nil {
			continue
		}

		installedTo = append(installedTo, tool.Name)
	}

	// If nothing was installed and nothing was already installed, try Claude Code default
	if len(installedTo) == 0 && len(alreadyInstalled) == 0 {
		settingsPath, _ := getClaudeSettingsPath()
		settings := map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"tasuku": map[string]interface{}{
					"command": executable,
					"args":    []string{"serve"},
				},
			},
		}

		// Ensure directory exists
		dir := filepath.Dir(settingsPath)
		os.MkdirAll(dir, 0755)

		newData, _ := json.MarshalIndent(settings, "", "  ")
		if err := os.WriteFile(settingsPath, newData, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", settingsPath, err)
		}

		fmt.Printf("Tasuku MCP server installed\n")
		fmt.Printf("  Config: %s\n", settingsPath)
		fmt.Printf("  Binary: %s\n", executable)
		fmt.Println()
		fmt.Println("Restart your AI tool to activate the MCP server.")
		return nil
	}

	if len(installedTo) > 0 {
		fmt.Println("Tasuku MCP server installed to:")
		for _, name := range installedTo {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Printf("\nBinary: %s\n", executable)
		fmt.Println("\nRestart your AI tools to activate the MCP server.")
	}

	if len(alreadyInstalled) > 0 {
		fmt.Println("\nAlready installed in:")
		for _, name := range alreadyInstalled {
			fmt.Printf("  - %s\n", name)
		}
	}

	return nil
}

func mcpUninstall() error {
	settingsPath, err := getClaudeSettingsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No Claude Code settings found")
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", settingsPath, err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse %s: %w", settingsPath, err)
	}

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		fmt.Println("No MCP servers configured")
		return nil
	}

	if _, exists := mcpServers["tasuku"]; !exists {
		fmt.Println("Tasuku MCP server is not installed")
		return nil
	}

	delete(mcpServers, "tasuku")
	settings["mcpServers"] = mcpServers

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	realPath := settingsPath
	if info, err := os.Lstat(settingsPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		realPath, err = os.Readlink(settingsPath)
		if err != nil {
			return fmt.Errorf("failed to resolve symlink: %w", err)
		}
	}

	if err := os.WriteFile(realPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", realPath, err)
	}

	fmt.Println("Tasuku MCP server uninstalled from Claude Code")
	fmt.Println("Restart Claude Code to apply changes.")

	return nil
}
