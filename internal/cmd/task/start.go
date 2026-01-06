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

Use --timer to also start a time tracking timer on the task.

Examples:
  tk task start my-task             # Start working on "my-task"
  tk task start blocked --unblock   # Clear blockers and start
  tk task start my-task --timer     # Start task and begin timing`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			unblock, _ := cmd.Flags().GetBool("unblock")
			startTimer, _ := cmd.Flags().GetBool("timer")
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

			// Start timer if requested
			if startTimer {
				if err := s.StartTimer(taskID); err != nil {
					// Don't fail the whole command if timer fails to start
					fmt.Printf("Started: %s\n", taskID)
					fmt.Printf("  Warning: could not start timer: %v\n", err)
					return nil
				}
				fmt.Printf("Started: %s (timer running)\n", taskID)
			} else {
				fmt.Printf("Started: %s\n", taskID)
			}
			return nil
		},
	}

	cmd.Flags().Bool("unblock", false, "Clear blockers before starting (for blocked tasks)")
	cmd.Flags().Bool("timer", false, "Start a timer when beginning work")

	return cmd
}

var startCmd = newStartCmd()
