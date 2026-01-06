package task

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var pauseCmd = &cobra.Command{
	Use:   "pause <task-id>",
	Short: "Pause work on a task",
	Long: `Move a task from in_progress back to ready status.

Use this when you need to temporarily stop working on a task.
If a timer is running on the task, it will be automatically stopped.

Examples:
  tk task pause my-task           # Pause work on "my-task"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.DefaultStorageWithWarning()

		// Auto-stop timer if running
		elapsed, wasRunning, err := s.StopTimerIfRunning(taskID)
		if err != nil {
			return err
		}

		if err := s.SetStatus(taskID, task.StatusReady); err != nil {
			return err
		}

		fmt.Printf("Paused: %s\n", taskID)
		if wasRunning {
			fmt.Printf("  Timer stopped: +%s\n", formatDuration(elapsed))
		}
		return nil
	},
}
