package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <task-id>",
		Short: "Restore an archived task to active tasks",
		Long: `Restore an archived task back to the active task list.

The restored task will have status "ready" and can be worked on again.

Examples:
  tk task restore auth-feature
  tk task restore my-old-task`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			s := store.DefaultStorageWithWarning()

			if err := s.RestoreTask(taskID); err != nil {
				return err
			}

			fmt.Printf("Restored task %s (status: ready)\n", taskID)
			return nil
		},
	}

	return cmd
}

var restoreCmd = newRestoreCmd()
