package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

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
