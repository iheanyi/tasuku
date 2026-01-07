// Package rules provides synchronization of Tasuku learnings and decisions
// to editor-specific rules directories (.claude/rules/, .cursor/rules/).
package rules

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

// EditorTarget represents a supported editor for rules sync.
type EditorTarget struct {
	Name       string
	RulesDir   string // e.g., ".claude/rules/tasuku"
	DetectDirs []string
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	Editor       string
	FilesWritten []string
	Errors       []string
}

// GetTargets returns all detected editor targets in the current directory.
func GetTargets() []EditorTarget {
	targets := []EditorTarget{}

	// Claude Code: detect via .claude/ or CLAUDE.md
	if dirExists(".claude") || fileExists("CLAUDE.md") {
		targets = append(targets, EditorTarget{
			Name:       "Claude Code",
			RulesDir:   ".claude/rules/tasuku",
			DetectDirs: []string{".claude", "CLAUDE.md"},
		})
	}

	// Cursor: detect via .cursor/ or .cursorrules
	if dirExists(".cursor") || fileExists(".cursorrules") {
		targets = append(targets, EditorTarget{
			Name:       "Cursor",
			RulesDir:   ".cursor/rules/tasuku",
			DetectDirs: []string{".cursor", ".cursorrules"},
		})
	}

	// Codex: detect via .codex/ or CODEX.md
	// Codex uses ~/.codex/ globally, but for project rules we can use .codex/rules/
	if dirExists(".codex") || fileExists("CODEX.md") {
		targets = append(targets, EditorTarget{
			Name:       "Codex",
			RulesDir:   ".codex/rules/tasuku",
			DetectDirs: []string{".codex", "CODEX.md"},
		})
	}

	// OpenCode: detect via opencode.json or .opencode/
	if fileExists("opencode.json") || dirExists(".opencode") {
		targets = append(targets, EditorTarget{
			Name:       "OpenCode",
			RulesDir:   ".opencode/rules/tasuku",
			DetectDirs: []string{"opencode.json", ".opencode"},
		})
	}

	return targets
}

// HasSyncedRules checks if any editor target has actual rules files synced.
// This is different from GetTargets which just checks if editors are detected.
// Use this to avoid skipping learnings in context when sync hasn't happened yet.
func HasSyncedRules() bool {
	targets := GetTargets()
	for _, target := range targets {
		// Check if the rules directory exists and has .md files
		if dirExists(target.RulesDir) {
			entries, err := os.ReadDir(target.RulesDir)
			if err == nil {
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
						return true
					}
				}
			}
		}
	}
	return false
}

// Sync synchronizes learnings and decisions to all detected editor targets.
func Sync(learnings []task.Learning, decisions []task.Decision) ([]SyncResult, error) {
	targets := GetTargets()
	if len(targets) == 0 {
		return nil, fmt.Errorf("no supported editors detected (need .claude/, .cursor/, .codex/, or .opencode/)")
	}

	results := make([]SyncResult, 0, len(targets))
	for _, target := range targets {
		result := syncToTarget(target, learnings, decisions)
		results = append(results, result)
	}

	return results, nil
}

// SyncToTool synchronizes learnings and decisions to a specific tool.
// Tool names: "claude", "cursor", "codex", "opencode"
func SyncToTool(learnings []task.Learning, decisions []task.Decision, tool string) ([]SyncResult, error) {
	// Normalize tool name
	toolLower := strings.ToLower(tool)

	// Map tool aliases to canonical names
	toolMap := map[string]string{
		"claude":     "Claude Code",
		"claude-code": "Claude Code",
		"claudecode": "Claude Code",
		"cursor":     "Cursor",
		"codex":      "Codex",
		"opencode":   "OpenCode",
		"open-code":  "OpenCode",
	}

	targetName, ok := toolMap[toolLower]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s (valid: claude, cursor, codex, opencode)", tool)
	}

	// Get the specific target configuration (even if not detected)
	target := getTargetByName(targetName)
	if target == nil {
		return nil, fmt.Errorf("tool %s not detected in current directory", targetName)
	}

	result := syncToTarget(*target, learnings, decisions)
	return []SyncResult{result}, nil
}

// getTargetByName returns a target by its canonical name, creating the config even if not detected.
func getTargetByName(name string) *EditorTarget {
	switch name {
	case "Claude Code":
		return &EditorTarget{
			Name:       "Claude Code",
			RulesDir:   ".claude/rules/tasuku",
			DetectDirs: []string{".claude", "CLAUDE.md"},
		}
	case "Cursor":
		return &EditorTarget{
			Name:       "Cursor",
			RulesDir:   ".cursor/rules/tasuku",
			DetectDirs: []string{".cursor", ".cursorrules"},
		}
	case "Codex":
		return &EditorTarget{
			Name:       "Codex",
			RulesDir:   ".codex/rules/tasuku",
			DetectDirs: []string{".codex", "CODEX.md"},
		}
	case "OpenCode":
		return &EditorTarget{
			Name:       "OpenCode",
			RulesDir:   ".opencode/rules/tasuku",
			DetectDirs: []string{"opencode.json", ".opencode"},
		}
	default:
		return nil
	}
}

// SyncToTarget syncs to a specific editor target.
func syncToTarget(target EditorTarget, learnings []task.Learning, decisions []task.Decision) SyncResult {
	result := SyncResult{
		Editor:       target.Name,
		FilesWritten: []string{},
		Errors:       []string{},
	}

	// Create rules directory
	if err := os.MkdirAll(target.RulesDir, 0755); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create %s: %v", target.RulesDir, err))
		return result
	}

	// Clean existing .md files to prevent orphaned scope files
	// (e.g., if all learnings with scope "src/api/**" are removed, learnings-api.md should be deleted)
	if entries, err := os.ReadDir(target.RulesDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				os.Remove(filepath.Join(target.RulesDir, entry.Name()))
			}
		}
	}

	// Group learnings by scope
	unscopedLearnings := []task.Learning{}
	scopedLearnings := make(map[string][]task.Learning) // scope -> learnings

	for _, l := range learnings {
		if l.Scope == "" {
			unscopedLearnings = append(unscopedLearnings, l)
		} else {
			scopedLearnings[l.Scope] = append(scopedLearnings[l.Scope], l)
		}
	}

	// Write unscoped learnings
	if len(unscopedLearnings) > 0 {
		path := filepath.Join(target.RulesDir, "learnings.md")
		content := generateLearningsMarkdown(unscopedLearnings, "")
		if err := os.WriteFile(path, content, 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to write %s: %v", path, err))
		} else {
			result.FilesWritten = append(result.FilesWritten, path)
		}
	}

	// Write scoped learnings (each scope gets its own file)
	for scope, scopeLearnings := range scopedLearnings {
		slug := scopeToSlug(scope)
		path := filepath.Join(target.RulesDir, fmt.Sprintf("learnings-%s.md", slug))
		content := generateLearningsMarkdown(scopeLearnings, scope)
		if err := os.WriteFile(path, content, 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to write %s: %v", path, err))
		} else {
			result.FilesWritten = append(result.FilesWritten, path)
		}
	}

	// Write decisions
	if len(decisions) > 0 {
		path := filepath.Join(target.RulesDir, "decisions.md")
		content := generateDecisionsMarkdown(decisions)
		if err := os.WriteFile(path, content, 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to write %s: %v", path, err))
		} else {
			result.FilesWritten = append(result.FilesWritten, path)
		}
	}

	return result
}

// generateLearningsMarkdown generates markdown content for learnings.
// If scope is provided, adds YAML frontmatter with paths field.
func generateLearningsMarkdown(learnings []task.Learning, scope string) []byte {
	var buf bytes.Buffer

	// Add YAML frontmatter with paths if scoped
	if scope != "" {
		buf.WriteString("---\n")
		buf.WriteString(fmt.Sprintf("paths: %s\n", scope))
		buf.WriteString("---\n\n")
	}

	buf.WriteString("# Tasuku Learnings\n\n")
	buf.WriteString("_Auto-synced from .tasuku/context/learnings.md_\n\n")

	// Separate rules from insights
	var rules, insights []task.Learning
	for _, l := range learnings {
		if l.IsRule {
			rules = append(rules, l)
		} else {
			insights = append(insights, l)
		}
	}

	// Write rules section
	if len(rules) > 0 {
		buf.WriteString("## Rules\n\n")
		for _, l := range rules {
			buf.WriteString(fmt.Sprintf("- %s\n", l.Text))
		}
		buf.WriteString("\n")
	}

	// Write insights section
	if len(insights) > 0 {
		buf.WriteString("## Insights\n\n")
		for _, l := range insights {
			buf.WriteString(fmt.Sprintf("- %s\n", l.Text))
		}
		buf.WriteString("\n")
	}

	return buf.Bytes()
}

// generateDecisionsMarkdown generates markdown content for decisions.
func generateDecisionsMarkdown(decisions []task.Decision) []byte {
	var buf bytes.Buffer

	buf.WriteString("# Tasuku Decisions\n\n")
	buf.WriteString("_Auto-synced from .tasuku/context/decisions.md_\n\n")

	for _, d := range decisions {
		dateStr := "unknown"
		if !d.CreatedAt.IsZero() {
			dateStr = d.CreatedAt.Format(time.DateOnly)
		}

		buf.WriteString(fmt.Sprintf("## %s (%s)\n\n", d.ID, dateStr))
		buf.WriteString(fmt.Sprintf("**Chose**: %s\n\n", d.Chose))
		if len(d.Over) > 0 {
			buf.WriteString(fmt.Sprintf("**Over**: %s\n\n", strings.Join(d.Over, ", ")))
		}
		buf.WriteString(fmt.Sprintf("**Because**: %s\n\n", d.Because))
	}

	return buf.Bytes()
}

// scopeToSlug converts a scope pattern to a filename-safe slug.
// e.g., "src/api/**" -> "api", "src/components/**/*.tsx" -> "components"
func scopeToSlug(scope string) string {
	// Split by slash to find path segments
	parts := strings.Split(scope, "/")

	// Find the last meaningful directory name (not a glob pattern)
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		// Skip if it's a glob pattern, extension pattern, or common prefix
		if part == "" || part == "src" || part == "." ||
			strings.Contains(part, "*") ||
			strings.HasPrefix(part, ".") {
			continue
		}
		// Found a meaningful directory name
		slug := strings.ToLower(part)
		// Sanitize: only alphanumeric and hyphens
		reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
		slug = reg.ReplaceAllString(slug, "-")
		slug = strings.Trim(slug, "-")
		if slug != "" {
			return slug
		}
	}

	return "scoped"
}

// Clean removes all Tasuku-generated rules files from detected targets.
func Clean() ([]string, error) {
	targets := GetTargets()
	if len(targets) == 0 {
		return nil, nil
	}

	removed := []string{}
	for _, target := range targets {
		cleaned := cleanTarget(target)
		removed = append(removed, cleaned...)
	}

	return removed, nil
}

// CleanTool removes Tasuku-generated rules files from a specific tool.
func CleanTool(tool string) ([]string, error) {
	// Normalize tool name
	toolLower := strings.ToLower(tool)

	// Map tool aliases to canonical names
	toolMap := map[string]string{
		"claude":      "Claude Code",
		"claude-code": "Claude Code",
		"claudecode":  "Claude Code",
		"cursor":      "Cursor",
		"codex":       "Codex",
		"opencode":    "OpenCode",
		"open-code":   "OpenCode",
	}

	targetName, ok := toolMap[toolLower]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s (valid: claude, cursor, codex, opencode)", tool)
	}

	target := getTargetByName(targetName)
	if target == nil {
		return nil, nil
	}

	return cleanTarget(*target), nil
}

// cleanTarget removes rules files from a single target.
func cleanTarget(target EditorTarget) []string {
	removed := []string{}

	if dirExists(target.RulesDir) {
		entries, _ := os.ReadDir(target.RulesDir)
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				path := filepath.Join(target.RulesDir, entry.Name())
				if err := os.Remove(path); err == nil {
					removed = append(removed, path)
				}
			}
		}
		// Try to remove empty directory
		os.Remove(target.RulesDir)
	}

	return removed
}

// Helper functions

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
