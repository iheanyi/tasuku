// Package plugin provides cross-tool plugin/skill installation for Tasuku.
//
// Different AI tools have different formats for "guided workflows":
//   - Claude Code: Plugins with commands/*.md (YAML frontmatter with description, argument-hint)
//   - Copilot CLI: Skills in .github/skills/ with SKILL.md format (name, description)
//   - Codex: Skills in .codex/skills/ with SKILL.md format (name, description)
//   - Cursor: Similar to Claude Code but may have different conventions
//
// This package provides:
//   - Detection of which tools are available
//   - Conversion from Claude Code plugin format to other tool formats
//   - Installation commands for each tool
package plugin

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/iheanyi/tasuku/internal/tools"
)

//go:embed commands/*.md
var embeddedCommands embed.FS

// ToolTarget represents a supported AI tool for plugin installation.
type ToolTarget struct {
	Name        string
	SkillsDir   string   // Where to install skills/commands
	GlobalDir   string   // Global installation directory
	LocalDir    string   // Local (project) installation directory
	Format      string   // "claude-plugin", "skill-md"
	DetectFiles []string // Files that indicate tool presence
}

// InstallResult contains the result of an installation.
type InstallResult struct {
	Tool        string
	FilesAdded  []string
	Errors      []string
	AlreadyDone bool
}

// Command represents a parsed plugin command.
type Command struct {
	Name        string
	Description string
	ArgumentHint string
	Content     string // Full markdown content after frontmatter
}

// SkillMDFrontmatter is the YAML frontmatter for SKILL.md format.
type SkillMDFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ClaudeCommandFrontmatter is the YAML frontmatter for Claude Code commands.
type ClaudeCommandFrontmatter struct {
	Description  string `yaml:"description"`
	ArgumentHint string `yaml:"argument-hint,omitempty"`
}

// GetSupportedTools returns all supported tool targets.
func GetSupportedTools() []ToolTarget {
	home, _ := os.UserHomeDir()

	return []ToolTarget{
		{
			Name:        "Claude Code",
			Format:      "claude-plugin",
			LocalDir:    ".claude",
			GlobalDir:   filepath.Join(home, ".claude"),
			DetectFiles: []string{".claude", "CLAUDE.md"},
		},
		{
			Name:        "Cursor",
			Format:      "cursor-command",
			LocalDir:    ".cursor/commands/tasuku",
			GlobalDir:   filepath.Join(home, ".cursor/commands/tasuku"),
			DetectFiles: []string{".cursor", ".cursorrules"},
		},
		{
			Name:        "Copilot CLI",
			Format:      "skill-md",
			LocalDir:    ".github/skills/tasuku",
			GlobalDir:   filepath.Join(home, ".copilot/skills/tasuku"),
			DetectFiles: []string{".github/hooks", ".copilot"},
		},
		{
			Name:        "Codex",
			Format:      "skill-md",
			LocalDir:    ".codex/skills/tasuku",
			GlobalDir:   filepath.Join(home, ".codex/skills/tasuku"),
			DetectFiles: []string{".codex", "CODEX.md"},
		},
		{
			Name:        "Amp",
			Format:      "skill-md",
			LocalDir:    ".agents/skills/tasuku",
			GlobalDir:   filepath.Join(home, ".config/agents/skills/tasuku"),
			DetectFiles: []string{".amp", "AGENTS.md"},
		},
	}
}

// GetDetectedTools returns tools that are detected in the current directory.
func GetDetectedTools() []ToolTarget {
	all := GetSupportedTools()
	detected := []ToolTarget{}

	for _, tool := range all {
		for _, detect := range tool.DetectFiles {
			if exists(detect) {
				detected = append(detected, tool)
				break
			}
		}
	}

	return detected
}

// GetToolByName returns a tool target by name (case-insensitive).
func GetToolByName(name string) *ToolTarget {
	tool, ok := tools.Resolve(name)
	if !ok {
		return nil
	}

	for _, t := range GetSupportedTools() {
		if t.Name == tool.String() {
			return &t
		}
	}
	return nil
}

// Cached embedded commands (immutable at runtime, so safe to cache)
var (
	cachedCommands    []Command
	cachedCommandsErr error
	loadCommandsOnce  sync.Once
)

// LoadEmbeddedCommands loads commands from the embedded filesystem.
// Results are cached since embedded content never changes at runtime.
func LoadEmbeddedCommands() ([]Command, error) {
	loadCommandsOnce.Do(func() {
		cachedCommands, cachedCommandsErr = loadEmbeddedCommandsInternal()
	})
	return cachedCommands, cachedCommandsErr
}

// loadEmbeddedCommandsInternal does the actual loading (called once via sync.Once).
func loadEmbeddedCommandsInternal() ([]Command, error) {
	commands := []Command{}

	err := fs.WalkDir(embeddedCommands, "commands", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := embeddedCommands.ReadFile(path)
		if err != nil {
			return err
		}

		cmd, err := parseClaudeCommand(filepath.Base(path), data)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		commands = append(commands, cmd)
		return nil
	})

	return commands, err
}

// parseClaudeCommand parses a Claude Code command file.
func parseClaudeCommand(filename string, data []byte) (Command, error) {
	name := strings.TrimSuffix(filename, ".md")
	cmd := Command{Name: name}

	// Split frontmatter and content
	content := string(data)
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content[4:], "\n---\n", 2)
		if len(parts) == 2 {
			var fm ClaudeCommandFrontmatter
			if err := yaml.Unmarshal([]byte(parts[0]), &fm); err != nil {
				return cmd, fmt.Errorf("invalid frontmatter: %w", err)
			}
			cmd.Description = fm.Description
			cmd.ArgumentHint = fm.ArgumentHint
			cmd.Content = strings.TrimSpace(parts[1])
		}
	} else {
		cmd.Content = content
	}

	return cmd, nil
}

// ConvertToSkillMD converts a Claude command to SKILL.md format.
func ConvertToSkillMD(cmd Command) []byte {
	var buf bytes.Buffer

	// Write frontmatter
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("name: %s\n", cmd.Name))
	buf.WriteString(fmt.Sprintf("description: %s\n", cmd.Description))
	buf.WriteString("---\n\n")

	// Write content
	buf.WriteString(cmd.Content)

	return buf.Bytes()
}

// titleCaser is a reusable title case converter for English.
var titleCaser = cases.Title(language.English)

// ConvertToCursorCommand converts a Claude command to Cursor command format.
// Cursor commands are plain Markdown files without frontmatter.
// Format: # Title\n\nDescription\n\n## Instructions\n\nContent
func ConvertToCursorCommand(cmd Command) []byte {
	var buf bytes.Buffer

	// Write title (convert name to Title Case)
	title := titleCaser.String(strings.ReplaceAll(cmd.Name, "-", " "))
	buf.WriteString(fmt.Sprintf("# %s\n\n", title))

	// Write description as overview
	if cmd.Description != "" {
		buf.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))
	}

	// Write content as instructions
	if cmd.Content != "" {
		buf.WriteString("## Instructions\n\n")
		buf.WriteString(cmd.Content)
	}

	return buf.Bytes()
}

// InstallToTool installs Tasuku commands to a specific tool.
func InstallToTool(tool ToolTarget, local bool) InstallResult {
	result := InstallResult{Tool: tool.Name}

	// Determine target directory
	targetDir := tool.GlobalDir
	if local {
		targetDir = tool.LocalDir
	}

	// Load commands
	commands, err := LoadEmbeddedCommands()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to load commands: %v", err))
		return result
	}

	switch tool.Format {
	case "claude-plugin":
		// For Claude Code, we can't directly install - guide user
		result.Errors = append(result.Errors, `Claude Code uses a plugin system that requires manual installation.

To install the Tasuku plugin:
  1. In Claude Code, run: /plugin marketplace add https://github.com/iheanyi/tasuku
  2. Then run: /plugin install tasuku

This provides all /tasuku:* commands.`)
		return result

	case "cursor-command":
		// Create commands directory
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create directory: %v", err))
			return result
		}

		// Convert and write each command as a Cursor command
		for _, cmd := range commands {
			cmdData := ConvertToCursorCommand(cmd)
			// Use tasuku- prefix to namespace commands
			filename := fmt.Sprintf("tasuku-%s.md", cmd.Name)
			path := filepath.Join(targetDir, filename)

			if err := safeWriteFile(path, cmdData, 0644); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to write %s: %v", filename, err))
			} else {
				result.FilesAdded = append(result.FilesAdded, path)
			}
		}

	case "skill-md":
		// Create skills directory
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to create directory: %v", err))
			return result
		}

		// Convert and write each command as a skill
		for _, cmd := range commands {
			skillData := ConvertToSkillMD(cmd)
			filename := fmt.Sprintf("%s.md", cmd.Name)
			path := filepath.Join(targetDir, filename)

			if err := safeWriteFile(path, skillData, 0644); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to write %s: %v", filename, err))
			} else {
				result.FilesAdded = append(result.FilesAdded, path)
			}
		}
	}

	return result
}

// UninstallFromTool removes Tasuku skills from a specific tool.
func UninstallFromTool(tool ToolTarget, local bool) InstallResult {
	result := InstallResult{Tool: tool.Name}

	// Determine target directory
	targetDir := tool.GlobalDir
	if local {
		targetDir = tool.LocalDir
	}

	switch tool.Format {
	case "claude-plugin":
		result.Errors = append(result.Errors, `To uninstall the Tasuku plugin from Claude Code:
  Run: /plugin uninstall tasuku`)
		return result

	case "cursor-command":
		// Remove Cursor commands directory
		if !exists(targetDir) {
			result.AlreadyDone = true
			return result
		}

		entries, _ := os.ReadDir(targetDir)
		for _, entry := range entries {
			// Only remove tasuku- prefixed files
			if strings.HasPrefix(entry.Name(), "tasuku-") && strings.HasSuffix(entry.Name(), ".md") {
				path := filepath.Join(targetDir, entry.Name())
				if err := os.Remove(path); err == nil {
					result.FilesAdded = append(result.FilesAdded, path)
				}
			}
		}

		// Try to remove empty directory
		os.Remove(targetDir)

	case "skill-md":
		// Remove skills directory
		if !exists(targetDir) {
			result.AlreadyDone = true
			return result
		}

		entries, _ := os.ReadDir(targetDir)
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".md") {
				path := filepath.Join(targetDir, entry.Name())
				if err := os.Remove(path); err == nil {
					result.FilesAdded = append(result.FilesAdded, path)
				}
			}
		}

		// Try to remove empty directory
		os.Remove(targetDir)
	}

	return result
}

// Helper functions

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// safeWriteFile writes content to a file, refusing to follow symlinks for security.
func safeWriteFile(path string, content []byte, perm os.FileMode) error {
	// Check if path exists and is a symlink
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write to symlink: %s", path)
		}
	}
	return os.WriteFile(path, content, perm)
}
