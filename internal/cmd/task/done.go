package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
)

var doneCmd = &cobra.Command{
	Use:   "done <task-id> [task-id...]",
	Short: "Mark task(s) as done",
	Long: `Mark one or more tasks as completed.

If a timer is running on any task, it will be automatically stopped.
Any tasks blocked solely by the completed task(s) will be automatically unblocked.

Examples:
  tk task done my-task                    # Mark "my-task" as complete
  tk task done task-1 task-2 task-3       # Mark multiple tasks as complete`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()

		for _, taskID := range args {
			// Auto-stop timer if running
			elapsed, wasRunning, err := s.StopTimerIfRunning(taskID)
			if err != nil {
				return fmt.Errorf("%s: %w", taskID, err)
			}

			// Mark done and auto-unblock dependent tasks
			unblocked, err := s.MarkDoneAndUnblock(taskID)
			if err != nil {
				return fmt.Errorf("%s: %w", taskID, err)
			}

			fmt.Printf("Completed: %s\n", taskID)
			if wasRunning {
				fmt.Printf("  Timer stopped: +%s\n", formatDuration(elapsed))
			}
			if len(unblocked) > 0 {
				fmt.Printf("  Unblocked: %v\n", unblocked)
			}
		}
		return nil
	},
}
