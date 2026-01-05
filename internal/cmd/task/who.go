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
