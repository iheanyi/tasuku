// Package agentsmd provides CLI commands for managing AGENTS.md file health.
package agentsmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iheanyi/tasuku/internal/mdlint"
	"github.com/spf13/cobra"
)

const (
	defaultMaxLines     = 200 // Recommended max lines for AGENTS.md
	defaultWarnLines    = 150 // Warn when approaching limit
	defaultSectionLimit = 50  // Max lines per section before suggesting split
)

func newAgentsMdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agentsmd",
		Short: "Manage AGENTS.md file health",
		Long: `Analyze AGENTS.md to keep context minimal and well-organized.

Large AGENTS.md files consume tokens and slow down agent comprehension.
This tool helps identify bloat and organize content within the file.

Subcommands:
  lint     - Check AGENTS.md health, warn if too large
  stats    - Show section breakdown with line counts
  organize - Interactive guided workflow to organize AGENTS.md`,
	}

	cmd.AddCommand(newLintCmd())
	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newOrganizeCmd())

	return cmd
}

// Cmd is the parent command for agentsmd operations.
var Cmd = newAgentsMdCmd()

func newLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Check AGENTS.md health and warn about bloat",
		Long: `Analyze AGENTS.md and report on its health.

Checks:
  - Total line count (warns if > 150, errors if > 200)
  - Section sizes (warns about sections > 50 lines)
  - Suggests reorganizing large sections

Exit codes:
  0 - All good
  1 - Warnings (approaching limits)
  2 - Problems found (too large)

Examples:
  tk agentsmd lint                # Check default AGENTS.md
  tk agentsmd lint --file ./docs/AGENTS.md   # Check specific file
  tk agentsmd lint --max-lines 300           # Custom threshold`,
		RunE: runLint,
	}

	cmd.Flags().String("file", "AGENTS.md", "Path to AGENTS.md file")
	cmd.Flags().Int("max-lines", defaultMaxLines, "Maximum recommended lines")
	cmd.Flags().Int("warn-lines", defaultWarnLines, "Line count to start warning")
	cmd.Flags().Int("section-limit", defaultSectionLimit, "Max lines per section")

	return cmd
}

func newStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show AGENTS.md section breakdown",
		Long: `Display a breakdown of AGENTS.md by section with line counts.

Helps identify which sections are candidates for reorganization.

Examples:
  tk agentsmd stats
  tk agentsmd stats --file ./docs/AGENTS.md`,
		RunE: runStats,
	}

	cmd.Flags().String("file", "AGENTS.md", "Path to AGENTS.md file")

	return cmd
}

func runLint(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	maxLines, _ := cmd.Flags().GetInt("max-lines")
	warnLines, _ := cmd.Flags().GetInt("warn-lines")
	sectionLimit, _ := cmd.Flags().GetInt("section-limit")

	cfg := mdlint.Config{
		MaxLines:     maxLines,
		WarnLines:    warnLines,
		SectionLimit: sectionLimit,
		FileName:     filepath.Base(filePath),
		RulesDir:     "", // no rules dir for AGENTS.md
		StatsCmd:     "tk agentsmd stats",
	}

	result, err := mdlint.Lint(filePath, cfg)
	if err != nil {
		return err
	}

	// File not found
	if result.TotalLines == 0 && result.Status == "ok" && len(result.Sections) == 0 {
		if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
			fmt.Printf("✓ No %s found (nothing to lint)\n", filePath)
			return nil
		}
	}

	// Report total lines
	fmt.Printf("📄 %s: %d lines\n", filePath, result.TotalLines)

	switch result.Status {
	case "error":
		fmt.Printf("   ❌ Exceeds recommended maximum (%d lines)\n", maxLines)
	case "warning":
		fmt.Printf("   ⚠️  Approaching limit (%d/%d lines)\n", result.TotalLines, maxLines)
	default:
		fmt.Printf("   ✓ Within recommended size\n")
	}

	// Large sections
	if len(result.LargeSections) > 0 {
		fmt.Printf("\n⚠️  Large sections (>%d lines) - consider reorganizing:\n", sectionLimit)
		for _, s := range result.LargeSections {
			fmt.Printf("   - %s (%d lines)\n", s.Name, s.Lines)
		}
	}

	// Suggestions
	if result.Status != "ok" || len(result.LargeSections) > 0 {
		fmt.Println("\n💡 Suggestions:")
		for i, r := range result.Recommendations {
			fmt.Printf("   %d. %s\n", i+1, r)
		}
	} else {
		fmt.Println("\n✓ AGENTS.md is well-organized!")
	}

	if result.Status == "error" {
		return fmt.Errorf("%s exceeds recommended maximum (%d lines)", filePath, maxLines)
	}

	return nil
}

func runStats(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")

	cfg := mdlint.Config{
		FileName: filepath.Base(filePath),
	}

	result, err := mdlint.Stats(filePath, cfg)
	if err != nil {
		return err
	}

	fmt.Printf("📄 %s Section Breakdown\n", filePath)
	fmt.Println(strings.Repeat("─", 50))

	for _, s := range result.Sections {
		bar := strings.Repeat("█", int(s.Percentage/5))
		fmt.Printf("%-30s %4d lines (%4.1f%%) %s\n", s.Name, s.Lines, s.Percentage, bar)
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("%-30s %4d lines (100%%)\n", "TOTAL", result.TotalLines)

	return nil
}

func newOrganizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "organize",
		Short: "Interactive guided workflow to organize AGENTS.md",
		Long: `Guided workflow to organize AGENTS.md content and structure.

This command:
  1. Shows current state (line count, sections)
  2. Identifies large sections that may need reorganization
  3. Provides an interactive menu to reorganize sections
  4. Can reorder sections, split large ones, or suggest edits

This helps keep AGENTS.md focused and well-structured for agents.

Example:
  tk agentsmd organize`,
		RunE: runOrganize,
	}

	cmd.Flags().String("file", "AGENTS.md", "Path to AGENTS.md file")
	cmd.Flags().Int("threshold", defaultSectionLimit, "Minimum lines to flag for reorganization")

	return cmd
}

func runOrganize(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	threshold, _ := cmd.Flags().GetInt("threshold")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("✓ No %s found - nothing to organize\n", filePath)
		return nil
	}

	// Analyze current state
	totalLines, sections, err := mdlint.AnalyzeFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to analyze %s: %w", filePath, err)
	}

	// Show current state
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║               AGENTS.md Organization Assistant               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Status indicators
	statusIcon := "✓"
	if totalLines > defaultMaxLines {
		statusIcon = "❌"
	} else if totalLines > defaultWarnLines {
		statusIcon = "⚠️"
	}

	fmt.Printf("📄 %s %s: %d lines\n", filePath, statusIcon, totalLines)
	fmt.Printf("   %d sections found\n", len(sections))
	fmt.Println()

	// Find large sections
	var largeSections []mdlint.Section
	for _, s := range sections {
		if s.Lines >= threshold {
			largeSections = append(largeSections, s)
		}
	}

	if len(largeSections) == 0 {
		fmt.Println("✓ All sections are well-sized - AGENTS.md is well-organized!")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  'v' = view all sections")
		fmt.Println("  'q' = quit")
		fmt.Println()
		fmt.Print("> ")

		var input string
		fmt.Scanln(&input)

		if input == "v" {
			fmt.Println()
			fmt.Println("All sections:")
			for i, s := range sections {
				fmt.Printf("  %d. %s (%d lines)\n", i+1, s.Name, s.Lines)
			}
		}
		return nil
	}

	// Show large sections
	fmt.Printf("📋 Large sections (>%d lines) that may need reorganization:\n\n", threshold)
	for i, s := range largeSections {
		fmt.Printf("  %d. %s (%d lines)\n", i+1, s.Name, s.Lines)
	}
	fmt.Println()

	// Interactive menu
	fmt.Println("Options:")
	fmt.Println("  'v' = view all sections")
	fmt.Println("  's' = show statistics")
	fmt.Println("  Enter number to see details of a large section")
	fmt.Println("  'q' = quit without changes")
	fmt.Println()
	fmt.Print("> ")

	var input string
	fmt.Scanln(&input)

	if input == "q" || input == "" {
		fmt.Println("No changes made.")
		return nil
	}

	if input == "v" {
		fmt.Println()
		fmt.Println("All sections:")
		for i, s := range sections {
			fmt.Printf("  %d. %s (%d lines)\n", i+1, s.Name, s.Lines)
		}
		return nil
	}

	if input == "s" {
		fmt.Println()
		fmt.Printf("📄 %s Section Breakdown\n", filePath)
		fmt.Println(strings.Repeat("─", 50))

		// Sort sections by line count (descending)
		sortedSections := make([]mdlint.Section, len(sections))
		copy(sortedSections, sections)
		sort.Slice(sortedSections, func(i, j int) bool {
			return sortedSections[i].Lines > sortedSections[j].Lines
		})

		for _, s := range sortedSections {
			pct := float64(s.Lines) / float64(totalLines) * 100
			bar := strings.Repeat("█", int(pct/5))
			fmt.Printf("%-30s %4d lines (%4.1f%%) %s\n", s.Name, s.Lines, pct, bar)
		}

		fmt.Println(strings.Repeat("─", 50))
		fmt.Printf("%-30s %4d lines (100%%)\n", "TOTAL", totalLines)
		return nil
	}

	// Try to parse as section number
	var num int
	if _, err := fmt.Sscanf(input, "%d", &num); err == nil && num > 0 && num <= len(largeSections) {
		s := largeSections[num-1]
		fmt.Println()
		fmt.Printf("Section: %s\n", s.Name)
		fmt.Printf("Lines: %d (lines %d-%d)\n", s.Lines, s.StartLine, s.EndLine)
		fmt.Println()
		fmt.Println("Suggestions for reorganization:")
		fmt.Println("  - Consider splitting this section into smaller subsections")
		fmt.Println("  - Move detailed content to separate reference files")
		fmt.Println("  - Keep only essential context in AGENTS.md")
	}

	return nil
}
