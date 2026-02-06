package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
)

// SuggestResult represents the result of analyzing a task description
type SuggestResult struct {
	ShouldPersist       bool   `json:"should_persist" yaml:"should_persist"`
	Reason              string `json:"reason" yaml:"reason"`
	MatchedKeyword      string `json:"matched_keyword,omitempty" yaml:"matched_keyword,omitempty"`
	Recommendation      string `json:"recommendation" yaml:"recommendation"`
	SuggestedCommand    string `json:"suggested_command,omitempty" yaml:"suggested_command,omitempty"`
	OriginalDescription string `json:"original_description" yaml:"original_description"`
}

// newSuggestCmd creates the suggest command for analyzing task descriptions
func newSuggestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest \"task description\"",
		Short: "Analyze if a task should persist to tk or stay session-only",
		Long: `Analyze a task description to determine if it should be tracked in Tasuku
(project-level, persistent across sessions) or kept as a TodoWrite item only
(session-level, ephemeral).

This helps agents and users decide where to track work:
  - Project-level tasks (features, bugs, milestones) → tk task add
  - Session-level tasks (implementation steps, quick fixes) → TodoWrite only

Examples:
  tk suggest "Implement user authentication"
  # → ✓ PERSIST TO TK (project-level feature)

  tk suggest "Fix type error in auth.ts"
  # → ✗ KEEP SESSION-ONLY (implementation step)

  tk suggest "Add dark mode support" -f json
  # → JSON output with full analysis`,
		Args: cobra.ExactArgs(1),
		RunE: runSuggest,
	}
}

func runSuggest(cmd *cobra.Command, args []string) error {
	description := args[0]
	result := analyzeSuggestion(description)
	return outputSuggestion(result)
}

func analyzeSuggestion(description string) SuggestResult {
	desc := strings.ToLower(description)

	// Keywords that indicate project-level tasks (should persist to tk)
	projectKeywords := []string{
		"implement", "add feature", "build", "create", "develop",
		"fix bug", "bugfix", "hotfix", "patch",
		"refactor", "rewrite", "redesign", "rearchitect",
		"migrate", "upgrade", "update dependency",
		"integrate", "connect", "setup", "configure",
		"support", "enable", "add support",
		"milestone", "epic", "feature", "story",
		"api endpoint", "database", "schema",
		"authentication", "authorization", "security",
		"performance", "optimize", "cache",
		"deploy", "release", "ship",
	}

	// Keywords that indicate session-level tasks (TodoWrite only)
	sessionKeywords := []string{
		"fix type error", "fix typo", "fix lint",
		"update file", "edit file", "modify file",
		"read file", "check file", "review file",
		"run test", "run build", "run script",
		"verify", "check", "confirm", "ensure",
		"debug", "investigate", "look into",
		"format", "cleanup", "tidy",
		"add comment", "add docstring", "add import",
		"remove unused", "delete unused",
		"rename variable", "rename function",
	}

	shouldPersist := false
	reason := "No strong project-level indicators found"
	matchedKeyword := ""

	// Check for project keywords
	for _, kw := range projectKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = true
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains project-level keyword '%s' - this looks like a feature, bug, or significant change that should be tracked across sessions", kw)
			break
		}
	}

	// Session keywords can override if they match
	for _, kw := range sessionKeywords {
		if strings.Contains(desc, kw) {
			shouldPersist = false
			matchedKeyword = kw
			reason = fmt.Sprintf("Contains session-level keyword '%s' - this looks like an implementation step that doesn't need to persist", kw)
			break
		}
	}

	result := SuggestResult{
		ShouldPersist:       shouldPersist,
		Reason:              reason,
		MatchedKeyword:      matchedKeyword,
		OriginalDescription: description,
	}

	if shouldPersist {
		result.SuggestedCommand = fmt.Sprintf("tk task add %q", description)
		result.Recommendation = "Add this to tk for persistent tracking across sessions"
	} else {
		result.Recommendation = "Keep this in TodoWrite only - it's a session-level implementation step"
	}

	return result
}

func outputSuggestion(result SuggestResult) error {
	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(result)
		fmt.Print(string(data))
	default:
		// Human-readable format
		if result.ShouldPersist {
			fmt.Println("✓ PERSIST TO TK")
			fmt.Println()
			fmt.Printf("  Reason: %s\n", result.Reason)
			fmt.Println()
			fmt.Printf("  Suggested command:\n")
			fmt.Printf("    %s\n", result.SuggestedCommand)
		} else {
			fmt.Println("✗ KEEP SESSION-ONLY")
			fmt.Println()
			fmt.Printf("  Reason: %s\n", result.Reason)
			fmt.Println()
			fmt.Println("  Use TodoWrite to track this implementation step.")
		}
	}
	return nil
}
