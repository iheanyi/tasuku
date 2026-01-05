package task

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

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
  tk task add "Implement user authentication"
  tk task add "Fix critical bug" -p 0               # Critical priority
  tk task add "Refactor database layer" --id db-refactor
  tk task add "Update documentation" --priority low
  tk task add "Add login page" --tag frontend --tag auth  # Multiple tags
  tk task add "Write unit tests" --parent feature-x       # Create subtask`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := args[0]
		id, _ := cmd.Flags().GetString("id")
		priority, _ := cmd.Flags().GetInt("priority")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		parentID, _ := cmd.Flags().GetString("parent")

		s := store.DefaultStorageWithWarning()

		// Generate ID if not provided, checking for collisions
		if id == "" {
			existingIDs := make(map[string]struct{})
			if f, err := s.Read(); err == nil {
				for taskID := range f.Tasks {
					existingIDs[taskID] = struct{}{}
				}
			}
			id = task.GenerateTaskID(description, existingIDs)
		}

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
