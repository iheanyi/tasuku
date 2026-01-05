package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

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
