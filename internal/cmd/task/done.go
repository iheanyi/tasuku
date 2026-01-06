package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var doneCmd = &cobra.Command{
	Use:   "done <task-id> [task-id...]",
	Short: "Mark task(s) as done",
	Long: `Mark one or more tasks as completed.

If a timer is running on any task, it will be automatically stopped.

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

			if err := s.SetStatus(taskID, task.StatusDone); err != nil {
				return fmt.Errorf("%s: %w", taskID, err)
			}

			fmt.Printf("Completed: %s\n", taskID)
			if wasRunning {
				fmt.Printf("  Timer stopped: +%s\n", formatDuration(elapsed))
			}
		}
		return nil
	},
}
