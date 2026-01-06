package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var pauseCmd = &cobra.Command{
	Use:   "pause <task-id> [task-id...]",
	Short: "Pause work on task(s)",
	Long: `Move one or more tasks from in_progress back to ready status.

Use this when you need to temporarily stop working on tasks.
If a timer is running on any task, it will be automatically stopped.

Examples:
  tk task pause my-task                   # Pause work on "my-task"
  tk task pause task-1 task-2             # Pause multiple tasks`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()

		for _, taskID := range args {
			// Auto-stop timer if running
			elapsed, wasRunning, err := s.StopTimerIfRunning(taskID)
			if err != nil {
				return fmt.Errorf("%s: %w", taskID, err)
			}

			if err := s.SetStatus(taskID, task.StatusReady); err != nil {
				return fmt.Errorf("%s: %w", taskID, err)
			}

			fmt.Printf("Paused: %s\n", taskID)
			if wasRunning {
				fmt.Printf("  Timer stopped: +%s\n", formatDuration(elapsed))
			}
		}
		return nil
	},
}
