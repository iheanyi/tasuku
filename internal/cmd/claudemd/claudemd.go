// Package claudemd provides CLI commands for managing CLAUDE.md file health.
package claudemd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultMaxLines     = 200  // Recommended max lines for CLAUDE.md
	defaultWarnLines    = 150  // Warn when approaching limit
	defaultSectionLimit = 50   // Max lines per section before suggesting split
)

func newClaudeMdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claudemd",
		Short: "Manage CLAUDE.md file health and organization",
		Long: `Analyze and manage CLAUDE.md to keep context minimal and well-organized.

Large CLAUDE.md files consume tokens and slow down agent comprehension.
This tool helps identify bloat and suggests splits into .claude/rules/ modules.

Subcommands:
  lint   - Check CLAUDE.md health, warn if too large
  stats  - Show section breakdown with line counts

Example workflow:
  1. tk claudemd lint      # Check if CLAUDE.md needs attention
  2. tk claudemd stats     # See which sections are largest
  3. Manually move large sections to .claude/rules/`,
	}

	cmd.AddCommand(newLintCmd())
	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newSplitCmd())
	cmd.AddCommand(newOrganizeCmd())

	return cmd
}

// Cmd is the parent command for claudemd operations.
var Cmd = newClaudeMdCmd()

type section struct {
	name      string
	startLine int
	endLine   int
	lines     int
}

func newLintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Check CLAUDE.md health and warn about bloat",
		Long: `Analyze CLAUDE.md and report on its health.

Checks:
  - Total line count (warns if > 150, errors if > 200)
  - Section sizes (warns about sections > 50 lines)
  - Suggests moving large sections to .claude/rules/

Exit codes:
  0 - All good
  1 - Warnings (approaching limits)
  2 - Problems found (too large)

Examples:
  tk claudemd lint                # Check default CLAUDE.md
  tk claudemd lint --file ./docs/CLAUDE.md   # Check specific file
  tk claudemd lint --max-lines 300           # Custom threshold`,
		RunE: runLint,
	}

	cmd.Flags().String("file", "CLAUDE.md", "Path to CLAUDE.md file")
	cmd.Flags().Int("max-lines", defaultMaxLines, "Maximum recommended lines")
	cmd.Flags().Int("warn-lines", defaultWarnLines, "Line count to start warning")
	cmd.Flags().Int("section-limit", defaultSectionLimit, "Max lines per section")

	return cmd
}

func newStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show CLAUDE.md section breakdown",
		Long: `Display a breakdown of CLAUDE.md by section with line counts.

Helps identify which sections are candidates for extraction to .claude/rules/.

Examples:
  tk claudemd stats
  tk claudemd stats --file ./docs/CLAUDE.md`,
		RunE: runStats,
	}

	cmd.Flags().String("file", "CLAUDE.md", "Path to CLAUDE.md file")

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

	// Count rules files
	rulesCount := countRulesFiles()

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

	// Report on rules modules
	if rulesCount > 0 {
		fmt.Printf("\n📁 .claude/rules/: %d module files found\n", rulesCount)
	}

	// Check for large sections
	largeSections := []section{}
	for _, s := range sections {
		if s.lines > sectionLimit {
			largeSections = append(largeSections, s)
		}
	}

	if len(largeSections) > 0 {
		fmt.Printf("\n⚠️  Large sections (>%d lines) - consider moving to .claude/rules/:\n", sectionLimit)
		for _, s := range largeSections {
			fmt.Printf("   - %s (%d lines)\n", s.name, s.lines)
		}
		hasWarnings = true
	}

	// Suggestions
	if hasProblems || hasWarnings {
		fmt.Println("\n💡 Suggestions:")
		fmt.Println("   1. Move reference sections (CLI, MCP, testing) to .claude/rules/")
		fmt.Println("   2. Keep CLAUDE.md focused on overview and key decisions")
		fmt.Println("   3. Use 'tk claudemd stats' to see full breakdown")
		fmt.Println("   4. Claude Code auto-loads all .claude/rules/*.md files")
	} else {
		fmt.Println("\n✓ CLAUDE.md is well-organized!")
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

	// Show rules files too
	rulesFiles := listRulesFiles()
	if len(rulesFiles) > 0 {
		fmt.Println()
		fmt.Println("📁 Modular rules files in .claude/rules/:")
		for _, f := range rulesFiles {
			lines := countFileLines(f)
			fmt.Printf("   - %s (%d lines)\n", filepath.Base(f), lines)
		}
	}

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

// countRulesFiles counts markdown files in .claude/rules/
func countRulesFiles() int {
	files := listRulesFiles()
	return len(files)
}

// listRulesFiles returns paths to all markdown files in .claude/rules/
func listRulesFiles() []string {
	var files []string

	rulesDir := ".claude/rules"
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		return files
	}

	filepath.Walk(rulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})

	return files
}

// countFileLines counts lines in a file
func countFileLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	lines := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines++
	}
	return lines
}

func newSplitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "split [section-name]",
		Short: "Extract a section from CLAUDE.md to .claude/rules/",
		Long: `Extract a section from CLAUDE.md to a separate rules file.

If no section name is provided, shows all sections and prompts for selection.

By default, the command:
  1. Extracts the section to .claude/rules/<section-name>.md
  2. Removes the section from CLAUDE.md
  3. Adds a reference in CLAUDE.md's Reference Documentation section

Use --extract-only to just extract without modifying CLAUDE.md.

Examples:
  tk claudemd split                    # Interactive section selection
  tk claudemd split "CLI Reference"    # Extract specific section
  tk claudemd split --list             # Just list sections without extracting
  tk claudemd split --extract-only     # Extract without modifying CLAUDE.md`,
		RunE: runSplit,
	}

	cmd.Flags().String("file", "CLAUDE.md", "Path to CLAUDE.md file")
	cmd.Flags().Bool("list", false, "Just list sections without extracting")
	cmd.Flags().String("output", "", "Output filename (default: kebab-case of section name)")
	cmd.Flags().Bool("extract-only", false, "Only extract, don't modify CLAUDE.md")

	return cmd
}

func runSplit(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	listOnly, _ := cmd.Flags().GetBool("list")
	outputName, _ := cmd.Flags().GetString("output")
	extractOnly, _ := cmd.Flags().GetBool("extract-only")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", filePath)
	}

	// Parse sections with their content
	totalLines, sections, sectionContents, err := analyzeFileWithContent(filePath)
	if err != nil {
		return fmt.Errorf("failed to analyze %s: %w", filePath, err)
	}

	if len(sections) == 0 {
		fmt.Println("No sections found in", filePath)
		return nil
	}

	// List mode
	if listOnly {
		fmt.Printf("Sections in %s (%d lines total):\n\n", filePath, totalLines)
		for i, s := range sections {
			fmt.Printf("  %d. %s (%d lines)\n", i+1, s.name, s.lines)
		}
		return nil
	}

	// Determine which section to extract
	var targetSection string
	if len(args) > 0 {
		targetSection = args[0]
	} else {
		// Show sections and let user choose
		fmt.Printf("Sections in %s:\n\n", filePath)
		for i, s := range sections {
			fmt.Printf("  %d. %s (%d lines)\n", i+1, s.name, s.lines)
		}
		fmt.Println()
		fmt.Println("Enter section number or name to extract (or 'q' to quit):")
		fmt.Print("> ")

		var input string
		fmt.Scanln(&input)

		if input == "q" || input == "" {
			return nil
		}

		// Try to parse as number
		var num int
		if _, err := fmt.Sscanf(input, "%d", &num); err == nil && num > 0 && num <= len(sections) {
			targetSection = sections[num-1].name
		} else {
			targetSection = input
		}
	}

	// Find matching section
	var found *section
	var content string
	for i, s := range sections {
		if strings.EqualFold(s.name, targetSection) || strings.Contains(strings.ToLower(s.name), strings.ToLower(targetSection)) {
			found = &sections[i]
			content = sectionContents[s.name]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("section %q not found", targetSection)
	}

	// Generate output filename
	if outputName == "" {
		outputName = toKebabCase(found.name) + ".md"
	}

	// Ensure .claude/rules/ exists
	rulesDir := ".claude/rules"
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", rulesDir, err)
	}

	outputPath := filepath.Join(rulesDir, outputName)

	// Check if output already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("%s already exists - use --output to specify a different name", outputPath)
	}

	// Write the extracted section
	header := fmt.Sprintf("# %s\n\n", found.name)
	if err := os.WriteFile(outputPath, []byte(header+content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	fmt.Printf("✓ Extracted '%s' (%d lines) to %s\n", found.name, found.lines, outputPath)

	// Unless extract-only, also update CLAUDE.md
	if !extractOnly {
		err := removeSectionAndAddReference(filePath, found, outputName)
		if err != nil {
			fmt.Printf("\n⚠️  Could not auto-update %s: %v\n", filePath, err)
			fmt.Println("   Please manually remove the section and add a reference.")
		} else {
			fmt.Printf("✓ Updated %s (removed section, added reference)\n", filePath)
		}
	} else {
		fmt.Println()
		fmt.Println("📝 Next steps (since --extract-only was used):")
		fmt.Printf("   1. Review %s\n", outputPath)
		fmt.Printf("   2. Remove the section from %s\n", filePath)
		fmt.Printf("   3. Add a reference in %s: \"See .claude/rules/%s\"\n", filePath, outputName)
	}

	return nil
}

// removeSectionAndAddReference removes a section from CLAUDE.md and adds a reference
func removeSectionAndAddReference(filePath string, sect *section, rulesFileName string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	// Find start and end of section (using 0-indexed)
	startIdx := sect.startLine - 1 // Convert to 0-indexed
	endIdx := sect.endLine - 1

	// Validate indices
	if startIdx < 0 || endIdx >= len(lines) || startIdx > endIdx {
		return fmt.Errorf("invalid section bounds: %d-%d (file has %d lines)", startIdx, endIdx, len(lines))
	}

	// Build new content: before section + after section
	var newLines []string
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, lines[endIdx+1:]...)

	// Write updated file
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return err
	}

	return nil
}

// analyzeFileWithContent parses a file and returns sections with their content
func analyzeFileWithContent(path string) (int, []section, map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, nil, err
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	var sections []section
	sectionContents := make(map[string]string)
	h2Pattern := regexp.MustCompile(`^##\s+(.+)$`)

	var currentSection *section
	var currentContent []string

	for lineNum, line := range lines {
		// Check for H2 header
		if matches := h2Pattern.FindStringSubmatch(line); len(matches) > 1 {
			// Save previous section
			if currentSection != nil {
				currentSection.endLine = lineNum
				currentSection.lines = currentSection.endLine - currentSection.startLine
				sections = append(sections, *currentSection)
				sectionContents[currentSection.name] = strings.Join(currentContent, "\n")
			}

			// Start new section
			currentSection = &section{
				name:      matches[1],
				startLine: lineNum + 1,
			}
			currentContent = []string{}
		} else if currentSection != nil {
			currentContent = append(currentContent, line)
		}
	}

	// Save final section
	if currentSection != nil {
		currentSection.endLine = totalLines
		currentSection.lines = currentSection.endLine - currentSection.startLine
		sections = append(sections, *currentSection)
		sectionContents[currentSection.name] = strings.Join(currentContent, "\n")
	}

	return totalLines, sections, sectionContents, nil
}

// toKebabCase converts a string to kebab-case
func toKebabCase(s string) string {
	// Remove special characters and convert to lowercase
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`[\s_]+`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func newOrganizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "organize",
		Short: "Interactive guided workflow to organize CLAUDE.md",
		Long: `Guided workflow to organize CLAUDE.md into modular rules files.

This command:
  1. Shows current state (line count, sections, existing rules)
  2. Recommends which sections to extract based on size
  3. Lets you select sections to extract interactively
  4. Handles extraction, removal, and reference updates automatically

This is the recommended way to reduce CLAUDE.md size while maintaining
all documentation through .claude/rules/ modules.

Example:
  tk claudemd organize`,
		RunE: runOrganize,
	}

	cmd.Flags().String("file", "CLAUDE.md", "Path to CLAUDE.md file")
	cmd.Flags().Int("threshold", defaultSectionLimit, "Minimum lines to recommend extraction")

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
	totalLines, sections, _, err := analyzeFileWithContent(filePath)
	if err != nil {
		return fmt.Errorf("failed to analyze %s: %w", filePath, err)
	}

	rulesFiles := listRulesFiles()

	// Show current state
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              CLAUDE.md Organization Assistant                ║")
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
	if len(rulesFiles) > 0 {
		fmt.Printf("📁 .claude/rules/: %d module files\n", len(rulesFiles))
	}
	fmt.Println()

	// Find sections that should be extracted
	var recommendations []section
	for _, s := range sections {
		if s.lines >= threshold {
			recommendations = append(recommendations, s)
		}
	}

	if len(recommendations) == 0 {
		fmt.Println("✓ No sections exceed the threshold - CLAUDE.md is well-organized!")
		return nil
	}

	// Show recommendations
	fmt.Printf("📋 Sections recommended for extraction (>%d lines):\n\n", threshold)
	for i, s := range recommendations {
		fmt.Printf("  %d. %s (%d lines)\n", i+1, s.name, s.lines)
	}
	fmt.Println()

	// Interactive extraction loop
	fmt.Println("Options:")
	fmt.Println("  Enter number(s) to extract (e.g., '1' or '1,2,3')")
	fmt.Println("  'a' = extract all recommended sections")
	fmt.Println("  'q' = quit without changes")
	fmt.Println()
	fmt.Print("> ")

	var input string
	fmt.Scanln(&input)

	if input == "q" || input == "" {
		fmt.Println("No changes made.")
		return nil
	}

	// Determine which sections to extract
	var toExtract []section
	if input == "a" {
		toExtract = recommendations
	} else {
		// Parse comma-separated numbers
		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			var num int
			if _, err := fmt.Sscanf(part, "%d", &num); err == nil && num > 0 && num <= len(recommendations) {
				toExtract = append(toExtract, recommendations[num-1])
			}
		}
	}

	if len(toExtract) == 0 {
		fmt.Println("No valid selections - no changes made.")
		return nil
	}

	fmt.Println()

	// Extract each section (in reverse order to preserve line numbers)
	// Sort by startLine descending
	sort.Slice(toExtract, func(i, j int) bool {
		return toExtract[i].startLine > toExtract[j].startLine
	})

	for _, s := range toExtract {
		outputName := toKebabCase(s.name) + ".md"
		rulesDir := ".claude/rules"
		outputPath := filepath.Join(rulesDir, outputName)

		// Skip if already exists
		if _, err := os.Stat(outputPath); err == nil {
			fmt.Printf("⏭️  Skipping '%s' - %s already exists\n", s.name, outputPath)
			continue
		}

		// Re-analyze to get fresh content (since file may have changed)
		_, currentSections, sectionContents, err := analyzeFileWithContent(filePath)
		if err != nil {
			fmt.Printf("❌ Error re-analyzing file: %v\n", err)
			continue
		}

		// Find the section again
		var found *section
		var content string
		for i, cs := range currentSections {
			if cs.name == s.name {
				found = &currentSections[i]
				content = sectionContents[cs.name]
				break
			}
		}

		if found == nil {
			fmt.Printf("⏭️  Section '%s' not found (may have been removed)\n", s.name)
			continue
		}

		// Ensure rules directory exists
		if err := os.MkdirAll(rulesDir, 0755); err != nil {
			fmt.Printf("❌ Error creating %s: %v\n", rulesDir, err)
			continue
		}

		// Write extracted content
		header := fmt.Sprintf("# %s\n\n", found.name)
		if err := os.WriteFile(outputPath, []byte(header+content), 0644); err != nil {
			fmt.Printf("❌ Error writing %s: %v\n", outputPath, err)
			continue
		}

		// Remove from CLAUDE.md
		if err := removeSectionAndAddReference(filePath, found, outputName); err != nil {
			fmt.Printf("⚠️  Extracted to %s but couldn't update %s: %v\n", outputPath, filePath, err)
		} else {
			fmt.Printf("✓ Extracted '%s' → %s\n", s.name, outputPath)
		}
	}

	// Show final state
	fmt.Println()
	newLines, _, _, _ := analyzeFileWithContent(filePath)
	newRulesCount := len(listRulesFiles())
	fmt.Printf("📊 Result: %s now has %d lines, %d rules modules\n", filePath, newLines, newRulesCount)

	return nil
}
