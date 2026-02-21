package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newTeammateIdleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teammate-idle",
		Short: "Show handoff guidance when a teammate goes idle",
		Long: `Called by Claude Code TeammateIdle hook when a teammate is about to go idle.

Shows:
  - Tasks owned by the teammate that block other tasks

Reads JSON from stdin with teammate_name, team_name, etc.
Always exits 0 (soft reminder, never blocks).

Examples:
  echo '{"teammate_name":"worker","team_name":"test"}' | tk hooks teammate-idle`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hookTeammateIdle()
		},
	}
}

type teammateIdleInput struct {
	TeammateName string `json:"teammate_name"`
	TeamName     string `json:"team_name"`
}

func hookTeammateIdle() error {
	var input teammateIdleInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		printTeammateIdleHeader("unknown")
		fmt.Println()
		printReflectionPrompts()
		return nil
	}

	name := input.TeammateName
	if name == "" {
		name = "unknown"
	}

	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		printTeammateIdleHeader(name)
		fmt.Println()
		printReflectionPrompts()
		return nil
	}
	if !s.Exists() {
		printTeammateIdleHeader(name)
		fmt.Println()
		printReflectionPrompts()
		return nil
	}

	f, err := s.Read()
	if err != nil {
		printTeammateIdleHeader(name)
		fmt.Println()
		printReflectionPrompts()
		return nil
	}

	// Find tasks owned by this teammate
	ownedWithDependents := findOwnedTasksWithDependents(name, f.Tasks)

	printTeammateIdleHeader(name)

	if len(ownedWithDependents) > 0 {
		fmt.Println()
		fmt.Println("Your tasks blocking others:")
		for _, owt := range ownedWithDependents {
			fmt.Printf("   %s [%s]: %s\n", owt.id, owt.task.Status, owt.task.Description)
			for _, dep := range owt.dependents {
				depTask := f.Tasks[dep]
				ownerStr := "unassigned"
				if depTask.Owner != nil && *depTask.Owner != "" {
					ownerStr = *depTask.Owner
				}
				fmt.Printf("     -> blocks: %s [%s] (owner: %s)\n", dep, depTask.Status, ownerStr)
			}
		}
	}

	fmt.Println()
	printReflectionPrompts()

	return nil
}

type ownedTaskWithDependents struct {
	id         string
	task       task.Task
	dependents []string
}

// findOwnedTasksWithDependents finds tasks owned by the given name that have dependents.
func findOwnedTasksWithDependents(owner string, tasks map[string]task.Task) []ownedTaskWithDependents {
	var result []ownedTaskWithDependents
	for id, t := range tasks {
		if t.Owner == nil || *t.Owner != owner {
			continue
		}
		if t.Status == task.StatusDone {
			continue
		}
		blocked := task.FindBlockedTasks(id, tasks)
		if len(blocked) > 0 {
			result = append(result, ownedTaskWithDependents{
				id:         id,
				task:       t,
				dependents: blocked,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].id < result[j].id
	})
	return result
}

func printTeammateIdleHeader(name string) {
	fmt.Printf("=== Teammate Idle: %s ===\n", name)
}
