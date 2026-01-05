package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <task-id>",
		Short: "Mark task as in_progress",
		Long: `Begin working on a task by setting its status to "in_progress".

This indicates that the task is actively being worked on.

Use --unblock to clear any blockers when starting a blocked task,
allowing you to begin work in a single command.

Examples:
  tk task start my-task             # Start working on "my-task"
  tk task start blocked --unblock   # Clear blockers and start`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			unblock, _ := cmd.Flags().GetBool("unblock")
			s := store.DefaultStorageWithWarning()

			// If unblock flag is set, clear blockers first
			if unblock {
				if err := s.UnblockTask(taskID); err != nil {
					return err
				}
			}

			if err := s.SetStatus(taskID, task.StatusInProgress); err != nil {
				return err
			}

			fmt.Printf("Started: %s\n", taskID)
			return nil
		},
	}

	cmd.Flags().Bool("unblock", false, "Clear blockers before starting (for blocked tasks)")

	return cmd
}

var startCmd = newStartCmd()
