package task

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

var claimCmd = &cobra.Command{
	Use:   "claim <task-id> [agent-name]",
	Short: "Claim a task for an agent",
	Long: `Claim a task for exclusive work by an agent.

This sets the task's owner, marks it as in_progress, and records a claim
timestamp. If another agent tries to claim the same task, they will be
rejected unless the claim is stale (older than 2 hours by default).

If agent-name is not provided, it's auto-detected from:
  1. TASUKU_AGENT environment variable
  2. Git user.name
  3. System username
  4. Hostname

Use --sync to commit and push the claim for multi-worktree coordination.

Examples:
  tk task claim auth-feature              # Claim with auto-detected name
  tk task claim auth-feature agent-1      # Claim for specific agent
  tk task claim auth-feature --sync       # Claim and push to share with others`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		agentName := ""
		if len(args) == 2 {
			agentName = args[1]
		} else {
			agentName = getAgentName()
		}

		syncFlag, _ := cmd.Flags().GetBool("sync")
		s := store.DefaultStorageWithWarning()

		if err := s.ClaimTask(taskID, agentName); err != nil {
			return err
		}

		if err := s.SetStatus(taskID, task.StatusInProgress); err != nil {
			return err
		}

		fmt.Printf("Task %s claimed by %s\n", taskID, agentName)

		if syncFlag {
			commitMsg := fmt.Sprintf("chore(tasuku): claim %s for %s", taskID, agentName)
			if err := gitCommitAndPush(commitMsg); err != nil {
				fmt.Printf("Warning: %v\n", err)
			} else {
				fmt.Println("Changes committed and pushed")
			}
		}

		return nil
	},
}

func init() {
	claimCmd.Flags().Bool("sync", false, "Commit and push claim for multi-worktree coordination")
}

var releaseCmd = &cobra.Command{
	Use:   "release <task-id>",
	Short: "Release a claimed task",
	Long: `Release a task that was previously claimed.

This clears the owner and claim timestamp, sets status back to ready,
making the task available for other agents to claim.

Use --sync to commit and push the release for multi-worktree coordination.

Examples:
  tk task release auth-feature          # Release the task
  tk task release auth-feature --sync   # Release and push to share with others`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		syncFlag, _ := cmd.Flags().GetBool("sync")
		s := store.DefaultStorageWithWarning()

		if err := s.ReleaseTask(taskID); err != nil {
			return err
		}

		// Set status back to ready (if not done/blocked)
		f, err := s.Read()
		if err == nil {
			if t, exists := f.Tasks[taskID]; exists {
				if t.Status == task.StatusInProgress {
					s.SetStatus(taskID, task.StatusReady)
				}
			}
		}

		fmt.Printf("Task %s released\n", taskID)

		if syncFlag {
			commitMsg := fmt.Sprintf("chore(tasuku): release %s", taskID)
			if err := gitCommitAndPush(commitMsg); err != nil {
				fmt.Printf("Warning: %v\n", err)
			} else {
				fmt.Println("Changes committed and pushed")
			}
		}

		return nil
	},
}

func init() {
	releaseCmd.Flags().Bool("sync", false, "Commit and push release for multi-worktree coordination")
}

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
		s := store.DefaultStorageWithWarning()
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

		switch config.OutputFormat {
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
