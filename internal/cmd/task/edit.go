package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

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
		s, err := store.DefaultStorageWithWarning()
		if err != nil {
			return err
		}

		if err := s.EditTask(taskID, newDesc); err != nil {
			return err
		}

		fmt.Printf("Updated: %s\n", taskID)
		return nil
	},
}
