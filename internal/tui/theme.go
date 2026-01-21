// Package tui provides a terminal user interface for tasuku.
package tui

import "github.com/charmbracelet/lipgloss"

// Theme colors based on iheanyi.com dark theme
var (
	// Base colors
	ColorBg    = lipgloss.Color("#1a1a1a") // Near-black background
	ColorFg    = lipgloss.Color("#f0f0f0") // Light gray foreground
	ColorMuted = lipgloss.Color("#6b7280") // Gray for comments/muted text
	ColorSlate = lipgloss.Color("#94a3b8") // Slate for punctuation/dim

	// Accent colors
	ColorAccent  = lipgloss.Color("#5eead4") // Teal/cyan for highlights
	ColorPrimary = lipgloss.Color("#5c9eff") // Bright blue for primary
	ColorPurple  = lipgloss.Color("#c4b5fd") // Light purple for functions
	ColorAmber   = lipgloss.Color("#fbbf24") // Amber/gold for warnings

	// Status colors
	ColorReady      = lipgloss.Color("#5eead4") // Teal for ready tasks
	ColorInProgress = lipgloss.Color("#fbbf24") // Amber for in-progress
	ColorBlocked    = lipgloss.Color("#ef4444") // Red for blocked
	ColorDone       = lipgloss.Color("#9ca3af") // Brighter gray for done (5:1 contrast)
	ColorCritical   = lipgloss.Color("#ef4444") // Red for critical priority
	ColorHigh       = lipgloss.Color("#f97316") // Orange for high priority
)

// Style presets
var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorFg)

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Subtitle style
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// Task styles by status
	TaskReadyStyle = lipgloss.NewStyle().
			Foreground(ColorReady)

	TaskInProgressStyle = lipgloss.NewStyle().
				Foreground(ColorInProgress).
				Bold(true)

	TaskBlockedStyle = lipgloss.NewStyle().
				Foreground(ColorBlocked)

	TaskDoneStyle = lipgloss.NewStyle().
			Foreground(ColorDone).
			Strikethrough(true)

	// Priority styles
	PriorityCriticalStyle = lipgloss.NewStyle().
				Foreground(ColorCritical).
				Bold(true)

	PriorityHighStyle = lipgloss.NewStyle().
				Foreground(ColorHigh).
				Bold(true)

	PriorityNormalStyle = lipgloss.NewStyle().
				Foreground(ColorFg)

	PriorityLowStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSlate).
			Padding(1, 2)

	// Selected item style - high contrast white on dark blue
	SelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2d4a7c")). // Darker blue for better contrast
			Foreground(lipgloss.Color("#ffffff")). // Pure white text
			Bold(true)

	// Selected description style - slightly dimmer but still readable
	SelectedDescStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2d4a7c")).
				Foreground(lipgloss.Color("#c0c0c0")) // Light gray for description

	// Help style - uses ColorSlate for better contrast on dark backgrounds
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorSlate)

	// Keybind style
	KeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	// Tag style
	TagStyle = lipgloss.NewStyle().
			Foreground(ColorPurple).
			Background(lipgloss.Color("#2d2d2d")).
			Padding(0, 1)

	// Timer running style - no blink to avoid distracting/triggering users
	TimerRunningStyle = lipgloss.NewStyle().
				Foreground(ColorAmber).
				Bold(true)

	// Learning/rule style
	RuleStyle = lipgloss.NewStyle().
			Foreground(ColorAmber).
			Bold(true)

	LearningStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// Filter match style - highlight matched characters during search
	FilterMatchStyle = lipgloss.NewStyle().
				Foreground(ColorAmber).
				Bold(true).
				Underline(true)
)

// StatusSymbol returns a colored symbol for a task status
func StatusSymbol(status string) string {
	switch status {
	case "ready":
		return TaskReadyStyle.Render("○")
	case "in_progress":
		return TaskInProgressStyle.Render("●")
	case "blocked":
		return TaskBlockedStyle.Render("◌")
	case "done":
		return TaskDoneStyle.Render("✓")
	default:
		return "?"
	}
}

// PrioritySymbol returns a colored priority indicator
func PrioritySymbol(priority int) string {
	switch priority {
	case 0: // Critical
		return PriorityCriticalStyle.Render("!!!")
	case 1: // High
		return PriorityHighStyle.Render("!!")
	case 2: // Normal
		return PriorityNormalStyle.Render("·")
	case 3: // Low
		return PriorityLowStyle.Render("·")
	case 4: // Backlog
		return PriorityLowStyle.Render("·")
	default:
		return ""
	}
}
