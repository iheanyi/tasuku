package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

func newUnblockCmd() *cobra.Command {
	cmd := &cobra.Command{
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
			s, err := store.DefaultStorageWithWarning()
			if err != nil {
				return err
			}

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

	cmd.Flags().String("from", "", "Remove only this specific blocker")

	return cmd
}

var unblockCmd = newUnblockCmd()
