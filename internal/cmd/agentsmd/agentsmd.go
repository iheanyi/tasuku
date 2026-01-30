// Package agentsmd provides CLI commands for managing AGENTS.md file health.
package agentsmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

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
  lint   - Check AGENTS.md health, warn if too large
  stats  - Show section breakdown with line counts`,
	}

	cmd.AddCommand(newLintCmd())
	cmd.AddCommand(newStatsCmd())

	return cmd
}

// Cmd is the parent command for agentsmd operations.
var Cmd = newAgentsMdCmd()

type section struct {
	name      string
	startLine int
	endLine   int
	lines     int
}

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

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("✓ No %s found (nothing to lint)\n", filePath)
		return nil
	}

	// Count lines
	totalLines, sections, err := analyzeFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to analyze %s: %w", filePath, err)
	}

	hasProblems := false
	hasWarnings := false

	// Report total lines
	fmt.Printf("📄 %s: %d lines\n", filePath, totalLines)

	if totalLines > maxLines {
		fmt.Printf("   ❌ Exceeds recommended maximum (%d lines)\n", maxLines)
		hasProblems = true
	} else if totalLines > warnLines {
		fmt.Printf("   ⚠️  Approaching limit (%d/%d lines)\n", totalLines, maxLines)
		hasWarnings = true
	} else {
		fmt.Printf("   ✓ Within recommended size\n")
	}

	// Check for large sections
	largeSections := []section{}
	for _, s := range sections {
		if s.lines > sectionLimit {
			largeSections = append(largeSections, s)
		}
	}

	if len(largeSections) > 0 {
		fmt.Printf("\n⚠️  Large sections (>%d lines) - consider reorganizing:\n", sectionLimit)
		for _, s := range largeSections {
			fmt.Printf("   - %s (%d lines)\n", s.name, s.lines)
		}
		hasWarnings = true
	}

	// Suggestions
	if hasProblems || hasWarnings {
		fmt.Println("\n💡 Suggestions:")
		fmt.Println("   1. Keep AGENTS.md focused on overview and key decisions")
		fmt.Println("   2. Move detailed reference docs to separate files")
		fmt.Println("   3. Use 'tk agentsmd stats' to see full breakdown")
	} else {
		fmt.Println("\n✓ AGENTS.md is well-organized!")
	}

	// Exit with appropriate code
	if hasProblems {
		os.Exit(2)
	} else if hasWarnings {
		os.Exit(1)
	}

	return nil
}

func runStats(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", filePath)
	}

	totalLines, sections, err := analyzeFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to analyze %s: %w", filePath, err)
	}

	fmt.Printf("📄 %s Section Breakdown\n", filePath)
	fmt.Println(strings.Repeat("─", 50))

	// Sort sections by line count (descending)
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].lines > sections[j].lines
	})

	for _, s := range sections {
		pct := float64(s.lines) / float64(totalLines) * 100
		bar := strings.Repeat("█", int(pct/5))
		fmt.Printf("%-30s %4d lines (%4.1f%%) %s\n", s.name, s.lines, pct, bar)
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("%-30s %4d lines (100%%)\n", "TOTAL", totalLines)

	return nil
}

// analyzeFile parses a markdown file and returns line count and sections
func analyzeFile(path string) (int, []section, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()

	var sections []section
	var currentSection *section

	lineNum := 0
	scanner := bufio.NewScanner(file)
	h2Pattern := regexp.MustCompile(`^##\s+(.+)$`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for H2 header (## Section)
		if matches := h2Pattern.FindStringSubmatch(line); len(matches) > 1 {
			// Close previous section
			if currentSection != nil {
				currentSection.endLine = lineNum - 1
				currentSection.lines = currentSection.endLine - currentSection.startLine + 1
				sections = append(sections, *currentSection)
			}

			// Start new section
			currentSection = &section{
				name:      matches[1],
				startLine: lineNum,
			}
		}
	}

	// Close final section
	if currentSection != nil {
		currentSection.endLine = lineNum
		currentSection.lines = currentSection.endLine - currentSection.startLine + 1
		sections = append(sections, *currentSection)
	}

	if err := scanner.Err(); err != nil {
		return 0, nil, err
	}

	return lineNum, sections, nil
}
