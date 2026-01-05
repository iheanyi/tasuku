package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the terminal user interface",
	Long: `Launch an interactive terminal user interface for managing tasks.

The TUI provides a visual dashboard for viewing and managing tasks,
with vim-like keybindings for navigation.

Keybindings:
  j/k, arrow keys   Navigate tasks
  enter             View task details
  s                 Start a task (mark as in_progress)
  d                 Mark task as done
  t                 Toggle timer on task
  a                 Archive a done task
  /                 Search/filter tasks
  r                 Refresh from disk
  esc               Go back
  q, ctrl+c         Quit

Colors based on iheanyi.com dark theme.

For a web-based dashboard, use:
  tk serve --http :8080

Then open http://localhost:8080 in your browser.`,
	RunE: runUI,
}

func init() {
	// No flags needed for now
}

func runUI(cmd *cobra.Command, args []string) error {
	s := store.DefaultStorageWithWarning()

	if !s.Exists() {
		return fmt.Errorf("no Tasuku storage found - run 'tk init' first")
	}

	model, err := tui.New(s)
	if err != nil {
		return fmt.Errorf("failed to initialize TUI: %w", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
