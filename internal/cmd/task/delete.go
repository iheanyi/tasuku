package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <task-id> [task-id...]",
	Short: "Delete task(s)",
	Long: `Permanently delete one or more tasks from the project.

This action cannot be undone. Notes associated with the tasks are also deleted.

Examples:
  tk task delete my-task                  # Delete "my-task"
  tk task delete task-1 task-2 task-3     # Delete multiple tasks`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}

		for _, taskID := range args {
			if err := s.DeleteTask(taskID); err != nil {
				return fmt.Errorf("%s: %w", taskID, err)
			}
			fmt.Printf("Deleted: %s\n", taskID)
		}
		return nil
	},
}
