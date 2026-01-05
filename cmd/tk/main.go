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
working on codebases.

Design Principles:
  - Pull over push: Agents query when needed, no constant injections
  - Parallel-safe: File locking for multiple simultaneous agents
  - Minimal context: Only load what's needed for the current task
  - Human-readable: JSON file that can be edited by hand

Getting Started:
  tk init                  # Create .tasuku/ directory
  tk add "My first task"   # Add a task
  tk list                  # View all tasks
  tk start <task-id>       # Begin working on a task
  tk done <task-id>        # Mark task complete

AI Tool Integration:
  tk mcp install           # Auto-configure MCP for Claude Code/Cursor
  tk mcp serve             # Start MCP server (for AI tools)

For full documentation: https://github.com/iheanyi/tasuku`,
	Version: version,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "table", "Output format: table, json, yaml")

	// Task parent command with all task subcommands
	taskCmd.AddCommand(listCmd)
	taskCmd.AddCommand(addCmd)
	taskCmd.AddCommand(showCmd)
	taskCmd.AddCommand(startCmd)
	taskCmd.AddCommand(doneCmd)
	taskCmd.AddCommand(blockCmd)
	taskCmd.AddCommand(unblockCmd)
	taskCmd.AddCommand(readyCmd)
	taskCmd.AddCommand(findCmd)
	taskCmd.AddCommand(priorityCmd)
	taskCmd.AddCommand(deleteCmd)
	taskCmd.AddCommand(editCmd)
	taskCmd.AddCommand(pauseCmd)
	taskCmd.AddCommand(ownerCmd)
	taskCmd.AddCommand(taskStatsCmd)
	taskCmd.AddCommand(taskDepsCmd)
	taskCmd.AddCommand(claimCmd)
	taskCmd.AddCommand(releaseCmd)
	taskCmd.AddCommand(whoCmd)
	taskCmd.AddCommand(tagCmd)
	taskCmd.AddCommand(fieldCmd)
	taskCmd.AddCommand(timerCmd)
	taskCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(taskCmd)

	// Non-task root commands
	rootCmd.AddCommand(initCmd)

	// Noun-verb pattern commands
	rootCmd.AddCommand(learningCmd)
	rootCmd.AddCommand(decisionCmd)
	rootCmd.AddCommand(noteParentCmd)

	// Context parent command (noun-verb pattern)
	contextParentCmd.AddCommand(contextShowCmd)
	contextParentCmd.AddCommand(contextValidateCmd)
	contextParentCmd.AddCommand(contextSchemaCmd)
	rootCmd.AddCommand(contextParentCmd)

	// Server parent command (noun-verb pattern)
	serverCmd.AddCommand(serverStartCmd)
	rootCmd.AddCommand(serverCmd)

	// MCP as top-level command (MCP is a noun - Model Context Protocol)
	rootCmd.AddCommand(mcpCmd)

	// Hooks parent command (includes session/sync from old hook command)
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)
	hooksCmd.AddCommand(hooksSessionCmd)
	hooksCmd.AddCommand(hooksSyncCmd)
	rootCmd.AddCommand(hooksCmd)

	// Migration commands
	rootCmd.AddCommand(migrateCmd)

	// GitHub PR integration (V2.0)
	rootCmd.AddCommand(prCmd)

	// Terminal UI
	rootCmd.AddCommand(uiCmd)

	// Doctor command for diagnosing setup issues
	rootCmd.AddCommand(doctorCmd)

	// Suggest command for agent nudge rule
	rootCmd.AddCommand(suggestCmd)

	// Deprecated commands (hidden, for backward compatibility)
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
	rootCmd.AddCommand(contextCmd)       // deprecated: use 'tk context show'
	rootCmd.AddCommand(serveCmd)         // deprecated: use 'tk server start'
	rootCmd.AddCommand(hookCmd)          // deprecated: use 'tk hooks session/sync'
	rootCmd.AddCommand(validateCmd)      // deprecated: use 'tk context validate'
	rootCmd.AddCommand(schemaCmd)        // deprecated: use 'tk context schema'
}

// =============================================================================
// Task Parent Command
// =============================================================================

var taskCmd = &cobra.Command{
	Use:     "task",
	Aliases: []string{"tasks", "t"},
	Short:   "Manage tasks",
	Long: `Manage tasks in your Tasuku project.

Subcommands:
  list      List all tasks
  add       Create a new task
  show      Show task details
  start     Mark task as in_progress
  done      Mark task as complete
  delete    Delete a task
  edit      Update task description
  pause     Pause work on a task
  block     Mark task as blocked
  unblock   Remove blockers from task
  ready     List tasks ready to work on
  find      Search across all content
  priority  Set task priority
  owner     Manage task ownership
  claim     Claim a task for an agent (V2.0)
  release   Release a claimed task (V2.0)
  who       Show claimed tasks by owner (V2.0)
  tag       Manage task tags (V2.0)
  field     Manage custom fields (V2.0)
  timer     Track time spent on tasks (V2.0)

Examples:
  tk task list                 # List all tasks
  tk task add "New feature"    # Add a new task
  tk task start my-task        # Start working on a task
  tk task claim my-task agent1 # Claim task for agent1
  tk task list --tag backend   # Filter by tag
  tk t ls                      # Short alias for list
  tk tasks ready               # Show ready tasks`,
}

// =============================================================================
// Init Command (not part of task subcommand)
// =============================================================================

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Tasuku in current directory",
	Long: `Initialize a new Tasuku project in the current directory.

Creates a .tasuku/ directory with:
  - tasks/     Individual task JSON files (one per task)
  - archive/   Completed tasks that have been archived
  - context/   Learnings and decisions

Benefits:
  - One file per task = cleaner git diffs, fewer merge conflicts
  - Human-readable JSON, can be edited directly
  - Safe for multiple agents working in parallel

If you have a legacy .tasuku.json file, use 'tk migrate v3' to upgrade.

Examples:
  tk init                    # Create .tasuku/ directory
  tk init && tk add "Setup"  # Initialize and add first task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already initialized
		storageType := store.DetectStorageType(".")
		if storageType != store.StorageTypeNone {
			if storageType == store.StorageTypeDir {
				return fmt.Errorf(".tasuku/ directory already exists")
			}
			return fmt.Errorf(".tasuku.json already exists - run 'tk migrate v3' to upgrade")
		}

		s := store.NewDirStore(store.DirName)
		if err := s.Init(); err != nil {
			return err
		}
		fmt.Println("Created .tasuku/ directory")
		fmt.Println("  tasks/    - Your task files")
		fmt.Println("  archive/  - Archived completed tasks")
		fmt.Println("  context/  - Learnings and decisions")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  tk task add \"Your first task\"")
		fmt.Println("  tk hooks install              # Optional: git hooks")
		fmt.Println("  tk mcp install                # Optional: AI tool integration (Claude Code, Cursor)")
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all tasks",
	Long: `Display all tasks in the project, sorted by status and priority.

Status Icons:
  *  in_progress - Currently being worked on
  -  ready       - Available to start
  !  blocked     - Waiting on other tasks
  +  done        - Completed

Sort Order:
  1. Status: in_progress > ready > blocked > done
  2. Priority: critical (0) > high (1) > normal (2) > low (3) > backlog (4)
  3. Task ID: alphabetically

Filtering:
  Use --status to show only tasks with a specific status.
  Use --tag to show only tasks with a specific tag.

Tree View:
  Use --tree to show tasks in a hierarchical tree format,
  with subtasks indented under their parent tasks.

Examples:
  tk list                    # List all tasks
  tk list -s ready           # Show only ready tasks
  tk list --status done      # Show completed tasks
  tk list --tag backend      # Show tasks with 'backend' tag
  tk list -t bug -s ready    # Combine filters
  tk list -f json            # Output as JSON
  tk list --tree             # Show tasks in tree view
  tk ls                      # Alias for 'list'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")
		tagFilter, _ := cmd.Flags().GetString("tag")
		treeView, _ := cmd.Flags().GetBool("tree")

		s := store.DefaultStorageWithWarning()
		f, err := s.Read()
		if err != nil {
			return err
		}

		var tasks []taskEntry

		for id, t := range f.Tasks {
			if status != "" && string(t.Status) != status {
				continue
			}
			if tagFilter != "" && !t.HasTag(tagFilter) {
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

		if treeView {
			return outputTasksTree(tasks)
		}
		return outputTasks(tasks)
	},
}

func init() {
	listCmd.Flags().StringP("status", "s", "", "Filter by status: ready, in_progress, blocked, done")
	listCmd.Flags().StringP("tag", "t", "", "Filter by tag")
	listCmd.Flags().Bool("tree", false, "Show tasks in tree view with subtasks indented")
}

var addCmd = &cobra.Command{
	Use:   "add <description>",
	Short: "Add a new task",
	Long: `Create a new task with the given description.

A unique task ID is auto-generated from the description (e.g., "Fix login bug"
becomes "fix-login-bug"). You can override this with --id.

Priority Levels (optional --priority flag):
  0 or critical  - Urgent, needs immediate attention
  1 or high      - Important, should be done soon
  2 or normal    - Standard priority (default)
  3 or low       - Can wait
  4 or backlog   - Future consideration

Tags (optional --tag flag):
  Add tags to categorize tasks. Use --tag multiple times or comma-separated.

Subtasks (optional --parent flag):
  Create a subtask under an existing parent task.

New tasks start with "ready" status.

Examples:
  tk add "Implement user authentication"
  tk add "Fix critical bug" -p 0               # Critical priority
  tk add "Refactor database layer" --id db-refactor
  tk add "Update documentation" --priority low
  tk add "Add login page" --tag frontend --tag auth  # Multiple tags (repeated)
  tk add "Add login page" --tag frontend,auth        # Multiple tags (comma-separated)
  tk add "Write unit tests" --parent feature-x       # Create subtask`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := args[0]
		id, _ := cmd.Flags().GetString("id")
		priority, _ := cmd.Flags().GetInt("priority")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		parentID, _ := cmd.Flags().GetString("parent")

		if id == "" {
			id = task.GenerateTaskID(description)
		}

		s := store.DefaultStorageWithWarning()

		var priorityPtr *int
		if priority >= 0 && priority <= 4 {
			priorityPtr = &priority
		}

		// Trim whitespace from tags
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}

		// Create task - either as subtask or regular task
		if parentID != "" {
			if err := s.AddSubtask(id, description, parentID); err != nil {
				return err
			}
			// Set priority and tags if specified
			if priorityPtr != nil {
				s.SetPriority(id, *priorityPtr)
			}
			for _, tag := range tags {
				s.AddTag(id, tag)
			}
		} else {
			if err := s.AddTaskWithTags(id, description, priorityPtr, tags); err != nil {
				return err
			}
		}

		priorityStr := ""
		if priorityPtr != nil {
			priorityStr = fmt.Sprintf(" (priority: %s)", task.PriorityName(*priorityPtr))
		}
		tagDisplay := ""
		if len(tags) > 0 {
			tagDisplay = fmt.Sprintf(" [%s]", strings.Join(tags, ", "))
		}
		parentStr := ""
		if parentID != "" {
			parentStr = fmt.Sprintf(" (subtask of: %s)", parentID)
		}
		fmt.Printf("Created task: %s%s%s%s\n", id, priorityStr, tagDisplay, parentStr)
		return nil
	},
}

func init() {
	addCmd.Flags().String("id", "", "Task ID (auto-generated if not provided)")
	addCmd.Flags().IntP("priority", "p", -1, "Priority: 0=critical, 1=high, 2=normal, 3=low, 4=backlog")
	addCmd.Flags().StringSliceP("tag", "t", nil, "Tags (repeatable or comma-separated)")
	addCmd.Flags().String("parent", "", "Parent task ID to create a subtask")
}

var showCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show task details",
	Long: `Display detailed information about a specific task.

Information Shown:
  - Task ID and description
  - Current status (ready, in_progress, blocked, done)
  - Priority level
  - Owner (if assigned)
  - Blocked by (list of blocking task IDs)
  - Created and updated timestamps
  - Notes attached to the task

Examples:
  tk show my-task            # Show details for "my-task"
  tk show fix-auth -f json   # Output as JSON
  tk show feature-x -f yaml  # Output as YAML`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		s := store.DefaultStorageWithWarning()
		f, err := s.Read()
		if err != nil {
			return err
		}

		t, exists := f.Tasks[taskID]
		if !exists {
			return fmt.Errorf("task not found: %s", taskID)
		}

		notes := f.Context.Notes[taskID]

		return outputTaskDetail(taskID, t, notes, f.Tasks)
	},
}

var startCmd = &cobra.Command{
	Use:   "start <task-id>",
	Short: "Mark task as in_progress",
	Long: `Begin working on a task by setting its status to "in_progress".

This indicates that the task is actively being worked on. Only one
agent should work on a task at a time to avoid conflicts.

Side Effects:
  - Status changes from "ready" to "in_progress"
  - Updates the task's updated_at timestamp

Prerequisites:
  - Task should exist
  - Task should typically be in "ready" status

Examples:
  tk start my-task           # Start working on "my-task"
  tk start fix-bug           # Begin bug fix`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

		if err := s.SetStatus(taskID, task.StatusInProgress); err != nil {
			return err
		}

		fmt.Printf("Started: %s\n", taskID)
		return nil
	},
}

var doneCmd = &cobra.Command{
	Use:   "done <task-id>",
	Short: "Mark task as done",
	Long: `Mark a task as completed.

This indicates that the work for this task has been finished.

Side Effects:
  - Status changes to "done"
  - Updates the task's updated_at timestamp
  - May unblock other tasks that were waiting on this one

If other tasks have this task in their blocked_by list, they may
become "ready" once all their blockers are completed.

Examples:
  tk done my-task            # Mark "my-task" as complete
  tk done fix-bug            # Complete bug fix task`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

		if err := s.SetStatus(taskID, task.StatusDone); err != nil {
			return err
		}

		fmt.Printf("Completed: %s\n", taskID)
		return nil
	},
}

var blockCmd = &cobra.Command{
	Use:   "block <task-id>",
	Short: "Mark task as blocked",
	Long: `Mark a task as blocked by one or more other tasks.

Use this when a task cannot proceed until other tasks are completed.
The --by flag is required and specifies which tasks are blocking.

Side Effects:
  - Status changes to "blocked"
  - Adds specified task IDs to the blocked_by array
  - Updates the task's updated_at timestamp

When all blocking tasks are marked as "done", use 'tk unblock' to
make this task ready again.

Examples:
  tk block my-task --by other-task              # Blocked by one task
  tk block feature --by api --by database       # Blocked by multiple (repeated flag)
  tk block feature --by api,database            # Blocked by multiple (comma-separated)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		blockers, _ := cmd.Flags().GetStringSlice("by")

		// Trim whitespace from each blocker
		for i := range blockers {
			blockers[i] = strings.TrimSpace(blockers[i])
		}

		s := store.DefaultStorageWithWarning()
		if err := s.BlockTask(taskID, blockers); err != nil {
			return err
		}

		fmt.Printf("Blocked: %s (by: %s)\n", taskID, strings.Join(blockers, ", "))
		return nil
	},
}

func init() {
	blockCmd.Flags().StringSlice("by", nil, "Blocking task IDs (repeatable or comma-separated)")
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
		s := store.DefaultStorageWithWarning()

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
	Use:   "delete <task-id>",
	Short: "Delete a task permanently",
	Long: `Permanently remove a task from the project.

WARNING: This action cannot be undone.

Side Effects:
  - Removes the task completely
  - Deletes all notes attached to the task
  - Removes this task from any other task's blocked_by list

Use with caution. Consider marking tasks as "done" instead if you
want to preserve history.

Examples:
  tk delete my-task          # Delete "my-task"
  tk delete obsolete-task    # Remove an obsolete task`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

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
	Use:   "edit <task-id> <new-description>",
	Short: "Update task description",
	Long: `Change the description of an existing task.

The task ID remains unchanged; only the description text is updated.
Use this to clarify, expand, or correct task descriptions.

Examples:
  tk edit my-task "Updated description with more detail"
  tk edit fix-bug "Fix null pointer in UserService.login()"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		newDescription := args[1]
		s := store.DefaultStorageWithWarning()

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
	Use:   "pause <task-id>",
	Short: "Pause work and revert task to ready status",
	Long: `Pause work on an in_progress task, reverting it to ready status.

This command:
  - Changes status from "in_progress" to "ready"
  - Clears the owner assignment
  - Makes the task available for other agents to pick up

Use this when you need to stop working on a task temporarily but
it's not blocked by anything.

Prerequisites:
  - Task must currently be in "in_progress" status

Examples:
  tk pause my-task             # Pause and return to ready
  tk pause feature-x           # Stop work on feature`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

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
		s := store.DefaultStorageWithWarning()

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
	Short: "Show tasks that are ready to work on",
	Long: `List all tasks that are ready to be started, sorted by priority.

A task is considered "ready" when:
  - Status is "ready" (not blocked, in_progress, or done)
  - All blocking tasks (blocked_by) are completed

This helps agents quickly identify what to work on next.

Output:
  Shows task ID, priority, and truncated description.
  Higher priority tasks appear first.

Examples:
  tk ready                     # List ready tasks
  tk ready -f json             # Output as JSON
  tk ready -f yaml             # Output as YAML`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()
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
	Use:   "find <query>",
	Short: "Search across tasks, notes, learnings, and decisions",
	Long: `Search for text across all content in your Tasuku project.

Searches (case-insensitive) in:
  - Task IDs and descriptions
  - Notes text
  - Learnings
  - Decision IDs, choices, and reasoning

Results are grouped by type for easy scanning.

Examples:
  tk find "auth"               # Find anything related to auth
  tk find "database"           # Search for database references
  tk find "redis" -f json      # Output matches as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(args[0])

		s := store.DefaultStorageWithWarning()
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
				if strings.Contains(strings.ToLower(note.Text), query) {
					results = append(results, searchResult{
						Type:    "note",
						ID:      taskID,
						Content: note.Text,
					})
				}
			}
		}

		// Search learnings
		for _, learning := range f.Context.Learnings {
			if strings.Contains(strings.ToLower(learning.Text), query) {
				results = append(results, searchResult{
					Type:    "learning",
					ID:      learning.ID,
					Content: learning.Text,
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
	Use:   "priority <task-id> <level>",
	Short: "Set task priority level",
	Long: `Change the priority of a task.

Priority Levels:
  0 or critical  - Urgent issues requiring immediate attention
  1 or high      - Important tasks to complete soon
  2 or normal    - Standard priority (default for new tasks)
  3 or low       - Can be done when time permits
  4 or backlog   - Future consideration, not actively planned

Priority affects:
  - Sort order in 'tk list' and 'tk ready'
  - Which task is suggested as "next" in session context

Examples:
  tk priority my-task 0          # Set to critical (numeric)
  tk priority my-task critical   # Set to critical (named)
  tk priority fix-bug high       # Set to high priority
  tk priority cleanup backlog    # Move to backlog`,
	Args: cobra.ExactArgs(2),
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

		s := store.DefaultStorageWithWarning()
		if err := s.SetPriority(taskID, priority); err != nil {
			return err
		}

		fmt.Printf("Set priority of %s to %s\n", taskID, task.PriorityName(priority))
		return nil
	},
}

// =============================================================================
// Deprecated Task Command Aliases (Hidden for backward compatibility)
// =============================================================================
// These aliases allow old commands like `tk list` to continue working
// while showing a deprecation warning recommending `tk task list`.

var listCmdAlias = &cobra.Command{
	Use:        "list",
	Aliases:    []string{"ls"},
	Hidden:     true,
	Deprecated: "use 'tk task list' instead",
	Short:      "List all tasks (deprecated: use 'tk task list')",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listCmd.RunE(cmd, args)
	},
}

func init() {
	listCmdAlias.Flags().StringP("status", "s", "", "Filter by status: ready, in_progress, blocked, done")
	listCmdAlias.Flags().StringP("tag", "t", "", "Filter by tag")
	rootCmd.AddCommand(listCmdAlias)
}

var addCmdAlias = &cobra.Command{
	Use:        "add",
	Hidden:     true,
	Deprecated: "use 'tk task add' instead",
	Short:      "Add a new task (deprecated: use 'tk task add')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return addCmd.RunE(cmd, args)
	},
}

func init() {
	addCmdAlias.Flags().String("id", "", "Task ID (auto-generated if not provided)")
	addCmdAlias.Flags().IntP("priority", "p", -1, "Priority: 0=critical, 1=high, 2=normal, 3=low, 4=backlog")
	addCmdAlias.Flags().StringSliceP("tag", "t", nil, "Tags (repeatable or comma-separated)")
	rootCmd.AddCommand(addCmdAlias)
}

var showCmdAlias = &cobra.Command{
	Use:        "show",
	Hidden:     true,
	Deprecated: "use 'tk task show' instead",
	Short:      "Show task details (deprecated: use 'tk task show')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return showCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(showCmdAlias)
}

var startCmdAlias = &cobra.Command{
	Use:        "start",
	Hidden:     true,
	Deprecated: "use 'tk task start' instead",
	Short:      "Mark task as in_progress (deprecated: use 'tk task start')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return startCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(startCmdAlias)
}

var doneCmdAlias = &cobra.Command{
	Use:        "done",
	Hidden:     true,
	Deprecated: "use 'tk task done' instead",
	Short:      "Mark task as done (deprecated: use 'tk task done')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doneCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(doneCmdAlias)
}

var blockCmdAlias = &cobra.Command{
	Use:        "block",
	Hidden:     true,
	Deprecated: "use 'tk task block' instead",
	Short:      "Mark task as blocked (deprecated: use 'tk task block')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return blockCmd.RunE(cmd, args)
	},
}

func init() {
	blockCmdAlias.Flags().StringSlice("by", nil, "Blocking task IDs (repeatable or comma-separated)")
	blockCmdAlias.MarkFlagRequired("by")
	rootCmd.AddCommand(blockCmdAlias)
}

var unblockCmdAlias = &cobra.Command{
	Use:        "unblock",
	Hidden:     true,
	Deprecated: "use 'tk task unblock' instead",
	Short:      "Remove blockers from task (deprecated: use 'tk task unblock')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return unblockCmd.RunE(cmd, args)
	},
}

func init() {
	unblockCmdAlias.Flags().String("from", "", "Remove only this specific blocker (partial unblock)")
	rootCmd.AddCommand(unblockCmdAlias)
}

var deleteCmdAlias = &cobra.Command{
	Use:        "delete",
	Hidden:     true,
	Deprecated: "use 'tk task delete' instead",
	Short:      "Delete a task (deprecated: use 'tk task delete')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmdAlias)
}

var editCmdAlias = &cobra.Command{
	Use:        "edit",
	Hidden:     true,
	Deprecated: "use 'tk task edit' instead",
	Short:      "Update task description (deprecated: use 'tk task edit')",
	Args:       cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return editCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(editCmdAlias)
}

var pauseCmdAlias = &cobra.Command{
	Use:        "pause",
	Hidden:     true,
	Deprecated: "use 'tk task pause' instead",
	Short:      "Pause work on a task (deprecated: use 'tk task pause')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return pauseCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(pauseCmdAlias)
}

var ownerCmdAlias = &cobra.Command{
	Use:        "owner",
	Hidden:     true,
	Deprecated: "use 'tk task owner' instead",
	Short:      "Manage task owner (deprecated: use 'tk task owner')",
	Args:       cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return ownerCmd.RunE(cmd, args)
	},
}

func init() {
	ownerCmdAlias.Flags().Bool("clear", false, "Clear the task owner")
	rootCmd.AddCommand(ownerCmdAlias)
}

var readyCmdAlias = &cobra.Command{
	Use:        "ready",
	Hidden:     true,
	Deprecated: "use 'tk task ready' instead",
	Short:      "Show ready tasks (deprecated: use 'tk task ready')",
	RunE: func(cmd *cobra.Command, args []string) error {
		return readyCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(readyCmdAlias)
}

var findCmdAlias = &cobra.Command{
	Use:        "find",
	Hidden:     true,
	Deprecated: "use 'tk task find' instead",
	Short:      "Search across all content (deprecated: use 'tk task find')",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return findCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(findCmdAlias)
}

var priorityCmdAlias = &cobra.Command{
	Use:        "priority",
	Hidden:     true,
	Deprecated: "use 'tk task priority' instead",
	Short:      "Set task priority (deprecated: use 'tk task priority')",
	Args:       cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return priorityCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(priorityCmdAlias)
}

// =============================================================================
// Context Commands (noun-verb pattern)
// =============================================================================

// -----------------------------------------------------------------------------
// Learning Parent Command and Subcommands
// -----------------------------------------------------------------------------

var learningCmd = &cobra.Command{
	Use:     "learning",
	Short:   "Manage learnings",
	Long:    `Manage project learnings - insights and knowledge discovered during work.`,
	Aliases: []string{"learnings"},
}

var learningListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recorded learnings",
	Long: `Display all learnings recorded in the project context.

Examples:
  tk learning list              # List all learnings
  tk learning list --format json  # Output as JSON`,
	RunE: runLearningList,
}

var learningAddCmd = &cobra.Command{
	Use:   "add \"insight\"",
	Short: "Record an insight or knowledge discovered during work",
	Long: `Record an insight, discovery, or piece of knowledge learned while working.
Learnings are stored in the context section and help build project knowledge.

Use --permanent to also append the learning to CLAUDE.md for persistent documentation.

Examples:
  tk learning add "Redis connection pooling significantly improves API latency"
  tk learning add "The auth middleware must run before rate limiting" --permanent
  tk learning add "Users expect the save button in the top-right corner"`,
	Args: cobra.ExactArgs(1),
	RunE: runLearningAdd,
}

var learningRemoveCmd = &cobra.Command{
	Use:   "remove <id or text>",
	Short: "Remove a learning by ID or partial match",
	Long: `Remove a learning from the project context.

You can specify either:
- An ID (6-character code from 'tk learning list' output, e.g., a3x9k2)
- A partial text match (case-insensitive)

Examples:
  tk learning remove a3x9k2               # Remove learning by ID
  tk learning remove "redis"              # Remove first learning containing "redis"`,
	Args: cobra.ExactArgs(1),
	RunE: runLearningRemove,
}

var learningPromoteCmd = &cobra.Command{
	Use:   "promote <id or text>",
	Short: "Promote a learning to permanent documentation",
	Long: `Move a learning from Tasuku to your AI context file.

Tasuku auto-detects which context file to use based on your project:
- CLAUDE.md (Claude Code)
- .cursorrules (Cursor)
- .github/copilot-instructions.md (GitHub Copilot)
- AGENTS.md (Generic)

Use --to to specify a custom target file.

Examples:
  tk learning promote a3x9k2                # Promote learning by ID to auto-detected file
  tk learning promote "redis"               # Promote learning containing "redis"
  tk learning promote a3x9k2 --to AGENTS.md # Promote to specific file
  tk learning promote a3x9k2 --keep         # Keep in learnings after promoting`,
	Args: cobra.ExactArgs(1),
	RunE: runLearningPromote,
}

var learningRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List all never/always rule learnings",
	Long: `Display learnings that are marked as rules (never/always patterns).

Rules are learnings that contain key instruction words like "never" or "always".
These are typically important guidelines that should be promoted to permanent docs.

Examples:
  tk learning rules              # List all rule learnings
  tk learning rules --format json  # Output as JSON`,
	RunE: runLearningRules,
}

func runLearningRules(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	// Filter for rules only
	var rules []task.Learning
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			rules = append(rules, l)
		}
	}

	if len(rules) == 0 {
		fmt.Println("No rule learnings recorded yet.")
		fmt.Println("Rules are learnings that start with or contain 'never' or 'always'.")
		fmt.Println("Use: tk learning add \"Never use raw SQL queries\"")
		return nil
	}

	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(rules, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(rules)
		fmt.Print(string(data))
	default:
		fmt.Printf("Rules (%d):\n\n", len(rules))
		for _, l := range rules {
			age := formatAge(l.CreatedAt)
			if age != "" {
				fmt.Printf("  [%s] %s (%s)\n", l.ID, l.Text, age)
			} else {
				fmt.Printf("  [%s] %s\n", l.ID, l.Text)
			}
		}
		fmt.Println()
		fmt.Println("Hint: Promote rules to permanent docs with: tk learning promote <id>")
	}
	return nil
}

func init() {
	// Learning subcommand flags
	learningAddCmd.Flags().Bool("permanent", false, "Also append learning to CLAUDE.md")
	learningAddCmd.Flags().Bool("rule", false, "Explicitly mark this learning as a rule")
	learningPromoteCmd.Flags().String("to", "", "Target context file (auto-detected if not specified)")
	learningPromoteCmd.Flags().Bool("keep", false, "Keep the learning in Tasuku after promoting")

	// Register learning subcommands
	learningCmd.AddCommand(learningListCmd)
	learningCmd.AddCommand(learningAddCmd)
	learningCmd.AddCommand(learningRemoveCmd)
	learningCmd.AddCommand(learningPromoteCmd)
	learningCmd.AddCommand(learningRulesCmd)
}

// Shared implementation functions for learning commands
func runLearningList(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	learnings := f.Context.Learnings
	if len(learnings) == 0 {
		fmt.Println("No learnings recorded yet.")
		fmt.Println("Use: tk learning add \"your insight here\"")
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
		// Count rules for header
		ruleCount := 0
		for _, l := range learnings {
			if l.IsRule {
				ruleCount++
			}
		}

		if ruleCount > 0 {
			fmt.Printf("Learnings (%d, %d rules):\n\n", len(learnings), ruleCount)
		} else {
			fmt.Printf("Learnings (%d):\n\n", len(learnings))
		}
		for _, l := range learnings {
			age := formatAge(l.CreatedAt)
			ruleMarker := ""
			if l.IsRule {
				ruleMarker = " [RULE]"
			}
			if age != "" {
				fmt.Printf("  [%s] %s%s (%s)\n", l.ID, l.Text, ruleMarker, age)
			} else {
				fmt.Printf("  [%s] %s%s\n", l.ID, l.Text, ruleMarker)
			}
		}
	}
	return nil
}

func runLearningAdd(cmd *cobra.Command, args []string) error {
	learningText := args[0]
	permanent, _ := cmd.Flags().GetBool("permanent")
	forceRule, _ := cmd.Flags().GetBool("rule")
	s := store.DefaultStorageWithWarning()

	var id string
	var isRule bool
	var err error

	if forceRule {
		// Explicitly mark as rule
		ruleVal := true
		id, isRule, err = s.AddLearningWithRule(learningText, &ruleVal)
	} else {
		// Auto-detect
		id, isRule, err = s.AddLearningWithRule(learningText, nil)
	}
	if err != nil {
		return err
	}

	if permanent {
		if err := appendToCLAUDEmd(learningText, "learning"); err != nil {
			fmt.Printf("Warning: could not append to CLAUDE.md: %v\n", err)
		} else {
			fmt.Printf("Learning added [%s] (also appended to CLAUDE.md)\n", id)
			return nil
		}
	}

	if isRule {
		fmt.Printf("Learning added [%s] [RULE]\n", id)
		fmt.Println("Hint: Consider promoting this rule to permanent docs with: tk learning promote", id)
	} else {
		fmt.Printf("Learning added [%s]\n", id)
	}
	return nil
}

func runLearningRemove(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	query := args[0]

	// First try to remove by ID
	removedText, err := s.RemoveLearning(query)
	if err == nil {
		fmt.Printf("Removed learning: %s\n", removedText)
		return nil
	}

	// If ID not found, try to find by text match
	learning, err := s.FindLearningByText(query)
	if err != nil {
		return fmt.Errorf("no learning found matching %q", query)
	}

	// Remove by the found ID
	removedText, err = s.RemoveLearning(learning.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Removed learning: %s\n", removedText)
	return nil
}

func runLearningPromote(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
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

	var foundLearning *task.Learning

	// First try to find by ID
	for i := range f.Context.Learnings {
		if f.Context.Learnings[i].ID == query {
			foundLearning = &f.Context.Learnings[i]
			break
		}
	}

	// If not found by ID, search by text
	if foundLearning == nil {
		lowerQuery := strings.ToLower(query)
		for i := range f.Context.Learnings {
			if strings.Contains(strings.ToLower(f.Context.Learnings[i].Text), lowerQuery) {
				foundLearning = &f.Context.Learnings[i]
				break
			}
		}
	}

	if foundLearning == nil {
		return fmt.Errorf("no learning found matching %q", query)
	}

	// Append to context file
	if err := appendToContextFile(targetFile, foundLearning.Text); err != nil {
		return fmt.Errorf("failed to write to %s: %w", targetFile, err)
	}

	// Remove from learnings unless --keep
	if !keep {
		if _, err := s.RemoveLearning(foundLearning.ID); err != nil {
			return err
		}
	}

	if keep {
		fmt.Printf("Promoted to %s (kept in learnings): %s\n", targetFile, foundLearning.Text)
	} else {
		fmt.Printf("Promoted to %s: %s\n", targetFile, foundLearning.Text)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Decision Parent Command and Subcommands
// -----------------------------------------------------------------------------

var decisionCmd = &cobra.Command{
	Use:     "decision",
	Short:   "Manage decisions",
	Long:    `Manage architectural and design decisions recorded during development.`,
	Aliases: []string{"decisions"},
}

var decisionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recorded decisions",
	Long: `Display all decisions recorded in the project context.

Examples:
  tk decision list              # List all decisions
  tk decision list --format json  # Output as JSON`,
	RunE: runDecisionList,
}

var decisionAddCmd = &cobra.Command{
	Use:   "add --id <id> --chose <option> --over <alternatives> --because <reason>",
	Short: "Record an architectural or design decision",
	Long: `Document a decision made during development for future reference.

Decisions capture:
  - What was chosen
  - What alternatives were considered
  - Why this choice was made

This creates an audit trail of architectural choices, helping future
developers (or agents) understand why things are the way they are.

Required Flags:
  --id       Unique identifier for this decision (e.g., "use-postgres")
  --chose    The option that was selected
  --because  The reasoning behind the choice

Optional Flags:
  --over     Alternatives considered (repeatable or comma-separated)

Examples:
  tk decision add --id db-choice --chose PostgreSQL --over MySQL --over SQLite --because "Better JSON support"
  tk decision add --id auth-method --chose JWT --over sessions,OAuth --because "Stateless and scalable"
  tk decision add --id framework --chose Cobra --because "Standard Go CLI library"`,
	RunE: runDecisionAdd,
}

var decisionRemoveCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove a decision by ID",
	Long: `Remove a decision from the project context by its ID.

Examples:
  tk decision remove json-format          # Remove decision with ID "json-format"
  tk decision remove use-cobra            # Remove decision with ID "use-cobra"`,
	Args: cobra.ExactArgs(1),
	RunE: runDecisionRemove,
}

func init() {
	// Decision add subcommand flags
	decisionAddCmd.Flags().String("id", "", "Decision ID")
	decisionAddCmd.Flags().String("chose", "", "The option chosen")
	decisionAddCmd.Flags().StringSlice("over", nil, "Alternatives considered (repeatable or comma-separated)")
	decisionAddCmd.Flags().String("because", "", "Reasoning")
	decisionAddCmd.MarkFlagRequired("id")
	decisionAddCmd.MarkFlagRequired("chose")
	decisionAddCmd.MarkFlagRequired("because")

	// Register decision subcommands
	decisionCmd.AddCommand(decisionListCmd)
	decisionCmd.AddCommand(decisionAddCmd)
	decisionCmd.AddCommand(decisionRemoveCmd)
}

// Shared implementation functions for decision commands
func runDecisionList(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	decisions := f.Context.Decisions
	if len(decisions) == 0 {
		fmt.Println("No decisions recorded yet.")
		fmt.Println("Use: tk decision add --id <id> --chose <choice> --over <alternatives> --because <reason>")
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
}

func runDecisionAdd(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	chose, _ := cmd.Flags().GetString("chose")
	alternatives, _ := cmd.Flags().GetStringSlice("over")
	because, _ := cmd.Flags().GetString("because")

	if id == "" || chose == "" || because == "" {
		return fmt.Errorf("usage: tk decision add --id <id> --chose <choice> --over <options> --because <reason>")
	}

	// Trim whitespace from alternatives
	for i := range alternatives {
		alternatives[i] = strings.TrimSpace(alternatives[i])
	}

	d := task.Decision{
		ID:      id,
		Chose:   chose,
		Over:    alternatives,
		Because: because,
	}

	s := store.DefaultStorageWithWarning()
	if err := s.AddDecision(d); err != nil {
		return err
	}

	fmt.Printf("Decision recorded: %s\n", id)
	return nil
}

func runDecisionRemove(cmd *cobra.Command, args []string) error {
	decisionID := args[0]
	s := store.DefaultStorageWithWarning()

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
}

// -----------------------------------------------------------------------------
// Note Parent Command and Subcommands
// -----------------------------------------------------------------------------

var noteParentCmd = &cobra.Command{
	Use:     "note",
	Short:   "Manage notes",
	Long:    `Manage notes attached to tasks.`,
	Aliases: []string{"notes"},
}

var noteListCmd = &cobra.Command{
	Use:   "list [task-id]",
	Short: "List notes for a task or all notes",
	Long: `Display notes recorded in the project context.

If task-id is provided, show notes for that specific task.
If no task-id is provided, show all notes grouped by task.

Examples:
  tk note list                    # List all notes grouped by task
  tk note list my-task            # List notes for "my-task"
  tk note list --format json      # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNoteList,
}

var noteAddCmd = &cobra.Command{
	Use:   "add <task-id> <text>",
	Short: "Add a note to a task",
	Long: `Attach a note to a specific task for additional context.

Notes are useful for:
  - Recording progress updates
  - Documenting blockers or issues encountered
  - Capturing implementation details
  - Leaving messages for other agents

Notes appear when you run 'tk show' for the task.

Examples:
  tk note add my-task "Started implementation of auth flow"
  tk note add fix-bug "Root cause: null pointer in UserService"
  tk note add feature "Waiting for API spec from backend team"`,
	Args: cobra.ExactArgs(2),
	RunE: runNoteAdd,
}

var noteRemoveCmd = &cobra.Command{
	Use:   "remove <task-id> <note-id>",
	Short: "Remove a note from a task",
	Long: `Remove a note from a task by its ID.

Use 'tk note list <task-id>' to see available notes and their IDs.

Examples:
  tk note remove my-task a3x9k2    # Remove note with ID "a3x9k2" from "my-task"
  tk note remove fix-bug b7m4p1    # Remove note with ID "b7m4p1" from "fix-bug"`,
	Args: cobra.ExactArgs(2),
	RunE: runNoteRemove,
}

func init() {
	// Register note subcommands
	noteParentCmd.AddCommand(noteListCmd)
	noteParentCmd.AddCommand(noteAddCmd)
	noteParentCmd.AddCommand(noteRemoveCmd)
}

// Shared implementation functions for note commands
func runNoteList(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	notes := f.Context.Notes
	if len(notes) == 0 {
		fmt.Println("No notes recorded yet.")
		fmt.Println("Use: tk note add <task-id> \"your note here\"")
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
			data, _ := json.MarshalIndent(map[string][]task.Note{taskID: taskNotes}, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(map[string][]task.Note{taskID: taskNotes})
			fmt.Print(string(data))
		default:
			fmt.Printf("Notes for %s:\n", taskID)
			for _, note := range taskNotes {
				fmt.Printf("  [%s] %s\n", note.ID, note.Text)
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
			for _, note := range taskNotes {
				fmt.Printf("    [%s] %s\n", note.ID, note.Text)
			}
			fmt.Println()
		}
	}
	return nil
}

func runNoteAdd(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	note := args[1]

	s := store.DefaultStorageWithWarning()
	noteID, err := s.AddNote(taskID, note)
	if err != nil {
		return err
	}

	fmt.Printf("Note [%s] added to: %s\n", noteID, taskID)
	return nil
}

func runNoteRemove(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	noteID := args[1]

	s := store.DefaultStorageWithWarning()
	removedText, err := s.RemoveNote(taskID, noteID)
	if err != nil {
		return err
	}

	fmt.Printf("Removed note: %s\n", removedText)
	return nil
}

// =============================================================================
// Deprecated Context Commands (kept for backward compatibility)
// =============================================================================

var learnCmd = &cobra.Command{
	Use:        "learn \"insight\"",
	Short:      "Record an insight or knowledge discovered during work",
	Hidden:     true,
	Deprecated: "use 'tk learning add' instead",
	Long:       `Record an insight, discovery, or piece of knowledge learned while working.
Learnings are stored in the context section and help build project knowledge.

Use --permanent to also append the learning to CLAUDE.md for persistent documentation.

Examples:
  tk learn "Redis connection pooling significantly improves API latency"
  tk learn "The auth middleware must run before rate limiting" --permanent
  tk learn "Users expect the save button in the top-right corner"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		learningText := args[0]
		permanent, _ := cmd.Flags().GetBool("permanent")
		s := store.DefaultStorageWithWarning()

		id, err := s.AddLearning(learningText)
		if err != nil {
			return err
		}

		if permanent {
			if err := appendToCLAUDEmd(learningText, "learning"); err != nil {
				fmt.Printf("Warning: could not append to CLAUDE.md: %v\n", err)
			} else {
				fmt.Printf("Learning added [%s] (also appended to CLAUDE.md)\n", id)
				return nil
			}
		}

		fmt.Printf("Learning added [%s]\n", id)
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
	Hidden:     true,
	Deprecated: "use 'tk learning list' instead",
	Long: `Display all learnings recorded in the project context.

Examples:
  tk learnings              # List all learnings
  tk learnings --format json  # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()
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
			for _, l := range learnings {
				age := formatAge(l.CreatedAt)
				if age != "" {
					fmt.Printf("  [%s] %s (%s)\n", l.ID, l.Text, age)
				} else {
					fmt.Printf("  [%s] %s\n", l.ID, l.Text)
				}
			}
		}
		return nil
	},
}

var unlearnCmd = &cobra.Command{
	Use:   "unlearn <id or text>",
	Short: "Remove a learning by ID or partial match",
	Hidden:     true,
	Deprecated: "use 'tk learning remove' instead",
	Long: `Remove a learning from the project context.

You can specify either:
- An ID (6-character code from 'tk learnings' output, e.g., a3x9k2)
- A partial text match (case-insensitive)

Examples:
  tk unlearn a3x9k2               # Remove learning by ID
  tk unlearn "redis"              # Remove first learning containing "redis"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()
		query := args[0]

		// First try to remove by ID
		removedText, err := s.RemoveLearning(query)
		if err == nil {
			fmt.Printf("Removed learning: %s\n", removedText)
			return nil
		}

		// If ID not found, try to find by text match
		learning, err := s.FindLearningByText(query)
		if err != nil {
			return fmt.Errorf("no learning found matching %q", query)
		}

		// Remove by the found ID
		removedText, err = s.RemoveLearning(learning.ID)
		if err != nil {
			return err
		}
		fmt.Printf("Removed learning: %s\n", removedText)
		return nil
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
	Use:   "promote <id or text>",
	Short: "Promote a learning to permanent documentation",
	Hidden:     true,
	Deprecated: "use 'tk learning promote' instead",
	Long: `Move a learning from Tasuku to your AI context file.

Tasuku auto-detects which context file to use based on your project:
- CLAUDE.md (Claude Code)
- .cursorrules (Cursor)
- .github/copilot-instructions.md (GitHub Copilot)
- AGENTS.md (Generic)

Use --to to specify a custom target file.

Examples:
  tk promote a3x9k2                # Promote learning by ID to auto-detected file
  tk promote "redis"               # Promote learning containing "redis"
  tk promote a3x9k2 --to AGENTS.md # Promote to specific file
  tk promote a3x9k2 --keep         # Keep in learnings after promoting`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()
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

		var foundLearning *task.Learning

		// First try to find by ID
		for i := range f.Context.Learnings {
			if f.Context.Learnings[i].ID == query {
				foundLearning = &f.Context.Learnings[i]
				break
			}
		}

		// If not found by ID, search by text
		if foundLearning == nil {
			lowerQuery := strings.ToLower(query)
			for i := range f.Context.Learnings {
				if strings.Contains(strings.ToLower(f.Context.Learnings[i].Text), lowerQuery) {
					foundLearning = &f.Context.Learnings[i]
					break
				}
			}
		}

		if foundLearning == nil {
			return fmt.Errorf("no learning found matching %q", query)
		}

		// Append to context file
		if err := appendToContextFile(targetFile, foundLearning.Text); err != nil {
			return fmt.Errorf("failed to write to %s: %w", targetFile, err)
		}

		// Remove from learnings unless --keep
		if !keep {
			if _, err := s.RemoveLearning(foundLearning.ID); err != nil {
				return err
			}
		}

		if keep {
			fmt.Printf("Promoted to %s (kept in learnings): %s\n", targetFile, foundLearning.Text)
		} else {
			fmt.Printf("Promoted to %s: %s\n", targetFile, foundLearning.Text)
		}
		return nil
	},
}

func init() {
	promoteCmd.Flags().String("to", "", "Target context file (auto-detected if not specified)")
	promoteCmd.Flags().Bool("keep", false, "Keep the learning in Tasuku after promoting")
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
	Use:   "decide --id <id> --chose <option> --over <alternatives> --because <reason>",
	Short: "Record an architectural or design decision",
	Hidden:     true,
	Deprecated: "use 'tk decision add' instead",
	Long: `Document a decision made during development for future reference.

Decisions capture:
  - What was chosen
  - What alternatives were considered
  - Why this choice was made

This creates an audit trail of architectural choices, helping future
developers (or agents) understand why things are the way they are.

Required Flags:
  --id       Unique identifier for this decision (e.g., "use-postgres")
  --chose    The option that was selected
  --because  The reasoning behind the choice

Optional Flags:
  --over     Alternatives considered (repeatable or comma-separated)

Examples:
  tk decide --id db-choice --chose PostgreSQL --over MySQL --over SQLite --because "Better JSON support"
  tk decide --id auth-method --chose JWT --over sessions,OAuth --because "Stateless and scalable"
  tk decide --id framework --chose Cobra --because "Standard Go CLI library"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		chose, _ := cmd.Flags().GetString("chose")
		alternatives, _ := cmd.Flags().GetStringSlice("over")
		because, _ := cmd.Flags().GetString("because")

		if id == "" || chose == "" || because == "" {
			return fmt.Errorf("usage: tk decide --id <id> --chose <choice> --over <options> --because <reason>")
		}

		// Trim whitespace from alternatives
		for i := range alternatives {
			alternatives[i] = strings.TrimSpace(alternatives[i])
		}

		d := task.Decision{
			ID:      id,
			Chose:   chose,
			Over:    alternatives,
			Because: because,
		}

		s := store.DefaultStorageWithWarning()
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
	decideCmd.Flags().StringSlice("over", nil, "Alternatives considered (repeatable or comma-separated)")
	decideCmd.Flags().String("because", "", "Reasoning")
	decideCmd.MarkFlagRequired("id")
	decideCmd.MarkFlagRequired("chose")
	decideCmd.MarkFlagRequired("because")
}

var noteCmd = &cobra.Command{
	Use:   "note <task-id> <text>",
	Short: "Add a note to a task",
	Hidden:     true,
	Deprecated: "use 'tk note add' instead",
	Long: `Attach a note to a specific task for additional context.

Notes are useful for:
  - Recording progress updates
  - Documenting blockers or issues encountered
  - Capturing implementation details
  - Leaving messages for other agents

Notes appear when you run 'tk show' for the task.

Examples:
  tk note my-task "Started implementation of auth flow"
  tk note fix-bug "Root cause: null pointer in UserService"
  tk note feature "Waiting for API spec from backend team"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		note := args[1]

		s := store.DefaultStorageWithWarning()
		noteID, err := s.AddNote(taskID, note)
		if err != nil {
			return err
		}

		fmt.Printf("Note [%s] added to: %s\n", noteID, taskID)
		return nil
	},
}

var decisionsCmd = &cobra.Command{
	Use:   "decisions",
	Short: "List all recorded decisions",
	Hidden:     true,
	Deprecated: "use 'tk decision list' instead",
	Long: `Display all decisions recorded in the project context.

Examples:
  tk decisions              # List all decisions
  tk decisions --format json  # Output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()
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
	Hidden:     true,
	Deprecated: "use 'tk decision remove' instead",
	Long: `Remove a decision from the project context by its ID.

Examples:
  tk undecide json-format          # Remove decision with ID "json-format"
  tk undecide use-cobra            # Remove decision with ID "use-cobra"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		decisionID := args[0]
		s := store.DefaultStorageWithWarning()

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
	Hidden:     true,
	Deprecated: "use 'tk note list' instead",
	Long: `Display notes recorded in the project context.

If task-id is provided, show notes for that specific task.
If no task-id is provided, show all notes grouped by task.

Examples:
  tk notes                    # List all notes grouped by task
  tk notes my-task            # List notes for "my-task"
  tk notes --format json      # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()
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
				data, _ := json.MarshalIndent(map[string][]task.Note{taskID: taskNotes}, "", "  ")
				fmt.Println(string(data))
			case "yaml":
				data, _ := yaml.Marshal(map[string][]task.Note{taskID: taskNotes})
				fmt.Print(string(data))
			default:
				fmt.Printf("Notes for %s:\n", taskID)
				for _, note := range taskNotes {
					fmt.Printf("  [%s] %s\n", note.ID, note.Text)
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
				for _, note := range taskNotes {
					fmt.Printf("    [%s] %s\n", note.ID, note.Text)
				}
				fmt.Println()
			}
		}
		return nil
	},
}

var unnoteCmd = &cobra.Command{
	Use:   "unnote <task-id> <note-id>",
	Short: "Remove a note from a task",
	Hidden:     true,
	Deprecated: "use 'tk note remove' instead",
	Long: `Remove a note from a task by its ID.

Use 'tk notes <task-id>' to see available notes and their IDs.

Examples:
  tk unnote my-task a3x9k2    # Remove note with ID "a3x9k2" from "my-task"
  tk unnote fix-bug b7m4p1    # Remove note with ID "b7m4p1" from "fix-bug"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		noteID := args[1]

		s := store.DefaultStorageWithWarning()
		removedText, err := s.RemoveNote(taskID, noteID)
		if err != nil {
			return err
		}

		fmt.Printf("Removed note: %s\n", removedText)
		return nil
	},
}

// =============================================================================
// Context Parent Command and Subcommands (noun-verb pattern)
// =============================================================================

var contextParentCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"ctx"},
	Short:   "Manage project context (learnings, decisions, notes)",
	Long: `Manage and inspect the project context in Tasuku.

Subcommands:
  show      Dump the complete project context for agent consumption
  validate  Validate Tasuku storage for correctness
  schema    Output JSON Schema for Tasuku files

Examples:
  tk context show              # Output full context as JSON
  tk context validate          # Validate Tasuku storage
  tk context schema            # Show JSON schema`,
}

var contextShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Dump the complete project context for agent consumption",
	Long: `Output the entire Tasuku storage contents as structured data.

This command is designed for AI agents that need the full project context,
including all tasks, learnings, decisions, and notes.

Output includes:
  - version: Schema version number
  - tasks: All tasks with their statuses, priorities, dependencies
  - context.learnings: Insights discovered during work
  - context.decisions: Documented architectural choices
  - context.notes: Notes attached to tasks

The output format defaults to JSON but can be changed to YAML.

Examples:
  tk context show              # Output as JSON
  tk context show -f yaml      # Output as YAML
  tk context show | jq '.tasks'  # Pipe to jq for processing`,
	RunE: runContextShow,
}

var contextValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Tasuku storage",
	Long: `Validate the Tasuku storage for correctness.

Checks performed:
- Version is supported
- All tasks have non-empty descriptions
- All tasks have valid statuses
- No circular dependencies in blocked_by relationships

Examples:
  tk context validate`,
	RunE: runContextValidate,
}

var contextSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Output JSON Schema for Tasuku task files",
	Long: `Output the JSON Schema definition for Tasuku task files.

The schema defines:
  - version: Schema version (integer)
  - tasks: Object mapping task IDs to task objects
    - status: ready, in_progress, blocked, or done
    - description: Task description (string)
    - priority: 0-4 (0=critical, 4=backlog)
    - blocked_by: Array of task IDs this task depends on
    - owner: Optional owner identifier
    - created_at/updated_at: ISO 8601 timestamps
  - context: Shared knowledge object
    - learnings: Array of insight strings
    - decisions: Array of decision objects (chose/over/because)
    - notes: Object mapping task IDs to note arrays

Use Cases:
  - IDE validation: Configure your editor to validate Tasuku files
  - Documentation: Reference for file format
  - Tooling: Build tools that work with Tasuku files

Examples:
  tk context schema                   # Print schema to stdout
  tk context schema > tasuku.schema.json  # Save schema to file`,
	RunE: runContextSchema,
}

// Shared implementation for context show
func runContextShow(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
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
}

// Shared implementation for context validate
func runContextValidate(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if f.Version < 1 || f.Version > 3 {
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
}

// Shared implementation for context schema
func runContextSchema(cmd *cobra.Command, args []string) error {
	schema := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/iheanyi/tasuku/schema.json",
  "title": "Tasuku File",
  "description": "Schema for Tasuku task management storage (V3 directory format)",
  "type": "object",
  "required": ["version", "tasks", "context"],
  "properties": {
    "version": { "type": "integer", "enum": [1, 2, 3] },
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
          "claimed_at": { "type": ["string", "null"], "format": "date-time" },
          "parent_id": { "type": ["string", "null"] },
          "tags": { "type": "array", "items": { "type": "string" } },
          "fields": { "type": "object", "additionalProperties": { "type": "string" } },
          "timer_start": { "type": ["string", "null"], "format": "date-time" },
          "duration": { "type": "integer", "minimum": 0, "description": "Duration in nanoseconds" },
          "notes": {
            "type": "array",
            "items": {
              "type": "object",
              "required": ["id", "text", "created_at"],
              "properties": {
                "id": { "type": "string" },
                "text": { "type": "string" },
                "created_at": { "type": "string", "format": "date-time" }
              }
            }
          },
          "created_at": { "type": "string", "format": "date-time" },
          "updated_at": { "type": "string", "format": "date-time" }
        }
      }
    },
    "context": {
      "type": "object",
      "required": ["learnings", "decisions"],
      "properties": {
        "learnings": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["id", "text", "created_at"],
            "properties": {
              "id": { "type": "string" },
              "text": { "type": "string" },
              "is_rule": { "type": "boolean" },
              "created_at": { "type": "string", "format": "date-time" }
            }
          }
        },
        "decisions": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["id", "chose", "over", "because", "created_at"],
            "properties": {
              "id": { "type": "string" },
              "chose": { "type": "string" },
              "over": { "type": "array", "items": { "type": "string" } },
              "because": { "type": "string" },
              "created_at": { "type": "string", "format": "date-time" }
            }
          }
        }
      }
    },
    "archive": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "required": ["original_task", "archived_at"],
        "properties": {
          "original_task": { "$ref": "#/properties/tasks/additionalProperties" },
          "summary": { "type": "string" },
          "archived_at": { "type": "string", "format": "date-time" }
        }
      }
    }
  }
}`
	fmt.Println(schema)
	return nil
}

// =============================================================================
// Deprecated Context Command (kept for backward compatibility)
// =============================================================================

var contextCmd = &cobra.Command{
	Use:        "context",
	Hidden:     true,
	Deprecated: "use 'tk context show' instead",
	Short:      "Dump the complete project context for agent consumption",
	Long: `Output the entire Tasuku storage contents as structured data.

This command is designed for AI agents that need the full project context,
including all tasks, learnings, decisions, and notes.

Output includes:
  - version: Schema version number
  - tasks: All tasks with their statuses, priorities, dependencies
  - context.learnings: Insights discovered during work
  - context.decisions: Documented architectural choices
  - context.notes: Notes attached to tasks

The output format defaults to JSON but can be changed to YAML.

Examples:
  tk context                   # Output as JSON
  tk context -f yaml           # Output as YAML
  tk context | jq '.tasks'     # Pipe to jq for processing`,
	RunE: runContextShow,
}

// =============================================================================
// Server Parent Command and Subcommands (noun-verb pattern)
// =============================================================================

var serverCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"srv"},
	Short:   "Manage Tasuku server",
	Long: `Manage the Tasuku server for AI tool integration or HTTP API access.

Subcommands:
  start     Start the MCP or HTTP server
  mcp       Manage MCP server configuration

Examples:
  tk server start              # Start MCP server (stdio mode)
  tk server start --http :3000 # Start HTTP server
  tk server mcp install        # Auto-configure MCP in AI tools`,
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Tasuku server (MCP or HTTP)",
	Long: `Start a server for AI tool integration or HTTP API access.

Server Modes:

  MCP stdio (default):
    Used by Claude Code and other MCP-compatible AI tools.
    Communicates via stdin/stdout using the MCP protocol.
    This is the mode used when configured in AI tool settings.

  HTTP server (--http or --port):
    Runs a REST API server for programmatic access.
    Useful for integration with other tools or custom scripts.
    Includes a web dashboard at the root URL (/).

MCP Tools Exposed:
  tk_list, tk_add, tk_start, tk_done, tk_block, tk_unblock,
  tk_learn, tk_decide, tk_context, and more.

Examples:
  tk server start                     # Start MCP server (stdio mode)
  tk server start --http :3000        # Start HTTP server on port 3000
  tk server start --http localhost:8080  # HTTP on specific address

Web Dashboard:
  When running in HTTP mode, open the root URL in your browser
  (e.g., http://localhost:3000) to view the interactive dashboard.
  The dashboard supports:
  - Real-time task status with HTMX
  - Click to start/done/archive tasks
  - Filter by status
  - Progress visualization

See also:
  tk mcp install               # Auto-configure MCP in your AI tools
  tk mcp config                # Show MCP configuration JSON`,
	RunE: runServerStart,
}

func init() {
	serverStartCmd.Flags().Int("port", 0, "HTTP port (deprecated, use --http)")
	serverStartCmd.Flags().String("http", "", "HTTP address (e.g., :3000 or localhost:8080)")
}

// Shared implementation for server start
func runServerStart(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	httpAddr, _ := cmd.Flags().GetString("http")

	// Use FindUp to locate .tasuku.json - enables MCP to work from any subdirectory
	s := store.DefaultStorageWithWarning()

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
}

// =============================================================================
// Deprecated serve Command (kept for backward compatibility)
// =============================================================================

var serveCmd = &cobra.Command{
	Use:        "serve",
	Hidden:     true,
	Deprecated: "use 'tk server start' instead",
	Short:      "Start the Tasuku server (MCP or HTTP)",
	Long: `Start a server for AI tool integration or HTTP API access.

Server Modes:

  MCP stdio (default):
    Used by Claude Code and other MCP-compatible AI tools.
    Communicates via stdin/stdout using the MCP protocol.
    This is the mode used when configured in AI tool settings.

  HTTP server (--http or --port):
    Runs a REST API server for programmatic access.
    Useful for integration with other tools or custom scripts.
    Includes a web dashboard at the root URL (/).

MCP Tools Exposed:
  tk_list, tk_add, tk_start, tk_done, tk_block, tk_unblock,
  tk_learn, tk_decide, tk_context, and more.

Examples:
  tk serve                     # Start MCP server (stdio mode)
  tk serve --http :3000        # Start HTTP server on port 3000
  tk serve --http localhost:8080  # HTTP on specific address
  tk serve --port 3000         # HTTP server (deprecated, use --http)

Web Dashboard:
  When running in HTTP mode, open the root URL in your browser
  (e.g., http://localhost:3000) to view the interactive dashboard.

See also:
  tk mcp install               # Auto-configure MCP in your AI tools
  tk mcp config                # Show MCP configuration JSON`,
	RunE: runServerStart,
}

func init() {
	serveCmd.Flags().Int("port", 0, "HTTP port (deprecated, use --http)")
	serveCmd.Flags().String("http", "", "HTTP address (e.g., :3000 or localhost:8080)")
}

// =============================================================================
// Migration Commands
// =============================================================================

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import tasks from another task management system",
	Long: `Migrate tasks from another task management system into Tasuku.

Available subcommands:
  beads    Import from Beads (.beads/issues.jsonl)

Run 'tk migrate <subcommand> --help' for details on each source.`,
}

var migrateBeadsCmd = &cobra.Command{
	Use:   "beads",
	Short: "Import tasks from Beads issue tracker",
	Long: `Migrate tasks from a Beads (.beads/issues.jsonl) directory to Tasuku.

This will:
  - Import all issues as tasks
  - Map Beads statuses to Tasuku statuses
  - Preserve priority and dependencies
  - Import descriptions and notes
  - Keep the original .beads/ directory intact

Use --dry-run to preview what would be imported without making changes.

Examples:
  tk migrate beads             # Import from Beads
  tk migrate beads --dry-run   # Preview migration without changes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return migrateFromBeads(dryRun)
	},
}

var migrateV3Cmd = &cobra.Command{
	Use:   "v3",
	Short: "Migrate to V3 directory-based storage format",
	Long: `Migrate from the single .tasuku.json file to the V3 directory-based format.

The V3 format uses a .tasuku/ directory with one file per task:
  .tasuku/
  ├── config.json       # Version and settings
  ├── tasks/            # One JSON file per task
  │   ├── task-1.json
  │   └── task-2.json
  ├── archive/          # Archived tasks
  └── context/          # Learnings, decisions, notes

Benefits of V3 format:
  - No merge conflicts when multiple agents work in parallel
  - Each task can be edited independently
  - Cleaner git history (changes show which task was modified)
  - Archive/restore is just moving files

The original .tasuku.json will be renamed to .tasuku.json.bak.

Use --dry-run to preview what would be migrated without making changes.

Examples:
  tk migrate v3            # Migrate to V3 format
  tk migrate v3 --dry-run  # Preview migration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return migrateToV3(dryRun)
	},
}

func migrateToV3(dryRun bool) error {
	// Check for existing .tasuku.json (use migration-specific function)
	oldStore := store.GetV2StoreForMigration()
	if oldStore == nil {
		return fmt.Errorf("no .tasuku.json found to migrate")
	}

	oldPath := oldStore.Path()
	newPath := filepath.Join(filepath.Dir(oldPath), ".tasuku")

	// Check if already migrated
	if info, err := os.Stat(newPath); err == nil && info.IsDir() {
		return fmt.Errorf(".tasuku/ directory already exists - already migrated?")
	}

	// Read old format
	f, err := oldStore.Read()
	if err != nil {
		return fmt.Errorf("failed to read .tasuku.json: %w", err)
	}

	fmt.Println("V3 Migration Preview")
	fmt.Println("====================")
	fmt.Printf("Source: %s\n", oldPath)
	fmt.Printf("Target: %s/\n", newPath)
	fmt.Println()

	fmt.Printf("Tasks to migrate: %d\n", len(f.Tasks))
	fmt.Printf("Archived tasks: %d\n", len(f.Archive))
	fmt.Printf("Learnings: %d\n", len(f.Context.Learnings))
	fmt.Printf("Decisions: %d\n", len(f.Context.Decisions))

	noteCount := 0
	for _, notes := range f.Context.Notes {
		noteCount += len(notes)
	}
	fmt.Printf("Notes: %d\n", noteCount)
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run - no changes made.")
		fmt.Println("Run without --dry-run to perform migration.")
		return nil
	}

	// Create new directory store
	newStore := store.NewDirStore(newPath)
	if err := newStore.Init(); err != nil {
		return fmt.Errorf("failed to create .tasuku/ directory: %w", err)
	}

	// Migrate tasks
	for id, t := range f.Tasks {
		if err := newStore.AddTaskWithTags(id, t.Description, t.Priority, t.Tags); err != nil {
			return fmt.Errorf("failed to migrate task %s: %w", id, err)
		}
		// Update additional fields
		newStore.Update(func(nf *task.File) error {
			nt := nf.Tasks[id]
			nt.Status = t.Status
			nt.BlockedBy = t.BlockedBy
			nt.Owner = t.Owner
			nt.ClaimedAt = t.ClaimedAt
			nt.Fields = t.Fields
			nt.TimerStart = t.TimerStart
			nt.Duration = t.Duration
			nt.ParentID = t.ParentID
			nt.CreatedAt = t.CreatedAt
			nt.UpdatedAt = t.UpdatedAt
			nf.Tasks[id] = nt
			return nil
		})
		fmt.Printf("  ✓ Task: %s\n", id)
	}

	// Migrate learnings
	for _, l := range f.Context.Learnings {
		newStore.AddLearningWithRule(l.Text, &l.IsRule)
	}
	if len(f.Context.Learnings) > 0 {
		fmt.Printf("  ✓ Learnings: %d\n", len(f.Context.Learnings))
	}

	// Migrate decisions
	for _, d := range f.Context.Decisions {
		newStore.AddDecision(d)
	}
	if len(f.Context.Decisions) > 0 {
		fmt.Printf("  ✓ Decisions: %d\n", len(f.Context.Decisions))
	}

	// Migrate notes
	for taskID, notes := range f.Context.Notes {
		for _, note := range notes {
			newStore.AddNote(taskID, note.Text)
		}
	}
	if noteCount > 0 {
		fmt.Printf("  ✓ Notes: %d\n", noteCount)
	}

	// Migrate archive
	for id, archived := range f.Archive {
		// Create task, set status to done, then archive
		newStore.AddTask(id, archived.Description)
		newStore.Update(func(nf *task.File) error {
			t := nf.Tasks[id]
			t.Status = task.StatusDone
			t.Priority = archived.Priority
			t.Tags = archived.Tags
			t.Fields = archived.Fields
			t.Duration = archived.Duration
			t.CreatedAt = archived.CreatedAt
			t.UpdatedAt = archived.UpdatedAt
			nf.Tasks[id] = t
			return nil
		})
		newStore.ArchiveTask(id, archived.Summary)
		fmt.Printf("  ✓ Archived: %s\n", id)
	}

	// Backup old file
	backupPath := oldPath + ".bak"
	if err := os.Rename(oldPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup old file: %w", err)
	}

	fmt.Println()
	fmt.Println("Migration complete!")
	fmt.Printf("  Old file backed up to: %s\n", backupPath)
	fmt.Printf("  New format at: %s/\n", newPath)
	fmt.Println()
	fmt.Println("You can safely delete the backup file after verifying the migration.")

	return nil
}

func init() {
	migrateBeadsCmd.Flags().Bool("dry-run", false, "Preview migration without making changes")
	migrateV3Cmd.Flags().Bool("dry-run", false, "Preview migration without making changes")
	migrateCmd.AddCommand(migrateBeadsCmd)
	migrateCmd.AddCommand(migrateV3Cmd)
}

// =============================================================================
// Hooks Commands (noun-verb pattern)
// =============================================================================

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage git hooks and AI integration hooks",
	Long: `Manage hooks for git and AI tool integration with Tasuku.

Git Hook Subcommands:
  install    Install pre-commit and post-commit hooks
  uninstall  Remove Tasuku hooks (preserves other hook content)

AI Integration Subcommands:
  session    Display Tasuku context summary at session start
  sync       Sync tasks from TodoWrite JSON input (uses nudge rule)

The git hooks provide:
  - pre-commit: Validates Tasuku storage before commits
  - post-commit: Suggests task status updates based on commit messages

The sync command applies the nudge rule: only project-level tasks are synced,
session-level implementation steps stay in TodoWrite only.

Run 'tk hooks <subcommand> --help' for more details.`,
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Tasuku git hooks",
	Long: `Install Tasuku git hooks (pre-commit and post-commit).

This will add Tasuku integration to your git hooks while preserving
any existing hook content. The hooks are marked with special comments
so they can be safely removed later.

Hooks installed:
  - pre-commit: Validates Tasuku storage before allowing commits
  - post-commit: Detects task references in commit messages and suggests updates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return installHooks()
	},
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Tasuku git hooks",
	Long: `Remove Tasuku git hooks while preserving other hook content.

This only removes the Tasuku-specific sections from your git hooks.
Any other hook content (from other tools) will be preserved.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return uninstallHooks()
	},
}

var hooksSessionCmd = &cobra.Command{
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

var hooksSyncCmd = &cobra.Command{
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

// =============================================================================
// Deprecated hook Command (kept for backward compatibility)
// =============================================================================

var hookCmd = &cobra.Command{
	Use:        "hook",
	Hidden:     true,
	Deprecated: "use 'tk hooks session' or 'tk hooks sync' instead",
	Short:      "Run Claude Code integration hooks",
	Long: `Run internal hooks for Claude Code integration.

Available subcommands:
  session    Display Tasuku context summary at session start
  sync       Sync tasks from TodoWrite JSON input

These are typically called automatically by Claude Code integration,
but can be run manually for debugging.`,
}

var hookSessionCmd = &cobra.Command{
	Use:        "session",
	Hidden:     true,
	Deprecated: "use 'tk hooks session' instead",
	Short:      "Display Tasuku context summary",
	Long: `Display a summary of Tasuku context for Claude Code session start.

Shows:
  - Task counts by status
  - Number of learnings and decisions
  - Suggested next task based on priority`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookSession()
	},
}

var hookSyncCmd = &cobra.Command{
	Use:        "sync",
	Hidden:     true,
	Deprecated: "use 'tk hooks sync' instead",
	Short:      "Sync tasks from TodoWrite JSON",
	Long: `Sync tasks from Claude Code's TodoWrite tool.

Reads JSON from stdin in TodoWrite format and applies the nudge rule.
Only project-level tasks are synced to Tasuku.

Examples:
  tk hook sync < todos.json          # Sync from file
  echo '[...]' | tk hook sync        # Sync from piped JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return hookSync()
	},
}

func init() {
	hookCmd.AddCommand(hookSessionCmd)
	hookCmd.AddCommand(hookSyncCmd)
}

// =============================================================================
// Doctor Command
// =============================================================================

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Tasuku setup and MCP configuration",
	Long: `Check your Tasuku installation and MCP configuration for common issues.

This command verifies:
  - tk binary is accessible and shows its location
  - Tasuku storage exists (.tasuku/ directory or .tasuku.json file)
  - MCP is configured in Claude Code, Cursor, and other AI tools
  - The configured binary path matches the current tk installation
  - The MCP server can start and respond to requests

Run this when Tasuku tools aren't appearing in your AI assistant.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	fmt.Println("Tasuku Doctor")
	fmt.Println("=============")
	fmt.Println()

	hasErrors := false

	// 1. Check tk binary
	executable, err := os.Executable()
	if err != nil {
		fmt.Println("✗ Could not determine tk binary location")
		hasErrors = true
	} else {
		fmt.Printf("✓ tk binary: %s\n", executable)
	}

	// 2. Check Tasuku storage
	s := store.DefaultStorageWithWarning()
	tasukuPath := s.Path()
	if !s.Exists() {
		fmt.Printf("✗ No Tasuku storage found (searched from %s)\n", mustGetwd())
		fmt.Println("  Run 'tk init' to create .tasuku/ directory")
		hasErrors = true
	} else {
		fmt.Printf("✓ Tasuku storage: %s\n", tasukuPath)
	}

	fmt.Println()
	fmt.Println("MCP Configuration")
	fmt.Println("-----------------")

	// 3. Check AI tool configurations
	tools := getSupportedAITools()
	configuredTools := 0
	mismatchedPaths := []string{}

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
			continue
		}

		mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
		if !ok {
			continue
		}

		tasukuConfig, exists := mcpServers["tasuku"].(map[string]interface{})
		if !exists {
			fmt.Printf("✗ %s: MCP not configured\n", tool.Name)
			fmt.Printf("  Run 'tk mcp install' to configure\n")
			hasErrors = true
			continue
		}

		configuredPath, _ := tasukuConfig["command"].(string)
		configuredArgs, _ := tasukuConfig["args"].([]interface{})

		// Check if path matches current executable
		if configuredPath != executable {
			fmt.Printf("⚠ %s: configured but path mismatch\n", tool.Name)
			fmt.Printf("  Configured: %s\n", configuredPath)
			fmt.Printf("  Current:    %s\n", executable)
			mismatchedPaths = append(mismatchedPaths, tool.Name)
		} else {
			argsStr := ""
			for _, arg := range configuredArgs {
				if s, ok := arg.(string); ok {
					argsStr += s + " "
				}
			}
			fmt.Printf("✓ %s: configured (%s %s)\n", tool.Name, filepath.Base(configuredPath), strings.TrimSpace(argsStr))
		}
		configuredTools++
	}

	if configuredTools == 0 {
		fmt.Println("✗ No AI tools have Tasuku MCP configured")
		fmt.Println("  Run 'tk mcp install' to auto-configure")
		hasErrors = true
	}

	// 4. Test MCP server
	fmt.Println()
	fmt.Println("MCP Server Test")
	fmt.Println("---------------")

	// Quick test: can we create a server and get tools?
	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		tools := mcpServer.Tools()
		fmt.Printf("✓ MCP server responds with %d tools\n", len(tools))

		// List a few tools
		if len(tools) > 0 {
			toolNames := []string{}
			for _, t := range tools {
				toolNames = append(toolNames, t.Name)
			}
			if len(toolNames) > 5 {
				toolNames = toolNames[:5]
				fmt.Printf("  Tools: %s, ... (+%d more)\n", strings.Join(toolNames, ", "), len(tools)-5)
			} else {
				fmt.Printf("  Tools: %s\n", strings.Join(toolNames, ", "))
			}
		}
	} else {
		fmt.Println("⚠ Cannot test MCP server (no Tasuku storage)")
	}

	// 5. Check CLI/MCP parity
	fmt.Println()
	fmt.Println("CLI/MCP Parity")
	fmt.Println("--------------")

	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		mcpTools := mcpServer.Tools()

		// Build set of MCP tool names
		mcpToolSet := make(map[string]bool)
		for _, t := range mcpTools {
			mcpToolSet[t.Name] = true
		}

		// Define expected MCP tools for CLI commands
		// Maps CLI command path to expected MCP tool(s)
		cliToMCP := map[string][]string{
			"task list":     {"tk_list"},
			"task add":      {"tk_add"},
			"task show":     {"tk_show"},
			"task start":    {"tk_start"},
			"task done":     {"tk_done"},
			"task block":    {"tk_block"},
			"task unblock":  {"tk_unblock"},
			"task pause":    {"tk_pause"},
			"task find":     {"tk_find"},
			"task priority": {"tk_priority"},
			"task delete":   {"tk_delete"},
			"task edit":     {"tk_edit"},
			"task owner":    {"tk_owner"},
			"task claim":    {"tk_claim"},
			"task release":  {"tk_release"},
			"task tag":      {"tk_tag_add", "tk_tag_remove"},
			"task field":    {"tk_field_set", "tk_field_remove"},
			"task timer":    {"tk_timer_start", "tk_timer_stop", "tk_timer_status"},
			"task archive":  {"tk_archive", "tk_archive_restore", "tk_archive_list"},
			"context learn": {"tk_learn"},
			"context decide":{"tk_decide"},
			"context note":  {"tk_note"},
			"context show":  {"tk_context"},
		}

		missingTools := []string{}
		for cli, expectedTools := range cliToMCP {
			for _, tool := range expectedTools {
				if !mcpToolSet[tool] {
					missingTools = append(missingTools, fmt.Sprintf("%s (missing %s)", cli, tool))
				}
			}
		}

		if len(missingTools) == 0 {
			fmt.Printf("✓ All %d CLI commands have corresponding MCP tools\n", len(cliToMCP))
		} else {
			fmt.Printf("✗ %d CLI commands missing MCP tools:\n", len(missingTools))
			for _, m := range missingTools {
				fmt.Printf("  - %s\n", m)
			}
			hasErrors = true
		}
	} else {
		fmt.Println("⚠ Cannot check parity (no Tasuku storage)")
	}

	// Summary
	fmt.Println()
	if hasErrors {
		fmt.Println("Issues found. See recommendations above.")
		if configuredTools > 0 {
			fmt.Println()
			fmt.Println("If MCP is configured but tools aren't visible:")
			fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
			fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")
		}
		return nil
	}

	if len(mismatchedPaths) > 0 {
		fmt.Println("Configuration path mismatch detected.")
		fmt.Println("Run 'tk mcp install' to update configuration.")
		return nil
	}

	fmt.Println("Everything looks good!")
	fmt.Println()
	fmt.Println("If tools still aren't visible in your AI assistant:")
	fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
	fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")

	return nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// =============================================================================
// MCP Commands
// =============================================================================

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server for AI tool integration",
	Long: `Model Context Protocol (MCP) server for AI tool integration.

Available subcommands:
  serve      Start the MCP server (stdio mode for AI tools)
  install    Auto-configure Tasuku MCP in Claude Code, Cursor, etc.
  uninstall  Remove Tasuku MCP configuration from AI tools
  config     Display MCP configuration JSON for manual setup

The MCP server enables AI tools like Claude Code and Cursor to
interact with Tasuku directly, allowing them to list, create,
and update tasks.

Examples:
  tk mcp install    # Auto-configure in Claude Code/Cursor
  tk mcp serve      # Start MCP server (used by AI tools)
  tk mcp config     # Show config for manual setup`,
}

var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Auto-configure MCP in AI tools",
	Long: `Automatically configure the Tasuku MCP server in supported AI tools.

Supported tools:
  - Claude Code (~/.claude/settings.json)
  - Cursor (~/.cursor/mcp.json)

The configuration will be added to existing settings without
overwriting other MCP servers or configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcpInstall()
	},
}

var mcpUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove MCP configuration from AI tools",
	Long: `Remove the Tasuku MCP server configuration from AI tools.

This removes only the Tasuku configuration; other MCP servers
will be preserved.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcpUninstall()
	},
}

var mcpConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show MCP configuration snippet",
	Long: `Display the MCP configuration snippet for manual setup.

Use this if automatic installation doesn't work or you want
to configure MCP manually in your AI tool settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcpConfig()
	},
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the MCP (Model Context Protocol) server in stdio mode.

This is the mode used by AI tools like Claude Code and Cursor.
The server communicates via stdin/stdout using the MCP protocol.

You typically don't run this directly - instead use 'tk mcp install'
to configure your AI tool to run it automatically.

MCP Tools Exposed:
  tk_list, tk_add, tk_start, tk_done, tk_block, tk_unblock,
  tk_learn, tk_decide, tk_context, and more.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()
		mcpServer := mcp.New(s)
		return mcpServer.Run()
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpInstallCmd)
	mcpCmd.AddCommand(mcpUninstallCmd)
	mcpCmd.AddCommand(mcpConfigCmd)
}

// =============================================================================
// Deprecated Utility Commands (kept for backward compatibility)
// =============================================================================

var validateCmd = &cobra.Command{
	Use:        "validate",
	Hidden:     true,
	Deprecated: "use 'tk context validate' instead",
	Short:      "Validate Tasuku storage",
	Long: `Validate the Tasuku storage for correctness.

Checks performed:
- Version is supported
- All tasks have non-empty descriptions
- All tasks have valid statuses
- No circular dependencies in blocked_by relationships

Examples:
  tk validate`,
	RunE: runContextValidate,
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
	Use:        "schema",
	Hidden:     true,
	Deprecated: "use 'tk context schema' instead",
	Short:      "Output JSON Schema for Tasuku task files",
	Long: `Output the JSON Schema definition for Tasuku task files.

The schema defines:
  - version: Schema version (integer)
  - tasks: Object mapping task IDs to task objects
    - status: ready, in_progress, blocked, or done
    - description: Task description (string)
    - priority: 0-4 (0=critical, 4=backlog)
    - blocked_by: Array of task IDs this task depends on
    - owner: Optional owner identifier
    - created_at/updated_at: ISO 8601 timestamps
  - context: Shared knowledge object
    - learnings: Array of insight strings
    - decisions: Array of decision objects (chose/over/because)
    - notes: Object mapping task IDs to note arrays

Use Cases:
  - IDE validation: Configure your editor to validate Tasuku files
  - Documentation: Reference for file format
  - Tooling: Build tools that work with Tasuku files

Examples:
  tk schema                      # Print schema to stdout
  tk schema > tasuku.schema.json # Save schema to file`,
	RunE: runContextSchema,
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

// outputTasksTree displays tasks in a hierarchical tree format,
// with subtasks indented under their parent tasks.
func outputTasksTree(tasks []taskEntry) error {
	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(tasks, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(tasks)
		fmt.Print(string(data))
	default: // table with tree structure
		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		// Build parent->children map
		children := make(map[string][]taskEntry)
		roots := []taskEntry{}

		for _, t := range tasks {
			parentID := t.Task.GetParentID()
			if parentID == "" {
				roots = append(roots, t)
			} else {
				children[parentID] = append(children[parentID], t)
			}
		}

		// Print tree recursively
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		var printTree func(entries []taskEntry, indent string)
		printTree = func(entries []taskEntry, indent string) {
			for i, t := range entries {
				statusIcon := statusToIcon(t.Task.Status)
				desc := t.Task.Description
				if len(desc) > 45 {
					desc = desc[:42] + "..."
				}

				// Tree branch characters
				prefix := indent
				if indent != "" {
					if i == len(entries)-1 {
						prefix = indent[:len(indent)-3] + "└─ "
					} else {
						prefix = indent[:len(indent)-3] + "├─ "
					}
				}

				blocked := ""
				if len(t.Task.BlockedBy) > 0 {
					blocked = fmt.Sprintf("(blocked by: %s)", strings.Join(t.Task.BlockedBy, ", "))
				}
				fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\n", prefix, statusIcon, t.ID, desc, blocked)

				// Print children
				if childTasks, ok := children[t.ID]; ok {
					nextIndent := indent
					if indent == "" {
						nextIndent = "   "
					} else if i == len(entries)-1 {
						nextIndent = indent[:len(indent)-3] + "   "
					} else {
						nextIndent = indent[:len(indent)-3] + "│  "
					}
					printTree(childTasks, nextIndent)
				}
			}
		}

		printTree(roots, "")
		w.Flush()
	}
	return nil
}

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

func outputTaskDetail(id string, t task.Task, notes []task.Note, allTasks map[string]task.Task) error {
	// Find which tasks this task blocks (reverse lookup)
	blocks := findBlockedTasks(id, allTasks)

	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(map[string]interface{}{
			"id":     id,
			"task":   t,
			"notes":  notes,
			"blocks": blocks,
		}, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(map[string]interface{}{
			"id":     id,
			"task":   t,
			"notes":  notes,
			"blocks": blocks,
		})
		fmt.Print(string(data))
	default: // table
		fmt.Printf("ID:          %s\n", id)
		fmt.Printf("Status:      %s\n", t.Status)
		fmt.Printf("Description: %s\n", t.Description)
		if t.Priority != nil {
			fmt.Printf("Priority:    %s\n", task.PriorityName(*t.Priority))
		}
		if len(t.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(t.Tags, ", "))
		}
		if len(t.Fields) > 0 {
			fmt.Printf("Fields:\n")
			// Sort keys for consistent output
			keys := make([]string, 0, len(t.Fields))
			for k := range t.Fields {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("  %s: %s\n", k, t.Fields[k])
			}
		}
		if len(t.BlockedBy) > 0 {
			fmt.Printf("Blocked by:  %s\n", strings.Join(t.BlockedBy, ", "))
		}
		if len(blocks) > 0 {
			fmt.Printf("Blocks:      %s\n", strings.Join(blocks, ", "))
		}
		if t.Owner != nil {
			ownerStr := *t.Owner
			if t.ClaimedAt != nil {
				ownerStr += fmt.Sprintf(" (claimed %s)", formatRelativeTime(*t.ClaimedAt))
			}
			fmt.Printf("Owner:       %s\n", ownerStr)
		}
		// Time tracking
		if t.TimerStart != nil || t.Duration > 0 {
			if t.TimerStart != nil {
				elapsed := time.Since(*t.TimerStart)
				fmt.Printf("Timer:       RUNNING (%s)\n", formatDuration(elapsed))
			}
			totalDuration := t.CurrentDuration()
			if totalDuration > 0 {
				fmt.Printf("Duration:    %s\n", formatDuration(totalDuration))
			}
		}
		fmt.Printf("Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))

		if len(notes) > 0 {
			fmt.Printf("\nNotes:\n")
			for _, note := range notes {
				timeStr := formatRelativeTime(note.CreatedAt)
				fmt.Printf("  - %s (%s)\n", note.Text, timeStr)
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
				fmt.Printf("  [learning %s] %s\n", r.ID, r.Content)
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

// formatRelativeTime formats a time as relative (e.g., "2 hours ago") for recent times
// or as a date (e.g., "2024-01-04") for older times.
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

// formatAge returns a human-readable age string, or empty string for zero time.
func formatAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return formatRelativeTime(t)
}

// generateID creates a kebab-case ID from description.
// generateID is deprecated - use task.GenerateTaskID instead.
// Kept for backward compatibility with tests.
func generateID(desc string) string {
	return task.GenerateTaskID(desc)
}

// =============================================================================
// Hook Implementations
// =============================================================================

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

	s := store.DefaultStorageWithWarning()
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
				f.Context.Notes = make(map[string][]task.Note)
			}
			if issue.Description != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], task.Note{Text: "Description: " + issue.Description, CreatedAt: createdAt})
			}
			if issue.Notes != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], task.Note{Text: issue.Notes, CreatedAt: createdAt})
			}
			if issue.CloseReason != "" {
				f.Context.Notes[id] = append(f.Context.Notes[id], task.Note{Text: "Close reason: " + issue.CloseReason, CreatedAt: updatedAt})
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

func installHooks() error {
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
			"args":    []string{"server", "start"},
			"type":    "stdio",
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	fmt.Println("Add this to ~/.claude.json under 'mcpServers':")
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
		{"Claude Code", home + "/.claude.json", "mcpServers"},
		{"Cursor", home + "/.cursor/mcp.json", "mcpServers"},
		{"Cursor (alt)", home + "/Library/Application Support/Cursor/User/globalStorage/mcp.json", "mcpServers"},
	}
}

func getClaudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Claude Code reads MCP servers from ~/.claude.json
	return home + "/.claude.json", nil
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
			"args":    []string{"server", "start"},
			"type":    "stdio",
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
					"args":    []string{"server", "start"},
					"type":    "stdio",
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

// =============================================================================
// Suggest Command - Agent Nudge Rule Helper
// =============================================================================

var suggestCmd = &cobra.Command{
	Use:   "suggest <description>",
	Short: "Suggest whether a task should be persisted to tk or kept session-only",
	Long: `Analyze a task description and suggest whether it should be:
- Persisted to tk (project-level: features, bugs, milestones)
- Kept as session-only (implementation steps like "fix type error")

This implements the "nudge rule" for AI agents: before adding items to TodoWrite,
call tk suggest to determine if they should also be tracked in tk.

Project-level indicators:
  - Keywords like "implement", "add feature", "fix bug", "refactor", "migrate"
  - Database, API, authentication, security work
  - Milestones, epics, stories

Session-level indicators:
  - Keywords like "fix type error", "update file", "run tests", "debug"
  - Small, temporary implementation steps

Examples:
  tk suggest "Implement user authentication"
    → Should persist: yes (project-level feature)

  tk suggest "Fix type error in auth.ts"
    → Should persist: no (session-level implementation step)

  tk suggest "Add dark mode support"
    → Should persist: yes (project-level feature)

  tk suggest "Update the imports in helper.go"
    → Should persist: no (session-level step)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSuggest(args[0])
	},
}

func runSuggest(description string) error {
	desc := strings.ToLower(description)

	// Keywords that indicate project-level tasks (should persist to tk)
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

	// Keywords that indicate session-level tasks (TodoWrite only)
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
	reason := "No strong project-level indicators found"
	matchedKeyword := ""

	// Check for project keywords
	for _, kw := range projectKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = true
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains project-level keyword '%s'", kw)
			break
		}
	}

	// Session keywords can override if they match
	for _, kw := range sessionKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = false
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains session-level keyword '%s'", kw)
			break
		}
	}

	// Output based on format
	switch outputFormat {
	case "json":
		result := map[string]interface{}{
			"should_persist":       shouldPersist,
			"reason":               reason,
			"matched_keyword":      matchedKeyword,
			"original_description": description,
		}
		if shouldPersist {
			result["suggested_command"] = fmt.Sprintf("tk task add %q", description)
			result["recommendation"] = "Add this to tk for persistent tracking across sessions"
		} else {
			result["recommendation"] = "Keep this in TodoWrite only - it's a session-level implementation step"
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))

	case "yaml":
		result := map[string]interface{}{
			"should_persist":       shouldPersist,
			"reason":               reason,
			"matched_keyword":      matchedKeyword,
			"original_description": description,
		}
		if shouldPersist {
			result["suggested_command"] = fmt.Sprintf("tk task add %q", description)
			result["recommendation"] = "Add this to tk for persistent tracking across sessions"
		} else {
			result["recommendation"] = "Keep this in TodoWrite only - it's a session-level implementation step"
		}
		data, _ := yaml.Marshal(result)
		fmt.Print(string(data))

	default:
		// Table/human-readable format
		if shouldPersist {
			fmt.Println("✓ PERSIST TO TK")
			fmt.Println()
			fmt.Printf("  Description: %s\n", description)
			fmt.Printf("  Reason: %s\n", reason)
			fmt.Println()
			fmt.Println("  Suggested command:")
			fmt.Printf("    tk task add %q\n", description)
		} else {
			fmt.Println("✗ KEEP SESSION-ONLY")
			fmt.Println()
			fmt.Printf("  Description: %s\n", description)
			fmt.Printf("  Reason: %s\n", reason)
			fmt.Println()
			fmt.Println("  Recommendation: Use TodoWrite for this implementation step.")
			fmt.Println("  It doesn't need to persist across sessions.")
		}
	}

	return nil
}
