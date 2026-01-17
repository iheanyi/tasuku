package task

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive [task-id]",
		Short: "Archive a completed task",
		Long: `Archive completed tasks to reduce clutter while preserving history.

Archive a single task by ID, or use --older-than to bulk archive
done tasks older than a specified duration.

Archived tasks can be listed with 'tk task list --status archived'
and restored with 'tk task restore <id>'.

Duration format for --older-than: 1h (hours), 1d (days), 1w (weeks)

Examples:
  tk task archive my-task                # Archive a done task
  tk task archive my-task --summary "Implemented auth feature"
  tk task archive --older-than 7d        # Archive tasks done 7+ days ago
  tk task archive --older-than 24h       # Archive tasks done 24+ hours ago
  tk task list --status archived         # List archived tasks
  tk task restore my-task                # Restore to active tasks

Subcommands:
  show      Show details of an archived task
  clear     Permanently delete all archived tasks`,
		Args: cobra.MaximumNArgs(1),
		RunE: runArchive,
	}

	cmd.Flags().String("summary", "", "Summary of what was accomplished (only with task ID)")
	cmd.Flags().String("older-than", "", "Archive all done tasks older than duration (e.g., 7d, 24h, 2w)")

	// Keep show and clear as subcommands
	cmd.AddCommand(archiveShowCmd)
	cmd.AddCommand(archiveClearCmd)

	return cmd
}

var archiveCmd = newArchiveCmd()

func runArchive(cmd *cobra.Command, args []string) error {
	olderThan, _ := cmd.Flags().GetString("older-than")
	summary, _ := cmd.Flags().GetString("summary")

	// Validate: can't use both task ID and --older-than
	if len(args) > 0 && olderThan != "" {
		return fmt.Errorf("cannot use both task ID and --older-than; use one or the other")
	}

	// Validate: must provide either task ID or --older-than
	if len(args) == 0 && olderThan == "" {
		return fmt.Errorf("provide a task ID to archive, or use --older-than for bulk archiving")
	}

	// Validate: --summary only makes sense with single task
	if summary != "" && olderThan != "" {
		return fmt.Errorf("--summary can only be used when archiving a single task")
	}

	s := store.DefaultStorageWithWarning()

	// Bulk archive mode
	if olderThan != "" {
		return archiveBulk(s, olderThan)
	}

	// Single task archive mode
	taskID := args[0]
	return archiveSingle(s, taskID, summary)
}

func archiveSingle(s store.Storage, taskID, summary string) error {
	if err := s.ArchiveTask(taskID, summary); err != nil {
		return err
	}

	if summary != "" {
		fmt.Printf("Archived task %s with summary: %s\n", taskID, summary)
	} else {
		fmt.Printf("Archived task %s\n", taskID)
	}
	return nil
}

func archiveBulk(s store.Storage, olderThan string) error {
	duration, err := parseArchiveDuration(olderThan)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", olderThan, err)
	}

	archived, err := s.ArchiveDoneTasks(duration)
	if err != nil {
		return err
	}

	if len(archived) == 0 {
		fmt.Printf("No done tasks older than %s to archive\n", olderThan)
	} else {
		fmt.Printf("Archived %d tasks:\n", len(archived))
		for _, id := range archived {
			fmt.Printf("  - %s\n", id)
		}
	}
	return nil
}

func parseArchiveDuration(s string) (time.Duration, error) {
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

type archivedTaskInfo struct {
	ID          string    `json:"id" yaml:"id"`
	Description string    `json:"description" yaml:"description"`
	Summary     string    `json:"summary,omitempty" yaml:"summary,omitempty"`
	TotalTime   string    `json:"total_time,omitempty" yaml:"total_time,omitempty"`
	ArchivedAt  time.Time `json:"archived_at" yaml:"archived_at"`
	CompletedAt time.Time `json:"completed_at" yaml:"completed_at"`
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
		s := store.DefaultStorageWithWarning()

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

		switch config.OutputFormat {
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
			fmt.Printf("  Completed: %s\n", task.FormatLocalTime(archived.UpdatedAt))
			fmt.Printf("  Archived: %s\n", task.FormatLocalTime(archived.ArchivedAt))
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

var archiveClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Permanently delete all archived tasks",
	Long: `Permanently delete all archived tasks.

This action cannot be undone. Use with caution.

Examples:
  tk task archive clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.DefaultStorageWithWarning()

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
