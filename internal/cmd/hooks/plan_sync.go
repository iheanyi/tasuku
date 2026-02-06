package hooks

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/cmdutil"
	"github.com/iheanyi/tasuku/internal/store"
)

// PlanItem represents a parsed item from a plan file
type PlanItem struct {
	Description string
	Level       int  // Indentation level (0 = top-level)
	IsCheckbox  bool
	IsChecked   bool
	LineNumber  int
}

func newPlanSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan-sync <file>",
		Short: "Extract tasks from a plan file",
		Long: `Extract project-level tasks from a markdown plan file and create Tasuku tasks.

Applies the nudge rule to filter out session-level implementation steps,
only creating tasks for meaningful project-level work.

Supported formats:
  - Markdown checkboxes: - [ ] Task description
  - Bullet points: - Task description
  - Numbered lists: 1. Task description

Examples:
  tk hooks plan-sync plan.md           # Sync plan file
  tk hooks plan-sync plan.md --dry-run # Preview without creating
  tk hooks plan-sync plan.md --all     # Skip nudge rule, sync all`,
		Args: cobra.ExactArgs(1),
		RunE: runPlanSync,
	}

	cmd.Flags().Bool("dry-run", false, "Preview what would be created without making changes")
	cmd.Flags().Bool("all", false, "Sync all items, skip nudge rule filtering")

	return cmd
}

func runPlanSync(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	syncAll, _ := cmd.Flags().GetBool("all")

	// Parse the plan file
	items, err := ParsePlanFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse plan file: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("No actionable items found in plan file.")
		return nil
	}

	// Get storage
	s, err := store.DefaultStorageWithWarning()
	if err != nil {
		return err
	}
	if !s.Exists() {
		return fmt.Errorf("no Tasuku storage found - run 'tk init' first")
	}

	// Read existing tasks to check for duplicates
	f, err := s.Read()
	if err != nil {
		return fmt.Errorf("failed to read storage: %w", err)
	}

	// Categorize items
	var toCreate []PlanItem
	var skipped []PlanItem
	var exists []PlanItem

	for _, item := range items {
		id := generateID(item.Description)
		if id == "" {
			continue
		}

		// Check if already exists
		if _, found := f.Tasks[id]; found {
			exists = append(exists, item)
			continue
		}

		// Apply nudge rule unless --all
		if !syncAll && !shouldPersistTask(item.Description) {
			skipped = append(skipped, item)
			continue
		}

		toCreate = append(toCreate, item)
	}

	// Print results
	fmt.Printf("Scanning %s...\n\n", filePath)

	if len(toCreate) > 0 {
		if dryRun {
			fmt.Println("Would create tasks:")
		} else {
			fmt.Println("Creating tasks:")
		}
		for _, item := range toCreate {
			id := generateID(item.Description)
			fmt.Printf("  + %s: %s\n", id, cmdutil.Truncate(item.Description, 50))
			if !dryRun {
				if err := s.AddTask(id, item.Description); err != nil {
					fmt.Printf("    Error: %v\n", err)
				}
			}
		}
		fmt.Println()
	}

	if len(skipped) > 0 {
		fmt.Println("Skipped (session-level):")
		for _, item := range skipped {
			fmt.Printf("  - %s\n", cmdutil.Truncate(item.Description, 60))
		}
		fmt.Println()
	}

	if len(exists) > 0 {
		fmt.Println("Already exists:")
		for _, item := range exists {
			id := generateID(item.Description)
			fmt.Printf("  = %s\n", id)
		}
		fmt.Println()
	}

	// Summary
	action := "Created"
	if dryRun {
		action = "Would create"
	}
	fmt.Printf("%s %d tasks, skipped %d session-level items, %d already exist.\n",
		action, len(toCreate), len(skipped), len(exists))

	return nil
}

// ParsePlanFile parses a markdown plan file and extracts actionable items
func ParsePlanFile(path string) ([]PlanItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []PlanItem
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Patterns for extractable items
	checkboxPattern := regexp.MustCompile(`^(\s*)-\s*\[([ xX])\]\s*(.+)$`)
	bulletPattern := regexp.MustCompile(`^(\s*)[-*]\s+(.+)$`)
	numberedPattern := regexp.MustCompile(`^(\s*)\d+\.\s+(.+)$`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and headers
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		var item *PlanItem

		// Try checkbox pattern first
		if matches := checkboxPattern.FindStringSubmatch(line); matches != nil {
			indent := len(matches[1])
			checked := matches[2] == "x" || matches[2] == "X"
			desc := strings.TrimSpace(matches[3])
			item = &PlanItem{
				Description: desc,
				Level:       indent / 2, // Assume 2-space indent
				IsCheckbox:  true,
				IsChecked:   checked,
				LineNumber:  lineNum,
			}
		} else if matches := bulletPattern.FindStringSubmatch(line); matches != nil {
			// Bullet point (but not a checkbox, which we already checked)
			indent := len(matches[1])
			desc := strings.TrimSpace(matches[2])
			item = &PlanItem{
				Description: desc,
				Level:       indent / 2,
				IsCheckbox:  false,
				LineNumber:  lineNum,
			}
		} else if matches := numberedPattern.FindStringSubmatch(line); matches != nil {
			// Numbered list
			indent := len(matches[1])
			desc := strings.TrimSpace(matches[2])
			item = &PlanItem{
				Description: desc,
				Level:       indent / 2,
				IsCheckbox:  false,
				LineNumber:  lineNum,
			}
		}

		// Only include top-level items (Level == 0) by default
		if item != nil && item.Level == 0 {
			// Skip already-checked items
			if item.IsCheckbox && item.IsChecked {
				continue
			}
			items = append(items, *item)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

