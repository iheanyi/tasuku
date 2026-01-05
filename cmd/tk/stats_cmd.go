package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

// =============================================================================
// Task Stats Command (under task parent - noun-verb pattern)
// =============================================================================

var taskStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show task statistics and progress",
	Long: `Display task statistics including counts by status, completion percentage,
context counts (learnings/decisions), and tasks with most blockers.

Examples:
  tk task stats              # Show stats in table format
  tk task stats -f json      # Output as JSON
  tk task stats -f yaml      # Output as YAML`,
	RunE: runStats,
}

// =============================================================================
// Deprecated stats Command (kept for backward compatibility)
// =============================================================================

func init() {
	rootCmd.AddCommand(statsCmd)
}

var statsCmd = &cobra.Command{
	Use:        "stats",
	Hidden:     true,
	Deprecated: "use 'tk task stats' instead",
	Short:      "Show task statistics and progress",
	Long: `Display task statistics including counts by status, completion percentage,
context counts (learnings/decisions), and tasks with most blockers.

Examples:
  tk stats              # Show stats in table format
  tk stats -f json      # Output as JSON
  tk stats -f yaml      # Output as YAML`,
	RunE: runStats,
}

// Shared implementation for stats
func runStats(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	stats := computeStats(f)
	return outputStats(stats)
}

// Stats represents task statistics
type Stats struct {
	TotalTasks      int            `json:"total_tasks" yaml:"total_tasks"`
	ByStatus        map[string]int `json:"by_status" yaml:"by_status"`
	StatusPercent   map[string]int `json:"status_percent" yaml:"status_percent"`
	CompletionPct   int            `json:"completion_percent" yaml:"completion_percent"`
	CompletedCount  int            `json:"completed_count" yaml:"completed_count"`
	LearningsCount  int            `json:"learnings_count" yaml:"learnings_count"`
	DecisionsCount  int            `json:"decisions_count" yaml:"decisions_count"`
	MostBlocked     []BlockedTask  `json:"most_blocked,omitempty" yaml:"most_blocked,omitempty"`
	OldestTasks     []OldTask      `json:"oldest_tasks,omitempty" yaml:"oldest_tasks,omitempty"`
}

// BlockedTask represents a task with blockers count
type BlockedTask struct {
	ID           string `json:"id" yaml:"id"`
	BlockerCount int    `json:"blocker_count" yaml:"blocker_count"`
}

// OldTask represents a task with its age
type OldTask struct {
	ID        string    `json:"id" yaml:"id"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	Age       string    `json:"age" yaml:"age"`
}

func computeStats(f *task.File) Stats {
	stats := Stats{
		ByStatus:      make(map[string]int),
		StatusPercent: make(map[string]int),
	}

	// Count tasks by status
	for _, t := range f.Tasks {
		stats.TotalTasks++
		stats.ByStatus[string(t.Status)]++
		if t.Status == task.StatusDone {
			stats.CompletedCount++
		}
	}

	// Calculate percentages
	if stats.TotalTasks > 0 {
		stats.CompletionPct = (stats.CompletedCount * 100) / stats.TotalTasks
		for status, count := range stats.ByStatus {
			stats.StatusPercent[status] = (count * 100) / stats.TotalTasks
		}
	}

	// Count context items
	stats.LearningsCount = len(f.Context.Learnings)
	stats.DecisionsCount = len(f.Context.Decisions)

	// Find tasks with most blockers
	var blockedTasks []BlockedTask
	for id, t := range f.Tasks {
		if len(t.BlockedBy) > 0 {
			blockedTasks = append(blockedTasks, BlockedTask{
				ID:           id,
				BlockerCount: len(t.BlockedBy),
			})
		}
	}
	sort.Slice(blockedTasks, func(i, j int) bool {
		return blockedTasks[i].BlockerCount > blockedTasks[j].BlockerCount
	})
	if len(blockedTasks) > 5 {
		blockedTasks = blockedTasks[:5]
	}
	stats.MostBlocked = blockedTasks

	// Find oldest non-done tasks
	var oldTasks []OldTask
	now := time.Now()
	for id, t := range f.Tasks {
		if t.Status != task.StatusDone {
			age := now.Sub(t.CreatedAt)
			oldTasks = append(oldTasks, OldTask{
				ID:        id,
				CreatedAt: t.CreatedAt,
				Age:       formatDurationAge(age),
			})
		}
	}
	sort.Slice(oldTasks, func(i, j int) bool {
		return oldTasks[i].CreatedAt.Before(oldTasks[j].CreatedAt)
	})
	if len(oldTasks) > 5 {
		oldTasks = oldTasks[:5]
	}
	stats.OldestTasks = oldTasks

	return stats
}

func formatDurationAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 30 {
		months := days / 30
		return fmt.Sprintf("%d months", months)
	}
	if days > 0 {
		return fmt.Sprintf("%d days", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%d minutes", minutes)
}

func outputStats(stats Stats) error {
	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(stats)
		fmt.Print(string(data))
	default: // table
		fmt.Println("Task Statistics")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Total tasks:     %d\n", stats.TotalTasks)

		// Status breakdown
		statuses := []struct {
			name   string
			status string
		}{
			{"Ready", "ready"},
			{"In Progress", "in_progress"},
			{"Blocked", "blocked"},
			{"Done", "done"},
		}

		for _, s := range statuses {
			count := stats.ByStatus[s.status]
			pct := stats.StatusPercent[s.status]
			fmt.Printf("  %-14s %d (%d%%)\n", s.name+":", count, pct)
		}

		fmt.Println()
		fmt.Printf("Completion:      %d%% (%d/%d)\n", stats.CompletionPct, stats.CompletedCount, stats.TotalTasks)

		fmt.Println()
		fmt.Println("Context:")
		fmt.Printf("  Learnings:     %d\n", stats.LearningsCount)
		fmt.Printf("  Decisions:     %d\n", stats.DecisionsCount)

		if len(stats.MostBlocked) > 0 {
			fmt.Println()
			fmt.Println("Most Blocked:")
			for _, bt := range stats.MostBlocked {
				fmt.Printf("  %-16s blocked by %d tasks\n", bt.ID, bt.BlockerCount)
			}
		}

		if len(stats.OldestTasks) > 0 {
			fmt.Println()
			fmt.Println("Oldest Tasks:")
			for _, ot := range stats.OldestTasks {
				fmt.Printf("  %-16s %s old\n", ot.ID, ot.Age)
			}
		}
	}
	return nil
}
