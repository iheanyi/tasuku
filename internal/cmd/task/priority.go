package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

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
		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}

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
