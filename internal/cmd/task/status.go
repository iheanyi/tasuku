package task

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

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
  tk task show my-task            # Show details for "my-task"
  tk task show fix-auth -f json   # Output as JSON`,
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

This indicates that the task is actively being worked on.

Examples:
  tk task start my-task           # Start working on "my-task"`,
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

Examples:
  tk task done my-task            # Mark "my-task" as complete`,
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

var pauseCmd = &cobra.Command{
	Use:   "pause <task-id>",
	Short: "Pause work on a task",
	Long: `Move a task from in_progress back to ready status.

Use this when you need to temporarily stop working on a task.

Examples:
  tk task pause my-task           # Pause work on "my-task"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

		if err := s.SetStatus(taskID, task.StatusReady); err != nil {
			return err
		}

		fmt.Printf("Paused: %s\n", taskID)
		return nil
	},
}

var blockCmd = &cobra.Command{
	Use:   "block <task-id>",
	Short: "Mark task as blocked",
	Long: `Mark a task as blocked by one or more other tasks.

Use the --by flag to specify which tasks are blocking.

Examples:
  tk task block my-task --by other-task
  tk task block my-task --by task-a,task-b`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		blockers, _ := cmd.Flags().GetStringSlice("by")
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
	Use:   "unblock <task-id>",
	Short: "Remove blockers from task",
	Long: `Remove blockers from a task and set status to ready.

By default, removes all blockers. Use --from to remove a specific blocker.

Examples:
  tk task unblock my-task              # Remove all blockers
  tk task unblock my-task --from task-a # Remove specific blocker`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		fromBlocker, _ := cmd.Flags().GetString("from")
		s := store.DefaultStorageWithWarning()

		if fromBlocker != "" {
			if err := s.RemoveBlocker(taskID, fromBlocker); err != nil {
				return err
			}
			fmt.Printf("Removed blocker %s from: %s\n", fromBlocker, taskID)
		} else {
			if err := s.UnblockTask(taskID); err != nil {
				return err
			}
			fmt.Printf("Unblocked: %s\n", taskID)
		}
		return nil
	},
}

func init() {
	unblockCmd.Flags().String("from", "", "Remove only this specific blocker")
}

var deleteCmd = &cobra.Command{
	Use:   "delete <task-id>",
	Short: "Delete a task",
	Long: `Permanently delete a task from the project.

This action cannot be undone. Notes associated with the task are also deleted.

Examples:
  tk task delete my-task          # Delete "my-task"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

		if err := s.DeleteTask(taskID); err != nil {
			return err
		}

		fmt.Printf("Deleted: %s\n", taskID)
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <task-id> <new-description>",
	Short: "Update task description",
	Long: `Update the description of an existing task.

Examples:
  tk task edit my-task "Updated description"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		newDesc := args[1]
		s := store.DefaultStorageWithWarning()

		if err := s.EditTask(taskID, newDesc); err != nil {
			return err
		}

		fmt.Printf("Updated: %s\n", taskID)
		return nil
	},
}

var ownerCmd = &cobra.Command{
	Use:   "owner <task-id> [owner-name]",
	Short: "Manage task ownership",
	Long: `Set, view, or clear the owner of a task.

Examples:
  tk task owner my-task agent-1      # Set owner to agent-1
  tk task owner my-task --clear      # Clear owner
  tk task owner my-task              # Show current owner`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		clearFlag, _ := cmd.Flags().GetBool("clear")
		s := store.DefaultStorageWithWarning()

		if len(args) == 2 {
			ownerName := args[1]
			if err := s.SetOwner(taskID, ownerName); err != nil {
				return err
			}
			fmt.Printf("Set owner of %s to: %s\n", taskID, ownerName)
			return nil
		}

		if clearFlag {
			if err := s.ClearOwner(taskID); err != nil {
				return err
			}
			fmt.Printf("Cleared owner of: %s\n", taskID)
			return nil
		}

		// Show current owner
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

var priorityCmd = &cobra.Command{
	Use:   "priority <task-id> <level>",
	Short: "Set task priority",
	Long: `Set the priority level of a task.

Priority Levels:
  0 or critical  - Urgent, needs immediate attention
  1 or high      - Important, should be done soon
  2 or normal    - Standard priority
  3 or low       - Can wait
  4 or backlog   - Future consideration

Examples:
  tk task priority my-task 0           # Set to critical
  tk task priority my-task high        # Set to high`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		levelStr := args[1]
		s := store.DefaultStorageWithWarning()

		level := task.ParsePriority(levelStr)
		if level < 0 || level > 4 {
			return fmt.Errorf("invalid priority: %s (use 0-4 or critical/high/normal/low/backlog)", levelStr)
		}

		if err := s.SetPriority(taskID, level); err != nil {
			return err
		}

		fmt.Printf("Set priority of %s to: %s\n", taskID, task.PriorityName(level))
		return nil
	},
}
