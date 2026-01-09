// Package learning provides CLI commands for managing project learnings.
package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

func newLearningCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "learning",
		Short:   "Manage learnings",
		Long:    `Manage project learnings - insights and knowledge discovered during work.`,
		Aliases: []string{"learnings"},
		// Note: "learn" is a root-level shortcut command for "tk learn 'insight'" directly
	}

	cmd.AddCommand(listCmd)
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(removeCmd)
	cmd.AddCommand(newPromoteCmd())
	cmd.AddCommand(rulesCmd)

	return cmd
}

// Cmd is the parent command for all learning operations
var Cmd = newLearningCmd()

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recorded learnings",
	Long: `Display all learnings recorded in the project context.

Examples:
  tk learning list              # List all learnings
  tk learning list -f json      # Output as JSON`,
	RunE: runList,
}

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add \"insight\"",
		Short: "Record an insight or knowledge discovered during work",
		Long: `Record an insight, discovery, or piece of knowledge learned while working.
Learnings are stored in the context section and help build project knowledge.

Use --scope to make the learning apply to specific files (glob pattern).
Path-scoped learnings are automatically synced to .claude/rules/ with frontmatter.

Use --permanent to also append the learning to CLAUDE.md for persistent documentation.

Examples:
  tk learning add "Redis connection pooling significantly improves API latency"
  tk learning add "The auth middleware must run before rate limiting" --permanent
  tk learning add "Always validate request bodies" --scope "src/api/**"
  tk learning add "Use memo for expensive computations" --scope "src/components/**/*.tsx"`,
		Args: cobra.ExactArgs(1),
		RunE: runAdd,
	}

	cmd.Flags().Bool("permanent", false, "Also append learning to CLAUDE.md")
	cmd.Flags().Bool("rule", false, "Explicitly mark this learning as a rule")
	cmd.Flags().String("scope", "", "Glob pattern for path-scoped rules (e.g., 'src/api/**')")

	return cmd
}

var removeCmd = &cobra.Command{
	Use:   "remove <id or text>",
	Short: "Remove a learning by ID or partial match",
	Long: `Remove a learning from the project context.

You can specify either:
- An ID (6-character code from 'tk learning list' output, e.g., a3x9k2)
- A partial text match (case-insensitive)

Examples:
  tk learning remove a3x9k2               # Remove learning by ID
  tk learning remove "redis"              # Remove first learning containing "redis"`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func newPromoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "promote <id or text>",
		Short: "Promote a learning to permanent documentation",
		Long: `Move a learning from Tasuku to your AI context file.

Tasuku auto-detects which context file to use based on your project:
- CLAUDE.md (Claude Code)
- .cursorrules (Cursor)
- .github/copilot-instructions.md (GitHub Copilot)
- AGENTS.md (Generic)

Use --to to specify a custom target file.

Examples:
  tk learning promote a3x9k2                # Promote learning by ID to auto-detected file
  tk learning promote "redis"               # Promote learning containing "redis"
  tk learning promote a3x9k2 --to AGENTS.md # Promote to specific file
  tk learning promote a3x9k2 --keep         # Keep in learnings after promoting`,
		Args: cobra.ExactArgs(1),
		RunE: runPromote,
	}

	cmd.Flags().String("to", "", "Target context file (auto-detected if not specified)")
	cmd.Flags().Bool("keep", false, "Keep the learning in Tasuku after promoting")

	return cmd
}

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List all never/always rule learnings",
	Long: `Display learnings that are marked as rules (never/always patterns).

Rules are learnings that contain key instruction words like "never" or "always".
These are typically important guidelines that should be promoted to permanent docs.

Examples:
  tk learning rules              # List all rule learnings
  tk learning rules -f json      # Output as JSON`,
	RunE: runRules,
}

func runList(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	learnings := f.Context.Learnings
	if len(learnings) == 0 {
		fmt.Println("No learnings recorded yet.")
		fmt.Println("Use: tk learning add \"your insight here\"")
		return nil
	}

	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(learnings, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(learnings)
		fmt.Print(string(data))
	default:
		ruleCount := 0
		scopedCount := 0
		for _, l := range learnings {
			if l.IsRule {
				ruleCount++
			}
			if l.Scope != "" {
				scopedCount++
			}
		}

		// Build header
		header := fmt.Sprintf("Learnings (%d", len(learnings))
		if ruleCount > 0 {
			header += fmt.Sprintf(", %d rules", ruleCount)
		}
		if scopedCount > 0 {
			header += fmt.Sprintf(", %d scoped", scopedCount)
		}
		header += "):\n\n"
		fmt.Print(header)

		for _, l := range learnings {
			age := formatAge(l.CreatedAt)
			markers := []string{}
			if l.IsRule {
				markers = append(markers, "RULE")
			}
			if l.Scope != "" {
				markers = append(markers, l.Scope)
			}
			markerStr := ""
			if len(markers) > 0 {
				markerStr = " [" + strings.Join(markers, "] [") + "]"
			}
			if age != "" {
				fmt.Printf("  [%s] %s%s (%s)\n", l.ID, l.Text, markerStr, age)
			} else {
				fmt.Printf("  [%s] %s%s\n", l.ID, l.Text, markerStr)
			}
		}
	}
	return nil
}

func runAdd(cmd *cobra.Command, args []string) error {
	learningText := args[0]
	permanent, _ := cmd.Flags().GetBool("permanent")
	forceRule, _ := cmd.Flags().GetBool("rule")
	scope, _ := cmd.Flags().GetString("scope")
	s := store.DefaultStorageWithWarning()

	var id string
	var isRule bool
	var err error

	var rulePtr *bool
	if forceRule {
		ruleVal := true
		rulePtr = &ruleVal
	}

	// Use AddLearningWithScope if scope is provided, otherwise use AddLearningWithRule
	if scope != "" {
		id, isRule, err = s.AddLearningWithScope(learningText, scope, rulePtr)
	} else {
		id, isRule, err = s.AddLearningWithRule(learningText, rulePtr)
	}
	if err != nil {
		return err
	}

	if permanent {
		if err := appendToCLAUDEmd(learningText, "learning"); err != nil {
			fmt.Printf("Warning: could not append to CLAUDE.md: %v\n", err)
		} else {
			fmt.Printf("Learning added [%s] (also appended to CLAUDE.md)\n", id)
			return nil
		}
	}

	// Build output string with markers
	markers := []string{}
	if isRule {
		markers = append(markers, "RULE")
	}
	if scope != "" {
		markers = append(markers, fmt.Sprintf("scope: %s", scope))
	}

	markerStr := ""
	if len(markers) > 0 {
		markerStr = " [" + strings.Join(markers, "] [") + "]"
	}

	fmt.Printf("Learning added [%s]%s\n", id, markerStr)

	if isRule {
		// Surface rules when a new one is added (limit to avoid noise)
		f, err := s.Read()
		if err == nil {
			var ruleCount int
			for _, l := range f.Context.Learnings {
				if l.IsRule {
					ruleCount++
				}
			}
			const maxRulesToShow = 7
			if ruleCount > 1 && ruleCount <= maxRulesToShow {
				fmt.Printf("\nActive rules (%d):\n", ruleCount)
				for _, l := range f.Context.Learnings {
					if l.IsRule {
						text := l.Text
						if len(text) > 70 {
							text = text[:67] + "..."
						}
						scopeInfo := ""
						if l.Scope != "" {
							scopeInfo = fmt.Sprintf(" [%s]", l.Scope)
						}
						fmt.Printf("  [%s] %s%s\n", l.ID, text, scopeInfo)
					}
				}
			} else if ruleCount > maxRulesToShow {
				fmt.Printf("\nActive rules: %d (run 'tk learning rules' to see all)\n", ruleCount)
			}
		}
		fmt.Println("\nHint: Promote rules to permanent docs with: tk learning promote", id)
	}
	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	query := args[0]

	removedText, err := s.RemoveLearning(query)
	if err == nil {
		fmt.Printf("Removed learning: %s\n", removedText)
		return nil
	}

	learning, err := s.FindLearningByText(query)
	if err != nil {
		return fmt.Errorf("no learning found matching %q", query)
	}

	removedText, err = s.RemoveLearning(learning.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Removed learning: %s\n", removedText)
	return nil
}

func runPromote(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	query := args[0]
	targetFile, _ := cmd.Flags().GetString("to")
	keep, _ := cmd.Flags().GetBool("keep")

	if targetFile == "" {
		targetFile = detectContextFile()
	}

	f, err := s.Read()
	if err != nil {
		return err
	}

	var foundLearning *task.Learning

	for i := range f.Context.Learnings {
		if f.Context.Learnings[i].ID == query {
			foundLearning = &f.Context.Learnings[i]
			break
		}
	}

	if foundLearning == nil {
		lowerQuery := strings.ToLower(query)
		for i := range f.Context.Learnings {
			if strings.Contains(strings.ToLower(f.Context.Learnings[i].Text), lowerQuery) {
				foundLearning = &f.Context.Learnings[i]
				break
			}
		}
	}

	if foundLearning == nil {
		return fmt.Errorf("no learning found matching %q", query)
	}

	if err := appendToContextFile(targetFile, foundLearning.Text); err != nil {
		return fmt.Errorf("failed to write to %s: %w", targetFile, err)
	}

	if !keep {
		if _, err := s.RemoveLearning(foundLearning.ID); err != nil {
			return err
		}
	}

	if keep {
		fmt.Printf("Promoted to %s (kept in learnings): %s\n", targetFile, foundLearning.Text)
	} else {
		fmt.Printf("Promoted to %s: %s\n", targetFile, foundLearning.Text)
	}
	return nil
}

func runRules(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	var rules []task.Learning
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			rules = append(rules, l)
		}
	}

	if len(rules) == 0 {
		fmt.Println("No rule learnings recorded yet.")
		fmt.Println("Rules are learnings that start with or contain 'never' or 'always'.")
		fmt.Println("Use: tk learning add \"Never use raw SQL queries\"")
		return nil
	}

	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(rules, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(rules)
		fmt.Print(string(data))
	default:
		fmt.Printf("Rules (%d):\n\n", len(rules))
		for _, l := range rules {
			age := formatAge(l.CreatedAt)
			if age != "" {
				fmt.Printf("  [%s] %s (%s)\n", l.ID, l.Text, age)
			} else {
				fmt.Printf("  [%s] %s\n", l.ID, l.Text)
			}
		}
		fmt.Println()
		fmt.Println("Hint: Promote rules to permanent docs with: tk learning promote <id>")
	}
	return nil
}

// Helper functions

func formatAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func detectContextFile() string {
	contextFiles := []struct {
		path        string
		description string
	}{
		{"CLAUDE.md", "Claude Code"},
		{"GEMINI.md", "Gemini"},
		{".cursorrules", "Cursor"},
		{".github/copilot-instructions.md", "GitHub Copilot"},
		{"AGENTS.md", "Generic AI agents"},
		{"AI.md", "Generic AI documentation"},
	}

	for _, cf := range contextFiles {
		if _, err := os.Stat(cf.path); err == nil {
			return cf.path
		}
	}

	return "CLAUDE.md"
}

func appendToCLAUDEmd(content, contentType string) error {
	claudePath := "CLAUDE.md"
	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(existing)
	var section string
	if contentType == "learning" {
		section = "\n\n## Learnings\n\n"
	} else {
		section = "\n\n## Notes\n\n"
	}

	entry := fmt.Sprintf("- %s\n", content)

	if strings.Contains(text, "## Learnings") {
		idx := strings.Index(text, "## Learnings")
		endOfLine := strings.Index(text[idx:], "\n") + idx + 1
		nextSection := strings.Index(text[endOfLine:], "\n## ")
		if nextSection == -1 {
			text = text + entry
		} else {
			insertAt := endOfLine + nextSection
			text = text[:insertAt] + entry + text[insertAt:]
		}
	} else {
		text = text + section + entry
	}

	return os.WriteFile(claudePath, []byte(text), 0644)
}

func appendToContextFile(filePath, learning string) error {
	existing, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(existing)
	entry := fmt.Sprintf("- %s\n", learning)

	if strings.Contains(text, "## Learnings") {
		idx := strings.Index(text, "## Learnings")
		endOfLine := strings.Index(text[idx:], "\n") + idx + 1

		nextSection := strings.Index(text[endOfLine:], "\n## ")
		if nextSection == -1 {
			text = text + entry
		} else {
			insertAt := endOfLine + nextSection
			text = text[:insertAt] + entry + text[insertAt:]
		}
	} else {
		if len(text) > 0 && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "\n## Learnings\n\n" + entry
	}

	return os.WriteFile(filePath, []byte(text), 0644)
}
