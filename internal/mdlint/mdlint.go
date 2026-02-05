// Package mdlint provides shared markdown file analysis for linting and stats.
package mdlint

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Section represents an H2 section in a markdown file.
type Section struct {
	Name      string
	StartLine int
	EndLine   int
	Lines     int
}

// RulesFile represents a markdown file in a rules directory.
type RulesFile struct {
	Name  string
	Path  string
	Lines int
}

// LintResult contains the outcome of linting a markdown file.
type LintResult struct {
	Status          string    // "ok", "warning", "error"
	File            string
	TotalLines      int
	Sections        []Section
	LargeSections   []Section
	Issues          []string
	Recommendations []string
}

// StatsResult contains section breakdown statistics.
type StatsResult struct {
	File       string
	TotalLines int
	Sections   []SectionStat
	RulesFiles []RulesFile
}

// SectionStat is a section with its percentage of total lines.
type SectionStat struct {
	Name       string
	Lines      int
	Percentage float64
}

// Config controls lint/stats thresholds and file-aware messages.
type Config struct {
	MaxLines     int    // default 200
	WarnLines    int    // default 150
	SectionLimit int    // default 50
	FileName     string // e.g., "CLAUDE.md" — used in messages
	RulesDir     string // e.g., ".claude/rules" — empty means no rules dir
	StatsCmd     string // e.g., "tk claudemd stats" — used in suggestions
}

// DefaultConfig returns a Config with default thresholds.
// Caller should set FileName, RulesDir, and StatsCmd.
func DefaultConfig() Config {
	return Config{
		MaxLines:     200,
		WarnLines:    150,
		SectionLimit: 50,
	}
}

// AnalyzeFile parses a markdown file and returns line count and H2 sections.
func AnalyzeFile(path string) (int, []Section, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()

	var sections []Section
	var currentSection *Section

	lineNum := 0
	scanner := bufio.NewScanner(file)
	h2Pattern := regexp.MustCompile(`^##\s+(.+)$`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if matches := h2Pattern.FindStringSubmatch(line); len(matches) > 1 {
			// Close previous section
			if currentSection != nil {
				currentSection.EndLine = lineNum - 1
				currentSection.Lines = currentSection.EndLine - currentSection.StartLine + 1
				sections = append(sections, *currentSection)
			}

			currentSection = &Section{
				Name:      matches[1],
				StartLine: lineNum,
			}
		}
	}

	// Close final section
	if currentSection != nil {
		currentSection.EndLine = lineNum
		currentSection.Lines = currentSection.EndLine - currentSection.StartLine + 1
		sections = append(sections, *currentSection)
	}

	if err := scanner.Err(); err != nil {
		return 0, nil, err
	}

	return lineNum, sections, nil
}

// CountFileLines counts lines in a file. Returns 0 if the file cannot be read.
func CountFileLines(path string) int {
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

// ListRulesFiles returns markdown files found in the given rules directory.
// Returns nil if the directory does not exist.
func ListRulesFiles(rulesDir string) []RulesFile {
	if rulesDir == "" {
		return nil
	}

	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		return nil
	}

	var files []RulesFile
	filepath.Walk(rulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, RulesFile{
				Name:  filepath.Base(path),
				Path:  path,
				Lines: CountFileLines(path),
			})
		}
		return nil
	})

	return files
}

// Lint analyzes a markdown file and returns a structured lint result.
// Returns error only for I/O failures; lint status is data, not an error.
func Lint(filePath string, cfg Config) (*LintResult, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &LintResult{
			Status: "ok",
			File:   filePath,
		}, nil
	}

	totalLines, sections, err := AnalyzeFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze %s: %w", filePath, err)
	}

	result := &LintResult{
		Status:     "ok",
		File:       filePath,
		TotalLines: totalLines,
		Sections:   sections,
	}

	// Determine status
	if totalLines > cfg.MaxLines {
		result.Status = "error"
		result.Issues = append(result.Issues, fmt.Sprintf("Exceeds recommended maximum (%d lines)", cfg.MaxLines))
	} else if totalLines > cfg.WarnLines {
		result.Status = "warning"
		result.Issues = append(result.Issues, fmt.Sprintf("Approaching limit (%d/%d lines)", totalLines, cfg.MaxLines))
	}

	// Check for large sections
	for _, s := range sections {
		if s.Lines > cfg.SectionLimit {
			result.LargeSections = append(result.LargeSections, s)
		}
	}

	// Build recommendations if there are issues or large sections
	if result.Status != "ok" || len(result.LargeSections) > 0 {
		if cfg.RulesDir != "" {
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Move large sections to %s/ modules", cfg.RulesDir))
		}

		fileName := cfg.FileName
		if fileName == "" {
			fileName = filepath.Base(filePath)
		}
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Keep %s focused on overview and key decisions", fileName))

		if cfg.StatsCmd != "" {
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Use '%s' to see full breakdown", cfg.StatsCmd))
		}
	}

	return result, nil
}

// Stats analyzes a markdown file and returns section breakdown statistics.
func Stats(filePath string, cfg Config) (*StatsResult, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s not found", filePath)
	}

	totalLines, sections, err := AnalyzeFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze %s: %w", filePath, err)
	}

	result := &StatsResult{
		File:       filePath,
		TotalLines: totalLines,
	}

	// Build section stats sorted by line count descending
	stats := make([]SectionStat, len(sections))
	for i, s := range sections {
		pct := 0.0
		if totalLines > 0 {
			pct = float64(s.Lines) / float64(totalLines) * 100
		}
		stats[i] = SectionStat{
			Name:       s.Name,
			Lines:      s.Lines,
			Percentage: pct,
		}
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Lines > stats[j].Lines
	})
	result.Sections = stats

	// List rules files if configured
	if cfg.RulesDir != "" {
		result.RulesFiles = ListRulesFiles(cfg.RulesDir)
	}

	return result, nil
}
