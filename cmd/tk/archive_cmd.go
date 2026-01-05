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
// Archive Management Commands
// =============================================================================

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Manage archived tasks",
	Long: `Archive completed tasks to reduce clutter while preserving history.

Archiving moves done tasks out of the active task list into a separate
archive section. Archived tasks can be listed, viewed, or restored.

Subcommands:
  add       Archive a specific done task
  all       Archive all done tasks older than a duration
  list      List archived tasks
  show      Show details of an archived task
  restore   Restore an archived task to active tasks
  clear     Permanently delete all archived tasks

Examples:
  tk task archive add my-task              # Archive a done task
  tk task archive add my-task --summary "Implemented auth feature"
  tk task archive all --older-than 7d      # Archive tasks done 7+ days ago
  tk task archive list                     # List archived tasks
  tk task archive restore my-task          # Restore to active tasks
  tk task archive clear                    # Clear all archived tasks`,
}

var archiveOlderThan string
var archiveSummary string

var archiveAddCmd = &cobra.Command{
	Use:   "add <task-id>",
	Short: "Archive a completed task",
	Long: `Archive a completed task by moving it to the archive.

The task must have status "done" to be archived. You can optionally
provide a summary that describes what was accomplished.

Examples:
  tk task archive add auth-feature
  tk task archive add auth-feature --summary "Added OAuth2 login flow"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		if err := s.ArchiveTask(taskID, archiveSummary); err != nil {
			return err
		}

		if archiveSummary != "" {
			fmt.Printf("Archived task %s with summary: %s\n", taskID, archiveSummary)
		} else {
			fmt.Printf("Archived task %s\n", taskID)
		}
		return nil
	},
}

var archiveAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Archive all done tasks older than a duration",
	Long: `Archive all completed tasks that are older than the specified duration.

The duration is measured from when the task was last updated (marked done).

Duration format: 1h (hours), 1d (days), 1w (weeks)
  Examples: 24h, 7d, 2w, 30d

Examples:
  tk task archive all --older-than 7d    # Archive tasks done 7+ days ago
  tk task archive all --older-than 24h   # Archive tasks done 24+ hours ago
  tk task archive all --older-than 30d   # Archive tasks done 30+ days ago`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if archiveOlderThan == "" {
			return fmt.Errorf("--older-than is required (e.g., 7d, 24h, 2w)")
		}

		duration, err := parseDuration(archiveOlderThan)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", archiveOlderThan, err)
		}

		s := store.Default()
		archived, err := s.ArchiveDoneTasks(duration)
		if err != nil {
			return err
		}

		if len(archived) == 0 {
			fmt.Printf("No done tasks older than %s to archive\n", archiveOlderThan)
		} else {
			fmt.Printf("Archived %d tasks:\n", len(archived))
			for _, id := range archived {
				fmt.Printf("  - %s\n", id)
			}
		}
		return nil
	},
}

// parseDuration parses a human-friendly duration string like "7d", "24h", "2w"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return 0, err
	}

	switch unit {
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %q (use h, d, or w)", string(unit))
	}
}

// archivedTaskInfo for JSON/YAML output
type archivedTaskInfo struct {
	ID          string    `json:"id" yaml:"id"`
	Description string    `json:"description" yaml:"description"`
	Summary     string    `json:"summary,omitempty" yaml:"summary,omitempty"`
	TotalTime   string    `json:"total_time,omitempty" yaml:"total_time,omitempty"`
	ArchivedAt  time.Time `json:"archived_at" yaml:"archived_at"`
	CompletedAt time.Time `json:"completed_at" yaml:"completed_at"`
}

var archiveListCmd = &cobra.Command{
	Use:   "list",
	Short: "List archived tasks",
	Long: `List all archived tasks.

Shows task ID, description, summary (if any), total time spent,
and when the task was archived.

Examples:
  tk task archive list
  tk task archive list -f json
  tk task archive list -f yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		archived, err := s.GetArchivedTasks()
		if err != nil {
			return err
		}

		if len(archived) == 0 {
			fmt.Println("No archived tasks")
			return nil
		}

		// Sort by archived date
		var items []archivedTaskInfo
		for id, t := range archived {
			item := archivedTaskInfo{
				ID:          id,
				Description: t.Description,
				Summary:     t.Summary,
				ArchivedAt:  t.ArchivedAt,
				CompletedAt: t.UpdatedAt,
			}
			if t.TotalTime > 0 {
				item.TotalTime = task.Duration(t.TotalTime).FormatHumanReadable()
			}
			items = append(items, item)
		}

		sort.Slice(items, func(i, j int) bool {
			return items[i].ArchivedAt.After(items[j].ArchivedAt)
		})

		switch outputFormat {
		case "json":
			data, _ := json.MarshalIndent(items, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(items)
			fmt.Print(string(data))
		default:
			fmt.Printf("Archived tasks (%d):\n", len(items))
			for _, item := range items {
				timeStr := ""
				if item.TotalTime != "" {
					timeStr = fmt.Sprintf(" [%s]", item.TotalTime)
				}
				summaryStr := ""
				if item.Summary != "" {
					summaryStr = fmt.Sprintf("\n      Summary: %s", item.Summary)
				}
				fmt.Printf("  %s: %s%s%s\n", item.ID, item.Description, timeStr, summaryStr)
				fmt.Printf("      Archived: %s\n", item.ArchivedAt.Format("2006-01-02 15:04"))
			}
		}
		return nil
	},
}

var archiveShowCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show details of an archived task",
	Long: `Show detailed information about an archived task.

Examples:
  tk task archive show auth-feature
  tk task archive show auth-feature -f json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		archived, err := s.GetArchivedTask(taskID)
		if err != nil {
			return err
		}

		item := archivedTaskInfo{
			ID:          taskID,
			Description: archived.Description,
			Summary:     archived.Summary,
			ArchivedAt:  archived.ArchivedAt,
			CompletedAt: archived.UpdatedAt,
		}
		if archived.TotalTime > 0 {
			item.TotalTime = task.Duration(archived.TotalTime).FormatHumanReadable()
		}

		switch outputFormat {
		case "json":
			data, _ := json.MarshalIndent(item, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(item)
			fmt.Print(string(data))
		default:
			fmt.Printf("Archived Task: %s\n", taskID)
			fmt.Printf("  Description: %s\n", archived.Description)
			if archived.Summary != "" {
				fmt.Printf("  Summary: %s\n", archived.Summary)
			}
			if item.TotalTime != "" {
				fmt.Printf("  Time Spent: %s\n", item.TotalTime)
			}
			fmt.Printf("  Completed: %s\n", archived.UpdatedAt.Format("2006-01-02 15:04"))
			fmt.Printf("  Archived: %s\n", archived.ArchivedAt.Format("2006-01-02 15:04"))
			if archived.Priority != nil {
				fmt.Printf("  Priority: %s\n", task.PriorityName(*archived.Priority))
			}
			if len(archived.Tags) > 0 {
				fmt.Printf("  Tags: %v\n", archived.Tags)
			}
		}
		return nil
	},
}

var archiveRestoreCmd = &cobra.Command{
	Use:   "restore <task-id>",
	Short: "Restore an archived task to active tasks",
	Long: `Restore an archived task back to the active task list.

The restored task will have status "ready" and can be worked on again.

Examples:
  tk task archive restore auth-feature`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		if err := s.RestoreTask(taskID); err != nil {
			return err
		}

		fmt.Printf("Restored task %s (status: ready)\n", taskID)
		return nil
	},
}

var archiveClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Permanently delete all archived tasks",
	Long: `Permanently delete all archived tasks.

This action cannot be undone. Use with caution.

Examples:
  tk task archive clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()

		count, err := s.ClearArchive()
		if err != nil {
			return err
		}

		if count == 0 {
			fmt.Println("No archived tasks to clear")
		} else {
			fmt.Printf("Permanently deleted %d archived tasks\n", count)
		}
		return nil
	},
}

func init() {
	archiveCmd.AddCommand(archiveAddCmd)
	archiveCmd.AddCommand(archiveAllCmd)
	archiveCmd.AddCommand(archiveListCmd)
	archiveCmd.AddCommand(archiveShowCmd)
	archiveCmd.AddCommand(archiveRestoreCmd)
	archiveCmd.AddCommand(archiveClearCmd)

	// Add flags
	archiveAddCmd.Flags().StringVar(&archiveSummary, "summary", "", "Summary of what was accomplished")
	archiveAllCmd.Flags().StringVar(&archiveOlderThan, "older-than", "", "Archive tasks older than duration (e.g., 7d, 24h, 2w)")
}
