package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

// =============================================================================
// Timer Management Commands (V2.0)
// =============================================================================

var timerCmd = &cobra.Command{
	Use:   "timer",
	Short: "Track time spent on tasks",
	Long: `Track time spent on tasks with start/stop timers.

Time tracking allows you to measure how long you spend on each task.
Duration is cumulative across multiple start/stop cycles.

Subcommands:
  start     Start a timer on a task
  stop      Stop a running timer
  status    Show active timers

Examples:
  tk task timer start my-task     # Start timing my-task
  tk task timer stop my-task      # Stop timer, record duration
  tk task timer status            # Show all active timers
  tk task timer status my-task    # Show timer for specific task`,
}

var timerStartCmd = &cobra.Command{
	Use:   "start <task-id>",
	Short: "Start a timer on a task",
	Long: `Start a timer on a task.

Only one timer can run per task at a time. Starting a timer on a task
that already has a running timer will return an error.

Examples:
  tk task timer start my-task
  tk task timer start fix-bug`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		if err := s.StartTimer(taskID); err != nil {
			return err
		}

		fmt.Printf("Timer started for task %s\n", taskID)
		return nil
	},
}

var timerStopCmd = &cobra.Command{
	Use:   "stop <task-id>",
	Short: "Stop a running timer",
	Long: `Stop a running timer on a task.

Stopping a timer calculates the elapsed time and adds it to the
task's total duration. The duration is cumulative across multiple
start/stop cycles.

Examples:
  tk task timer stop my-task
  tk task timer stop fix-bug`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		elapsed, err := s.StopTimer(taskID)
		if err != nil {
			return err
		}

		// Read the updated task to get total duration
		f, err := s.Read()
		if err != nil {
			return err
		}

		t := f.Tasks[taskID]
		fmt.Printf("Timer stopped for task %s\n", taskID)
		fmt.Printf("  Session:  %s\n", formatDuration(elapsed))
		fmt.Printf("  Total:    %s\n", t.Duration.FormatHumanReadable())
		return nil
	},
}

// timerStatusInfo holds timer status information for output
type timerStatusInfo struct {
	TaskID      string `json:"task_id" yaml:"task_id"`
	Description string `json:"description" yaml:"description"`
	StartedAt   string `json:"started_at" yaml:"started_at"`
	Elapsed     string `json:"elapsed" yaml:"elapsed"`
	Total       string `json:"total" yaml:"total"`
}

var timerStatusCmd = &cobra.Command{
	Use:   "status [task-id]",
	Short: "Show active timers",
	Long: `Show active timers.

Without a task ID, shows all tasks with running timers.
With a task ID, shows timer status for that specific task.

Examples:
  tk task timer status                  # Show all active timers
  tk task timer status my-task          # Show timer for specific task
  tk task timer status -f json          # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()

		if len(args) == 1 {
			// Show status for specific task
			taskID := args[0]
			f, err := s.Read()
			if err != nil {
				return err
			}

			t, exists := f.Tasks[taskID]
			if !exists {
				return fmt.Errorf("task not found: %s", taskID)
			}

			return outputTimerStatus(taskID, t)
		}

		// Show all active timers
		activeTimers, err := s.GetActiveTimers()
		if err != nil {
			return err
		}

		if len(activeTimers) == 0 {
			fmt.Println("No active timers")
			return nil
		}

		return outputActiveTimers(activeTimers)
	},
}

func outputTimerStatus(id string, t task.Task) error {
	info := timerStatusInfo{
		TaskID:      id,
		Description: t.Description,
		Total:       t.Duration.FormatHumanReadable(),
	}

	if t.TimerStart != nil {
		info.StartedAt = t.TimerStart.Format(time.RFC3339)
		elapsed := time.Since(*t.TimerStart)
		info.Elapsed = formatDuration(elapsed)
	}

	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(info)
		fmt.Print(string(data))
	default:
		fmt.Printf("Task:        %s\n", id)
		fmt.Printf("Description: %s\n", t.Description)
		if t.TimerStart != nil {
			fmt.Printf("Timer:       RUNNING (started %s ago)\n", formatDuration(time.Since(*t.TimerStart)))
			fmt.Printf("Session:     %s\n", info.Elapsed)
		} else {
			fmt.Printf("Timer:       stopped\n")
		}
		totalDuration := t.CurrentDuration()
		fmt.Printf("Total:       %s\n", formatDuration(totalDuration))
	}
	return nil
}

func outputActiveTimers(timers map[string]task.Task) error {
	// Sort by task ID for consistent output
	var ids []string
	for id := range timers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var infos []timerStatusInfo
	for _, id := range ids {
		t := timers[id]
		elapsed := time.Since(*t.TimerStart)
		infos = append(infos, timerStatusInfo{
			TaskID:      id,
			Description: t.Description,
			StartedAt:   t.TimerStart.Format(time.RFC3339),
			Elapsed:     formatDuration(elapsed),
			Total:       formatDuration(t.CurrentDuration()),
		})
	}

	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(infos, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(infos)
		fmt.Print(string(data))
	default:
		fmt.Printf("Active timers (%d):\n\n", len(infos))
		for _, info := range infos {
			fmt.Printf("  %s\n", info.TaskID)
			fmt.Printf("    Description: %s\n", truncateString(info.Description, 50))
			fmt.Printf("    Running:     %s\n", info.Elapsed)
			fmt.Printf("    Total:       %s\n", info.Total)
			fmt.Println()
		}
	}
	return nil
}

// formatDuration formats a time.Duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	timerCmd.AddCommand(timerStartCmd)
	timerCmd.AddCommand(timerStopCmd)
	timerCmd.AddCommand(timerStatusCmd)
}
