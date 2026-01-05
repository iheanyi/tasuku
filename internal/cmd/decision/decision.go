// Package decision provides CLI commands for managing architectural decisions.
package decision

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

// Cmd is the parent command for all decision operations
var Cmd = &cobra.Command{
	Use:     "decision",
	Short:   "Manage decisions",
	Long:    `Manage architectural and design decisions recorded during development.`,
	Aliases: []string{"decisions"},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(removeCmd)

	// Flags for add command
	addCmd.Flags().String("id", "", "Decision ID")
	addCmd.Flags().String("chose", "", "The option chosen")
	addCmd.Flags().StringSlice("over", nil, "Alternatives considered (repeatable or comma-separated)")
	addCmd.Flags().String("because", "", "Reasoning")
	addCmd.MarkFlagRequired("id")
	addCmd.MarkFlagRequired("chose")
	addCmd.MarkFlagRequired("because")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recorded decisions",
	Long: `Display all decisions recorded in the project context.

Examples:
  tk decision list              # List all decisions
  tk decision list -f json      # Output as JSON`,
	RunE: runList,
}

var addCmd = &cobra.Command{
	Use:   "add --id <id> --chose <option> --over <alternatives> --because <reason>",
	Short: "Record an architectural or design decision",
	Long: `Document a decision made during development for future reference.

Decisions capture:
  - What was chosen
  - What alternatives were considered
  - Why this choice was made

This creates an audit trail of architectural choices, helping future
developers (or agents) understand why things are the way they are.

Required Flags:
  --id       Unique identifier for this decision (e.g., "use-postgres")
  --chose    The option that was selected
  --because  The reasoning behind the choice

Optional Flags:
  --over     Alternatives considered (repeatable or comma-separated)

Examples:
  tk decision add --id db-choice --chose PostgreSQL --over MySQL --over SQLite --because "Better JSON support"
  tk decision add --id auth-method --chose JWT --over sessions,OAuth --because "Stateless and scalable"
  tk decision add --id framework --chose Cobra --because "Standard Go CLI library"`,
	RunE: runAdd,
}

var removeCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove a decision by ID",
	Long: `Remove a decision from the project context by its ID.

Examples:
  tk decision remove json-format          # Remove decision with ID "json-format"
  tk decision remove use-cobra            # Remove decision with ID "use-cobra"`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func runList(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	decisions := f.Context.Decisions
	if len(decisions) == 0 {
		fmt.Println("No decisions recorded yet.")
		fmt.Println("Use: tk decision add --id <id> --chose <choice> --over <alternatives> --because <reason>")
		return nil
	}

	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(decisions, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(decisions)
		fmt.Print(string(data))
	default:
		fmt.Printf("Decisions (%d):\n\n", len(decisions))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, d := range decisions {
			overStr := strings.Join(d.Over, ", ")
			if len(overStr) > 30 {
				overStr = overStr[:27] + "..."
			}
			becauseStr := d.Because
			if len(becauseStr) > 40 {
				becauseStr = becauseStr[:37] + "..."
			}
			fmt.Fprintf(w, "  %s\tChose: %s\tOver: %s\n", d.ID, d.Chose, overStr)
			fmt.Fprintf(w, "  \tBecause: %s\n\n", becauseStr)
		}
		w.Flush()
	}
	return nil
}

func runAdd(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	chose, _ := cmd.Flags().GetString("chose")
	alternatives, _ := cmd.Flags().GetStringSlice("over")
	because, _ := cmd.Flags().GetString("because")

	if id == "" || chose == "" || because == "" {
		return fmt.Errorf("usage: tk decision add --id <id> --chose <choice> --over <options> --because <reason>")
	}

	for i := range alternatives {
		alternatives[i] = strings.TrimSpace(alternatives[i])
	}

	d := task.Decision{
		ID:      id,
		Chose:   chose,
		Over:    alternatives,
		Because: because,
	}

	s := store.DefaultStorageWithWarning()
	if err := s.AddDecision(d); err != nil {
		return err
	}

	fmt.Printf("Decision recorded: %s\n", id)
	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	decisionID := args[0]
	s := store.DefaultStorageWithWarning()

	return s.Update(func(f *task.File) error {
		for i, d := range f.Context.Decisions {
			if d.ID == decisionID {
				removed := f.Context.Decisions[i]
				f.Context.Decisions = append(f.Context.Decisions[:i], f.Context.Decisions[i+1:]...)
				fmt.Printf("Removed decision: %s (chose %s)\n", removed.ID, removed.Chose)
				return nil
			}
		}
		return fmt.Errorf("decision not found: %s", decisionID)
	})
}
