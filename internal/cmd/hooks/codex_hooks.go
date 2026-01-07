package hooks

// Codex hooks configuration.
// Codex uses ~/.codex/config.toml for configuration with a `notify` setting
// that runs a command on supported events (currently agent-turn-complete).
//
// TOML format:
//
//	notify = ["command", "arg1", "arg2"]
//
// The notify command receives a JSON argument with event details:
//   - type: "agent-turn-complete"
//   - thread-id: session identifier
//   - turn-id: turn identifier
//   - cwd: working directory
//   - input-messages: user messages
//   - last-assistant-message: assistant response text
//
// Note: Codex's hook system is more limited than Claude Code's.
// Only agent-turn-complete is currently supported for notifications.

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func getCodexConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "config.toml")
}

func readCodexConfig() (map[string]interface{}, error) {
	path := getCodexConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	config := make(map[string]interface{})
	if _, err := toml.Decode(string(data), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config.toml: %w", err)
	}
	return config, nil
}

func writeCodexConfig(config map[string]interface{}) error {
	path := getCodexConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(config)
}

// installCodexHooks installs Tasuku hooks in Codex configuration.
// Codex only supports a single notify command, so we install a wrapper
// script that calls tk hooks session-end.
func installCodexHooks(force bool) error {
	config, err := readCodexConfig()
	if err != nil {
		return err
	}

	// Check if Codex is installed
	home, _ := os.UserHomeDir()
	codexDir := filepath.Join(home, ".codex")
	if _, err := os.Stat(codexDir); os.IsNotExist(err) {
		fmt.Println("Codex not detected (no ~/.codex directory)")
		return nil
	}

	// Check if notify is already configured
	if existing, ok := config["notify"]; ok && !force {
		// Check if it's already our hook
		if arr, ok := existing.([]interface{}); ok {
			for _, item := range arr {
				if str, ok := item.(string); ok {
					if containsTasukuMarker(str) || str == "tk" {
						fmt.Println("Tasuku hook already configured in Codex.")
						return nil
					}
				}
			}
		}
		fmt.Println("Codex already has a notify command configured.")
		fmt.Println("Use --force to override, or manually add Tasuku to your notify script.")
		return nil
	}

	// Set up notify to call tk hooks stop-reminder
	// Codex passes JSON with event details as first arg
	config["notify"] = []interface{}{"tk", "hooks", "codex-notify"}

	if err := writeCodexConfig(config); err != nil {
		return err
	}

	fmt.Println("Tasuku hooks installed in Codex:")
	fmt.Println("  - notify: Calls tk hooks codex-notify on agent turn completion")
	fmt.Println()
	fmt.Println("Restart Codex for changes to take effect.")

	return nil
}

// uninstallCodexHooks removes Tasuku hooks from Codex configuration.
func uninstallCodexHooks() error {
	config, err := readCodexConfig()
	if err != nil {
		return err
	}

	// Check if notify is our hook
	notify, ok := config["notify"]
	if !ok {
		fmt.Println("No Codex hooks configured.")
		return nil
	}

	isTasukuHook := false
	if arr, ok := notify.([]interface{}); ok {
		for _, item := range arr {
			if str, ok := item.(string); ok {
				if str == "tk" || containsTasukuMarker(str) {
					isTasukuHook = true
					break
				}
			}
		}
	}

	if !isTasukuHook {
		fmt.Println("Codex notify is not configured by Tasuku.")
		return nil
	}

	// Remove the notify setting
	delete(config, "notify")

	if err := writeCodexConfig(config); err != nil {
		return err
	}

	fmt.Println("Tasuku hooks removed from Codex.")
	return nil
}
