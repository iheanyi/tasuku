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
// Agent Coordination Commands (V2.0 - claim/release/who)
// =============================================================================

var claimCmd = &cobra.Command{
	Use:   "claim <task-id> <agent-name>",
	Short: "Claim a task for an agent",
	Long: `Claim a task for exclusive work by an agent.

This sets the task's owner and records a claim timestamp. If another
agent tries to claim the same task, they will be rejected unless the
claim is stale (older than 2 hours by default).

Claiming a task helps coordinate multiple agents working in parallel
to avoid conflicts and duplicate work.

Examples:
  tk task claim auth-feature agent-1     # Claim for agent-1
  tk task claim api-design claude        # Claim for claude`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		agentName := args[1]
		s := store.Default()

		if err := s.ClaimTask(taskID, agentName); err != nil {
			return err
		}

		fmt.Printf("Task %s claimed by %s\n", taskID, agentName)
		return nil
	},
}

var releaseCmd = &cobra.Command{
	Use:   "release <task-id>",
	Short: "Release a claimed task",
	Long: `Release a task that was previously claimed.

This clears the owner and claim timestamp, making the task available
for other agents to claim.

Examples:
  tk task release auth-feature    # Release the task`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		s := store.Default()

		if err := s.ReleaseTask(taskID); err != nil {
			return err
		}

		fmt.Printf("Task %s released\n", taskID)
		return nil
	},
}

// claimInfo holds claim information for output
type claimInfo struct {
	TaskID    string     `json:"task_id" yaml:"task_id"`
	Owner     string     `json:"owner" yaml:"owner"`
	ClaimedAt *time.Time `json:"claimed_at,omitempty" yaml:"claimed_at,omitempty"`
	Status    string     `json:"status" yaml:"status"`
	Stale     bool       `json:"stale" yaml:"stale"`
}

var whoCmd = &cobra.Command{
	Use:   "who [agent-name]",
	Short: "Show claimed tasks",
	Long: `Show which tasks are claimed and by whom.

Without arguments, lists all claimed tasks grouped by owner.
With an agent name, shows only tasks claimed by that agent.

Stale claims (older than 2 hours) are marked to indicate they
may be eligible for takeover.

Examples:
  tk task who                 # Show all claimed tasks
  tk task who agent-1         # Show tasks claimed by agent-1
  tk task who -f json         # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := store.Default()
		f, err := s.Read()
		if err != nil {
			return err
		}

		var filterOwner string
		if len(args) == 1 {
			filterOwner = args[0]
		}

		// Collect claimed tasks
		var claims []claimInfo
		for id, t := range f.Tasks {
			if t.Owner == nil {
				continue
			}
			if filterOwner != "" && *t.Owner != filterOwner {
				continue
			}
			claims = append(claims, claimInfo{
				TaskID:    id,
				Owner:     *t.Owner,
				ClaimedAt: t.ClaimedAt,
				Status:    string(t.Status),
				Stale:     t.IsClaimStale(task.DefaultClaimTimeout),
			})
		}

		// Sort by owner, then by task ID
		sort.Slice(claims, func(i, j int) bool {
			if claims[i].Owner != claims[j].Owner {
				return claims[i].Owner < claims[j].Owner
			}
			return claims[i].TaskID < claims[j].TaskID
		})

		switch outputFormat {
		case "json":
			data, _ := json.MarshalIndent(claims, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(claims)
			fmt.Print(string(data))
		default:
			if len(claims) == 0 {
				if filterOwner != "" {
					fmt.Printf("No tasks claimed by %s\n", filterOwner)
				} else {
					fmt.Println("No tasks are currently claimed")
				}
				return nil
			}

			// Group by owner for table output
			currentOwner := ""
			for _, c := range claims {
				if c.Owner != currentOwner {
					if currentOwner != "" {
						fmt.Println()
					}
					fmt.Printf("Owner: %s\n", c.Owner)
					currentOwner = c.Owner
				}

				staleMarker := ""
				if c.Stale {
					staleMarker = " (stale)"
				}

				claimTime := ""
				if c.ClaimedAt != nil {
					claimTime = fmt.Sprintf(" - claimed %s", formatRelativeTime(*c.ClaimedAt))
				}

				fmt.Printf("  %s [%s]%s%s\n", c.TaskID, c.Status, claimTime, staleMarker)
			}
		}

		return nil
	},
}

// Note: formatRelativeTime is defined in main.go
