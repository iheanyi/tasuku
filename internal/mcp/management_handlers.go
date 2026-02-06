// Package mcp provides MCP handlers for plugin, MCP, and hooks management.
package mcp

import (
	"fmt"
	"os"
	"strings"

	"github.com/iheanyi/tasuku/internal/plugin"
)

// Plugin handlers

func (s *Server) handlePluginInstall(args map[string]interface{}) (interface{}, error) {
	tool, _ := args["tool"].(string)
	local, _ := args["local"].(bool)

	var targets []plugin.ToolTarget

	if tool != "" {
		t := plugin.GetToolByName(tool)
		if t == nil {
			return nil, fmt.Errorf("unknown tool: %s (valid: claude, cursor, copilot, codex)", tool)
		}
		targets = append(targets, *t)
	} else {
		targets = plugin.GetDetectedTools()
		if len(targets) == 0 {
			return map[string]interface{}{
				"status":  "no_tools_detected",
				"message": "No supported AI tools detected",
				"supported_tools": func() []string {
					var names []string
					for _, t := range plugin.GetSupportedTools() {
						names = append(names, t.Name)
					}
					return names
				}(),
			}, nil
		}
	}

	var results []map[string]interface{}
	var hasErrors bool

	for _, target := range targets {
		result := plugin.InstallToTool(target, local)

		r := map[string]interface{}{
			"tool":   target.Name,
			"status": "success",
		}

		if len(result.Errors) > 0 {
			r["errors"] = result.Errors
			if len(result.FilesAdded) == 0 {
				r["status"] = "failed"
				hasErrors = true
			} else {
				r["status"] = "partial"
			}
		}

		if len(result.FilesAdded) > 0 {
			r["files_installed"] = len(result.FilesAdded)
		}

		results = append(results, r)
	}

	status := "success"
	if hasErrors {
		status = "partial_failure"
	}

	return map[string]interface{}{
		"status":  status,
		"results": results,
	}, nil
}

func (s *Server) handlePluginUninstall(args map[string]interface{}) (interface{}, error) {
	tool, _ := args["tool"].(string)
	local, _ := args["local"].(bool)

	var targets []plugin.ToolTarget

	if tool != "" {
		t := plugin.GetToolByName(tool)
		if t == nil {
			return nil, fmt.Errorf("unknown tool: %s (valid: claude, cursor, copilot, codex)", tool)
		}
		targets = append(targets, *t)
	} else {
		targets = plugin.GetDetectedTools()
		if len(targets) == 0 {
			return map[string]interface{}{
				"status":  "no_tools_detected",
				"message": "No supported AI tools detected",
			}, nil
		}
	}

	var results []map[string]interface{}

	for _, target := range targets {
		result := plugin.UninstallFromTool(target, local)

		r := map[string]interface{}{
			"tool": target.Name,
		}

		if len(result.Errors) > 0 {
			r["status"] = "error"
			r["errors"] = result.Errors
		} else if result.AlreadyDone {
			r["status"] = "already_uninstalled"
		} else {
			r["status"] = "success"
			r["files_removed"] = len(result.FilesAdded)
		}

		results = append(results, r)
	}

	return map[string]interface{}{
		"status":  "success",
		"results": results,
	}, nil
}

func (s *Server) handlePluginList(args map[string]interface{}) (interface{}, error) {
	commands, err := plugin.LoadEmbeddedCommands()
	if err != nil {
		return nil, fmt.Errorf("failed to load commands: %w", err)
	}

	type commandInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	var workflow []commandInfo
	var basic []commandInfo

	for _, c := range commands {
		info := commandInfo{
			Name:        c.Name,
			Description: c.Description,
		}
		switch c.Name {
		case "pickup", "complete", "reflect", "help", "tasuku":
			info.Type = "workflow"
			workflow = append(workflow, info)
		default:
			info.Type = "basic"
			basic = append(basic, info)
		}
	}

	return map[string]interface{}{
		"total_commands":    len(commands),
		"workflow_commands": workflow,
		"basic_commands":    basic,
	}, nil
}

func (s *Server) handlePluginStatus(args map[string]interface{}) (interface{}, error) {
	detected := plugin.GetDetectedTools()

	type toolStatus struct {
		Name           string `json:"name"`
		LocalInstalled bool   `json:"local_installed"`
		LocalDir       string `json:"local_dir,omitempty"`
		GlobalInstalled bool  `json:"global_installed"`
		GlobalDir      string `json:"global_dir,omitempty"`
	}

	var statuses []toolStatus

	for _, tool := range detected {
		status := toolStatus{
			Name:            tool.Name,
			LocalInstalled:  checkPluginInstalled(tool.LocalDir),
			LocalDir:        tool.LocalDir,
			GlobalInstalled: checkPluginInstalled(tool.GlobalDir),
			GlobalDir:       tool.GlobalDir,
		}
		statuses = append(statuses, status)
	}

	if len(statuses) == 0 {
		// Return info about supported tools
		var supported []string
		for _, t := range plugin.GetSupportedTools() {
			supported = append(supported, t.Name)
		}
		return map[string]interface{}{
			"status":          "no_tools_detected",
			"detected_tools":  []toolStatus{},
			"supported_tools": supported,
		}, nil
	}

	return map[string]interface{}{
		"status":         "success",
		"detected_tools": statuses,
	}, nil
}

func checkPluginInstalled(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			return true
		}
	}
	return false
}

// MCP handlers

func (s *Server) handleMCPInstall(args map[string]interface{}) (interface{}, error) {
	// The MCP install functionality is in internal/cmd/mcpcmd
	// We need to return guidance since we can't directly call unexported functions

	tool, _ := args["tool"].(string)
	local, _ := args["local"].(bool)
	force, _ := args["force"].(bool)

	return map[string]interface{}{
		"status":  "guidance",
		"message": "Use the CLI command to install MCP configuration",
		"command": buildMCPInstallCommand(tool, local, force),
		"hint":    "Run 'tk mcp install' in your terminal to auto-configure MCP",
	}, nil
}

func buildMCPInstallCommand(tool string, local, force bool) string {
	cmd := "tk mcp install"
	if tool != "" {
		cmd += " --tool " + tool
	}
	if local {
		cmd += " --local"
	}
	if force {
		cmd += " --force"
	}
	return cmd
}

func (s *Server) handleMCPUninstall(args map[string]interface{}) (interface{}, error) {
	local, _ := args["local"].(bool)

	cmd := "tk mcp uninstall"
	if local {
		cmd += " --local"
	}

	return map[string]interface{}{
		"status":  "guidance",
		"message": "Use the CLI command to uninstall MCP configuration",
		"command": cmd,
		"hint":    "Run 'tk mcp uninstall' in your terminal to remove MCP configuration",
	}, nil
}

// Hooks handlers

func (s *Server) handleHooksInstall(args map[string]interface{}) (interface{}, error) {
	// Build the CLI command based on args
	gitOnly, _ := args["git"].(bool)
	claudeOnly, _ := args["claude"].(bool)
	codexOnly, _ := args["codex"].(bool)
	opencodeOnly, _ := args["opencode"].(bool)
	copilotOnly, _ := args["copilot"].(bool)
	cursorOnly, _ := args["cursor"].(bool)
	local, _ := args["local"].(bool)
	force, _ := args["force"].(bool)

	cmd := "tk hooks install"
	if gitOnly {
		cmd += " --git"
	}
	if claudeOnly {
		cmd += " --claude"
	}
	if codexOnly {
		cmd += " --codex"
	}
	if opencodeOnly {
		cmd += " --opencode"
	}
	if copilotOnly {
		cmd += " --copilot"
	}
	if cursorOnly {
		cmd += " --cursor"
	}
	if local {
		cmd += " --local"
	}
	if force {
		cmd += " --force"
	}

	return map[string]interface{}{
		"status":  "guidance",
		"message": "Use the CLI command to install hooks",
		"command": cmd,
		"hint":    "Run 'tk hooks install' in your terminal to set up hooks",
	}, nil
}

func (s *Server) handleHooksUninstall(args map[string]interface{}) (interface{}, error) {
	// Build the CLI command based on args
	gitOnly, _ := args["git"].(bool)
	claudeOnly, _ := args["claude"].(bool)
	codexOnly, _ := args["codex"].(bool)
	opencodeOnly, _ := args["opencode"].(bool)
	copilotOnly, _ := args["copilot"].(bool)
	cursorOnly, _ := args["cursor"].(bool)
	local, _ := args["local"].(bool)

	cmd := "tk hooks uninstall"
	if gitOnly {
		cmd += " --git"
	}
	if claudeOnly {
		cmd += " --claude"
	}
	if codexOnly {
		cmd += " --codex"
	}
	if opencodeOnly {
		cmd += " --opencode"
	}
	if copilotOnly {
		cmd += " --copilot"
	}
	if cursorOnly {
		cmd += " --cursor"
	}
	if local {
		cmd += " --local"
	}

	return map[string]interface{}{
		"status":  "guidance",
		"message": "Use the CLI command to uninstall hooks",
		"command": cmd,
		"hint":    "Run 'tk hooks uninstall' in your terminal to remove hooks",
	}, nil
}
