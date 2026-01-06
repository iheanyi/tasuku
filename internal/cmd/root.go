// Package cmd provides the CLI commands for tasuku.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/cmd/config"
	contextcmd "github.com/iheanyi/tasuku/internal/cmd/context"
	"github.com/iheanyi/tasuku/internal/cmd/decision"
	"github.com/iheanyi/tasuku/internal/cmd/hooks"
	"github.com/iheanyi/tasuku/internal/cmd/learning"
	"github.com/iheanyi/tasuku/internal/cmd/mcpcmd"
	"github.com/iheanyi/tasuku/internal/cmd/migrate"
	"github.com/iheanyi/tasuku/internal/cmd/note"
	"github.com/iheanyi/tasuku/internal/cmd/pr"
	"github.com/iheanyi/tasuku/internal/cmd/serve"
	"github.com/iheanyi/tasuku/internal/cmd/skills"
	taskcmd "github.com/iheanyi/tasuku/internal/cmd/task"
	"github.com/iheanyi/tasuku/internal/cmd/ui"
	"github.com/iheanyi/tasuku/internal/mcp"
	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

const Version = "0.3.0"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tk",
		Short: "Tasuku - agent-first task management",
		Long: `tk is an agent-first task management system designed for AI agents
working on codebases.

Design Principles:
  - Pull over push: Agents query when needed, no constant injections
  - Parallel-safe: File locking for multiple simultaneous agents
  - Minimal context: Only load what's needed for the current task
  - Human-readable: JSON file that can be edited by hand

Getting Started:
  tk init                  # Create .tasuku/ directory
  tk task add "My task"    # Add a task
  tk task list             # View all tasks
  tk task start <id>       # Begin working on a task
  tk task done <id>        # Mark task complete

AI Tool Integration:
  tk mcp install           # Auto-configure MCP for Claude Code/Cursor
  tk serve mcp             # Start MCP server (for AI tools)

For full documentation: https://github.com/iheanyi/tasuku`,
		Version: Version,
	}

	// Global flags
	cmd.PersistentFlags().StringVarP(&config.OutputFormat, "format", "f", "table", "Output format: table, json, yaml")

	// Register all subcommands
	cmd.AddCommand(taskcmd.Cmd)
	cmd.AddCommand(learning.Cmd)
	cmd.AddCommand(decision.Cmd)
	cmd.AddCommand(note.Cmd)
	cmd.AddCommand(contextcmd.Cmd)
	cmd.AddCommand(serve.Cmd)
	cmd.AddCommand(mcpcmd.Cmd)
	cmd.AddCommand(hooks.Cmd)
	cmd.AddCommand(migrate.Cmd)
	cmd.AddCommand(pr.Cmd)
	cmd.AddCommand(ui.Cmd)
	cmd.AddCommand(skills.Cmd)

	// Root-level commands
	cmd.AddCommand(initCmd)
	cmd.AddCommand(doctorCmd)
	cmd.AddCommand(newValidateCmd())

	// Shortcut commands (documented in CLAUDE.md)
	cmd.AddCommand(newLearnShortcutCmd())
	cmd.AddCommand(newDecideShortcutCmd())
	cmd.AddCommand(newHealthCmd())
	cmd.AddCommand(newSuggestCmd())

	return cmd
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate Tasuku storage for correctness",
		Long: `Validate the Tasuku storage for correctness.

Checks performed:
  - Version is supported
  - All tasks have non-empty descriptions
  - All tasks have valid statuses
  - No circular dependencies in blocked_by relationships
  - Referenced blockers exist

This is the same as 'tk context validate'.

Examples:
  tk validate              # Validate storage
  tk validate --format json  # Output as JSON`,
		RunE: contextcmd.RunValidate,
	}
}

// RootCmd is the base command for tk
var RootCmd = newRootCmd()

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Tasuku in current directory",
	Long: `Initialize a new Tasuku project in the current directory.

Creates a .tasuku/ directory with:
  - tasks/     Individual task JSON files (one per task)
  - archive/   Completed tasks that have been archived
  - context/   Learnings and decisions

Benefits:
  - One file per task = cleaner git diffs, fewer merge conflicts
  - Human-readable JSON, can be edited directly
  - Safe for multiple agents working in parallel

If you have a legacy .tasuku.json file, use 'tk migrate v3' to upgrade.

Examples:
  tk init                    # Create .tasuku/ directory
  tk init && tk task add "Setup"  # Initialize and add first task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already initialized
		storageType := store.DetectStorageType(".")
		if storageType != store.StorageTypeNone {
			if storageType == store.StorageTypeDir {
				return fmt.Errorf(".tasuku/ directory already exists")
			}
			return fmt.Errorf(".tasuku.json already exists - run 'tk migrate v3' to upgrade")
		}

		s := store.NewDirStore(store.DirName)
		if err := s.Init(); err != nil {
			return err
		}
		fmt.Println("Created .tasuku/ directory")
		fmt.Println("  tasks/    - Your task files")
		fmt.Println("  archive/  - Archived completed tasks")
		fmt.Println("  context/  - Learnings and decisions")
		fmt.Println()
		fmt.Println("Tip: Commit .tasuku/ to git so tasks travel with your code.")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  tk task add \"Your first task\"")
		fmt.Println("  tk hooks install              # Optional: git hooks")
		fmt.Println("  tk mcp install                # Optional: AI tool integration (Claude Code, Cursor)")
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose Tasuku setup and MCP configuration",
	Long: `Check your Tasuku installation and MCP configuration for common issues.

This command verifies:
  - tk binary is accessible and shows its location
  - Tasuku storage exists (.tasuku/ directory or .tasuku.json file)
  - MCP is configured in Claude Code, Cursor, and other AI tools
  - The configured binary path matches the current tk installation
  - The MCP server can start and respond to requests

Run this when Tasuku tools aren't appearing in your AI assistant.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	fmt.Println("Tasuku Doctor")
	fmt.Println("=============")
	fmt.Println()

	hasErrors := false

	// 1. Check tk binary
	executable, err := os.Executable()
	if err != nil {
		fmt.Println("✗ Could not determine tk binary location")
		hasErrors = true
	} else {
		fmt.Printf("✓ tk binary: %s\n", executable)
	}

	// 2. Check Tasuku storage
	s := store.DefaultStorageWithWarning()
	tasukuPath := s.Path()
	if !s.Exists() {
		fmt.Printf("✗ No Tasuku storage found (searched from %s)\n", mustGetwd())
		fmt.Println("  Run 'tk init' to create .tasuku/ directory")
		hasErrors = true
	} else {
		fmt.Printf("✓ Tasuku storage: %s\n", tasukuPath)
	}

	fmt.Println()
	fmt.Println("MCP Configuration")
	fmt.Println("-----------------")

	// 3. Check AI tool configurations
	tools := getSupportedAITools()
	configuredTools := 0
	mismatchedPaths := []string{}

	for _, tool := range tools {
		// Check if settings file exists
		if _, err := os.Stat(tool.SettingsPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(tool.SettingsPath)
		if err != nil {
			continue
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			continue
		}

		mcpServers, ok := settings[tool.MCPKey].(map[string]interface{})
		if !ok {
			continue
		}

		tasukuConfig, exists := mcpServers["tasuku"].(map[string]interface{})
		if !exists {
			fmt.Printf("✗ %s: MCP not configured\n", tool.Name)
			fmt.Printf("  Run 'tk mcp install' to configure\n")
			hasErrors = true
			continue
		}

		configuredPath, _ := tasukuConfig["command"].(string)
		configuredArgs, _ := tasukuConfig["args"].([]interface{})

		// Check if path matches current executable
		if configuredPath != executable {
			fmt.Printf("⚠ %s: configured but path mismatch\n", tool.Name)
			fmt.Printf("  Configured: %s\n", configuredPath)
			fmt.Printf("  Current:    %s\n", executable)
			mismatchedPaths = append(mismatchedPaths, tool.Name)
		} else {
			argsStr := ""
			for _, arg := range configuredArgs {
				if s, ok := arg.(string); ok {
					argsStr += s + " "
				}
			}
			fmt.Printf("✓ %s: configured (%s %s)\n", tool.Name, filepath.Base(configuredPath), strings.TrimSpace(argsStr))
		}
		configuredTools++
	}

	if configuredTools == 0 {
		fmt.Println("✗ No AI tools have Tasuku MCP configured")
		fmt.Println("  Run 'tk mcp install' to auto-configure")
		hasErrors = true
	}

	// 4. Test MCP server
	fmt.Println()
	fmt.Println("MCP Server Test")
	fmt.Println("---------------")

	// Quick test: can we create a server and get tools?
	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		mcpTools := mcpServer.Tools()
		fmt.Printf("✓ MCP server responds with %d tools\n", len(mcpTools))

		// List a few tools
		if len(mcpTools) > 0 {
			toolNames := []string{}
			for _, t := range mcpTools {
				toolNames = append(toolNames, t.Name)
			}
			if len(toolNames) > 5 {
				toolNames = toolNames[:5]
				fmt.Printf("  Tools: %s, ... (+%d more)\n", strings.Join(toolNames, ", "), len(mcpTools)-5)
			} else {
				fmt.Printf("  Tools: %s\n", strings.Join(toolNames, ", "))
			}
		}
	} else {
		fmt.Println("⚠ Cannot test MCP server (no Tasuku storage)")
	}

	// 5. Check CLI/MCP parity
	fmt.Println()
	fmt.Println("CLI/MCP Parity")
	fmt.Println("--------------")

	if _, err := os.Stat(tasukuPath); err == nil {
		mcpServer := mcp.New(s)
		mcpTools := mcpServer.Tools()

		// Build set of MCP tool names
		mcpToolSet := make(map[string]bool)
		for _, t := range mcpTools {
			mcpToolSet[t.Name] = true
		}

		// Define expected MCP tools for CLI commands
		cliToMCP := map[string][]string{
			"task list":     {"tk_list"},
			"task add":      {"tk_add"},
			"task show":     {"tk_show"},
			"task start":    {"tk_start"},
			"task done":     {"tk_done"},
			"task block":    {"tk_block"},
			"task unblock":  {"tk_unblock"},
			"task pause":    {"tk_pause"},
			"task find":     {"tk_find"},
			"task priority": {"tk_priority"},
			"task delete":   {"tk_delete"},
			"task edit":     {"tk_edit"},
			"task owner":    {"tk_owner"},
			"task claim":    {"tk_claim"},
			"task release":  {"tk_release"},
			"task tag":      {"tk_tag_add", "tk_tag_remove"},
			"task field":    {"tk_field_set", "tk_field_remove"},
			"task timer":    {"tk_timer_start", "tk_timer_stop", "tk_timer_status"},
			"task archive":  {"tk_archive", "tk_archive_restore", "tk_archive_list"},
			"context learn": {"tk_learn"},
			"context decide":{"tk_decide"},
			"context note":  {"tk_note"},
			"context show":  {"tk_context"},
		}

		missingTools := []string{}
		for cli, expectedTools := range cliToMCP {
			for _, tool := range expectedTools {
				if !mcpToolSet[tool] {
					missingTools = append(missingTools, fmt.Sprintf("%s (missing %s)", cli, tool))
				}
			}
		}

		if len(missingTools) == 0 {
			fmt.Printf("✓ All %d CLI commands have corresponding MCP tools\n", len(cliToMCP))
		} else {
			fmt.Printf("✗ %d CLI commands missing MCP tools:\n", len(missingTools))
			for _, m := range missingTools {
				fmt.Printf("  - %s\n", m)
			}
			hasErrors = true
		}
	} else {
		fmt.Println("⚠ Cannot check parity (no Tasuku storage)")
	}

	// Summary
	fmt.Println()
	if hasErrors {
		fmt.Println("Issues found. See recommendations above.")
		if configuredTools > 0 {
			fmt.Println()
			fmt.Println("If MCP is configured but tools aren't visible:")
			fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
			fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")
		}
		return nil
	}

	if len(mismatchedPaths) > 0 {
		fmt.Println("Configuration path mismatch detected.")
		fmt.Println("Run 'tk mcp install' to update configuration.")
		return nil
	}

	fmt.Println("Everything looks good!")
	fmt.Println()
	fmt.Println("If tools still aren't visible in your AI assistant:")
	fmt.Println("  1. Restart your AI tool (Claude Code, Cursor, etc.)")
	fmt.Println("  2. Run '/mcp' in Claude Code to check MCP status")

	return nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// newLearnShortcutCmd creates a top-level shortcut for adding learnings.
// This allows "tk learn 'insight'" instead of "tk learning add 'insight'".
func newLearnShortcutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn \"insight\"",
		Short: "Record an insight (shortcut for 'tk learning add')",
		Long: `Record an insight or knowledge discovered during work.
This is a shortcut for 'tk learning add'.

Examples:
  tk learn "Redis connection pooling significantly improves API latency"
  tk learn "The auth middleware must run before rate limiting" --permanent
  tk learn "Never use raw SQL queries" --rule`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			learningText := args[0]
			permanent, _ := cmd.Flags().GetBool("permanent")
			forceRule, _ := cmd.Flags().GetBool("rule")
			s := store.DefaultStorageWithWarning()

			var id string
			var isRule bool
			var err error

			if forceRule {
				ruleVal := true
				id, isRule, err = s.AddLearningWithRule(learningText, &ruleVal)
			} else {
				id, isRule, err = s.AddLearningWithRule(learningText, nil)
			}
			if err != nil {
				return err
			}

			if permanent {
				if err := appendLearningToCLAUDEmd(learningText); err != nil {
					fmt.Printf("Warning: could not append to CLAUDE.md: %v\n", err)
				} else {
					fmt.Printf("Learning added [%s] (also appended to CLAUDE.md)\n", id)
					return nil
				}
			}

			if isRule {
				fmt.Printf("Learning added [%s] [RULE]\n", id)
				fmt.Println("Hint: Promote rules to permanent docs with: tk learning promote", id)
			} else {
				fmt.Printf("Learning added [%s]\n", id)
			}
			return nil
		},
	}

	cmd.Flags().Bool("permanent", false, "Also append learning to CLAUDE.md")
	cmd.Flags().Bool("rule", false, "Explicitly mark this learning as a rule")

	return cmd
}

// newDecideShortcutCmd creates a top-level shortcut for adding decisions.
// This allows "tk decide --id X --chose Y --because Z" instead of "tk decision add ...".
func newDecideShortcutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decide --id <id> --chose <option> --because <reason>",
		Short: "Record a decision (shortcut for 'tk decision add')",
		Long: `Document an architectural or design decision.
This is a shortcut for 'tk decision add'.

Examples:
  tk decide --id db-choice --chose PostgreSQL --over MySQL,SQLite --because "Better JSON support"
  tk decide --id auth-method --chose JWT --over sessions --because "Stateless and scalable"
  tk decide --id framework --chose Cobra --because "Standard Go CLI library"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			chose, _ := cmd.Flags().GetString("chose")
			alternatives, _ := cmd.Flags().GetStringSlice("over")
			because, _ := cmd.Flags().GetString("because")

			if id == "" || chose == "" || because == "" {
				return fmt.Errorf("usage: tk decide --id <id> --chose <choice> --over <options> --because <reason>")
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
		},
	}

	cmd.Flags().String("id", "", "Decision ID")
	cmd.Flags().String("chose", "", "The option chosen")
	cmd.Flags().StringSlice("over", nil, "Alternatives considered (repeatable or comma-separated)")
	cmd.Flags().String("because", "", "Reasoning")
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("chose")
	cmd.MarkFlagRequired("because")

	return cmd
}

// appendLearningToCLAUDEmd appends a learning to CLAUDE.md
func appendLearningToCLAUDEmd(content string) error {
	claudePath := "CLAUDE.md"
	existing, err := os.ReadFile(claudePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	text := string(existing)
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
		text = text + "\n\n## Learnings\n\n" + entry
	}

	return os.WriteFile(claudePath, []byte(text), 0644)
}

// AITool represents a supported AI tool configuration
type AITool struct {
	Name         string
	SettingsPath string
	MCPKey       string
}

func getSupportedAITools() []AITool {
	home, _ := os.UserHomeDir()
	return []AITool{
		{"Claude Code", home + "/.claude.json", "mcpServers"},
		{"Cursor", home + "/.cursor/mcp.json", "mcpServers"},
		{"Cursor (alt)", home + "/Library/Application Support/Cursor/User/globalStorage/mcp.json", "mcpServers"},
	}
}

// HealthReport represents the project health check result
type HealthReport struct {
	HealthScore   int            `json:"health_score" yaml:"health_score"`
	HealthStatus  string         `json:"health_status" yaml:"health_status"`
	TaskCounts    map[string]int `json:"task_counts" yaml:"task_counts"`
	PriorityCounts map[string]int `json:"priority_counts" yaml:"priority_counts"`
	Issues        HealthIssues   `json:"issues" yaml:"issues"`
	Recommendations []string     `json:"recommendations" yaml:"recommendations"`
	LearningsCount int           `json:"learnings_count" yaml:"learnings_count"`
	DecisionsCount int           `json:"decisions_count" yaml:"decisions_count"`
}

// HealthIssues represents issues found in the health check
type HealthIssues struct {
	StaleInProgress     []string `json:"stale_in_progress,omitempty" yaml:"stale_in_progress,omitempty"`
	HighPriorityBlocked []string `json:"high_priority_blocked,omitempty" yaml:"high_priority_blocked,omitempty"`
	LongRunningTimers   []string `json:"long_running_timers,omitempty" yaml:"long_running_timers,omitempty"`
	StaleDoneCount      int      `json:"stale_done_count" yaml:"stale_done_count"`
	RuleLearnings       int      `json:"rule_learnings" yaml:"rule_learnings"`
}

// newHealthCmd creates the health check command
func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check project health and get recommendations",
		Long: `Perform a project health check with actionable recommendations.

Checks for:
  - Stale in-progress tasks (not updated in 24h)
  - High-priority blocked tasks
  - Long-running timers (4+ hours)
  - Old done tasks ready for archival
  - Rule learnings ready for promotion

Examples:
  tk health              # Show health report
  tk health -f json      # Output as JSON
  tk health -f yaml      # Output as YAML`,
		RunE: runHealth,
	}
}

func runHealth(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()
	f, err := s.Read()
	if err != nil {
		return err
	}

	report := computeHealth(f)
	return outputHealth(report)
}

func computeHealth(f *task.File) HealthReport {
	now := time.Now()

	statusCounts := map[string]int{}
	priorityCounts := map[string]int{}
	var staleInProgress []string
	var staleDone []string
	var longRunningTimers []string
	var highPriorityBlocked []string

	for id, t := range f.Tasks {
		statusCounts[string(t.Status)]++

		switch t.GetPriority() {
		case task.PriorityCritical:
			priorityCounts["critical"]++
		case task.PriorityHigh:
			priorityCounts["high"]++
		case task.PriorityNormal:
			priorityCounts["normal"]++
		case task.PriorityLow:
			priorityCounts["low"]++
		case task.PriorityBacklog:
			priorityCounts["backlog"]++
		}

		// Stale in_progress (>24h)
		if t.Status == task.StatusInProgress && now.Sub(t.UpdatedAt) > 24*time.Hour {
			staleInProgress = append(staleInProgress, id)
		}

		// Stale done tasks (>7 days)
		if t.Status == task.StatusDone && now.Sub(t.UpdatedAt) > 7*24*time.Hour {
			staleDone = append(staleDone, id)
		}

		// High priority blocked
		if t.Status == task.StatusBlocked && t.GetPriority() <= task.PriorityHigh {
			highPriorityBlocked = append(highPriorityBlocked, id)
		}

		// Long-running timers
		if t.IsTimerRunning() && t.TimerStart != nil && now.Sub(*t.TimerStart) > 4*time.Hour {
			longRunningTimers = append(longRunningTimers, id)
		}
	}

	// Count rule learnings
	ruleCount := 0
	for _, l := range f.Context.Learnings {
		if l.IsRule {
			ruleCount++
		}
	}

	// Build recommendations
	var recommendations []string

	if len(staleInProgress) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("STALE: %d in_progress task(s) not updated in 24h: %v - update or pause them",
				len(staleInProgress), staleInProgress))
	}

	if len(highPriorityBlocked) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("BLOCKED: %d high-priority task(s) blocked: %v - unblock to make progress",
				len(highPriorityBlocked), highPriorityBlocked))
	}

	if len(longRunningTimers) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("TIMERS: %d timer(s) running 4+ hours: %v - stop if not active",
				len(longRunningTimers), longRunningTimers))
	}

	if len(staleDone) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("ARCHIVE: %d done task(s) older than 7 days: consider archiving with 'tk task archive add'",
				len(staleDone)))
	}

	if ruleCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("PROMOTE: %d rule learning(s) ready for promotion to docs - use 'tk learning list --rules'",
				ruleCount))
	}

	// Calculate health score
	healthScore := 100
	healthScore -= len(staleInProgress) * 10
	healthScore -= len(highPriorityBlocked) * 15
	healthScore -= len(longRunningTimers) * 5
	if healthScore < 0 {
		healthScore = 0
	}

	var healthStatus string
	if healthScore >= 80 {
		healthStatus = "healthy"
	} else if healthScore >= 50 {
		healthStatus = "needs attention"
	} else {
		healthStatus = "unhealthy"
	}

	return HealthReport{
		HealthScore:  healthScore,
		HealthStatus: healthStatus,
		TaskCounts: map[string]int{
			"total":       len(f.Tasks),
			"ready":       statusCounts["ready"],
			"in_progress": statusCounts["in_progress"],
			"blocked":     statusCounts["blocked"],
			"done":        statusCounts["done"],
		},
		PriorityCounts: priorityCounts,
		Issues: HealthIssues{
			StaleInProgress:     staleInProgress,
			HighPriorityBlocked: highPriorityBlocked,
			LongRunningTimers:   longRunningTimers,
			StaleDoneCount:      len(staleDone),
			RuleLearnings:       ruleCount,
		},
		Recommendations: recommendations,
		LearningsCount:  len(f.Context.Learnings),
		DecisionsCount:  len(f.Context.Decisions),
	}
}

func outputHealth(report HealthReport) error {
	switch config.OutputFormat {
	case "json":
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(report)
		fmt.Print(string(data))
	default:
		// Table format
		fmt.Println("Project Health Check")
		fmt.Println("====================")
		fmt.Println()

		// Health score with emoji
		var emoji string
		switch report.HealthStatus {
		case "healthy":
			emoji = "✓"
		case "needs attention":
			emoji = "⚠"
		default:
			emoji = "✗"
		}
		fmt.Printf("Health: %s %s (%d/100)\n", emoji, report.HealthStatus, report.HealthScore)
		fmt.Println()

		// Task counts
		fmt.Println("Tasks")
		fmt.Println("-----")
		fmt.Printf("  Total:       %d\n", report.TaskCounts["total"])
		fmt.Printf("  Ready:       %d\n", report.TaskCounts["ready"])
		fmt.Printf("  In Progress: %d\n", report.TaskCounts["in_progress"])
		fmt.Printf("  Blocked:     %d\n", report.TaskCounts["blocked"])
		fmt.Printf("  Done:        %d\n", report.TaskCounts["done"])
		fmt.Println()

		// Priority breakdown
		if len(report.PriorityCounts) > 0 {
			fmt.Println("Priority Breakdown")
			fmt.Println("------------------")
			for _, p := range []string{"critical", "high", "normal", "low", "backlog"} {
				if count, ok := report.PriorityCounts[p]; ok && count > 0 {
					fmt.Printf("  %-10s %d\n", p+":", count)
				}
			}
			fmt.Println()
		}

		// Context
		fmt.Println("Context")
		fmt.Println("-------")
		fmt.Printf("  Learnings:   %d\n", report.LearningsCount)
		fmt.Printf("  Decisions:   %d\n", report.DecisionsCount)
		fmt.Println()

		// Recommendations
		if len(report.Recommendations) > 0 {
			fmt.Println("Recommendations")
			fmt.Println("---------------")
			for _, rec := range report.Recommendations {
				fmt.Printf("  • %s\n", rec)
			}
		} else {
			fmt.Println("No issues found. Project is healthy!")
		}
	}
	return nil
}

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
