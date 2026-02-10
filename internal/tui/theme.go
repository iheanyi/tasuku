// Package tui provides a terminal user interface for tasuku.
package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Theme colors (TerminalColor allows NoColor when NO_COLOR env set - https://no-color.org/)
var (
	ColorBg    lipgloss.TerminalColor = lipgloss.Color("#1a1a1a")
	ColorFg    lipgloss.TerminalColor = lipgloss.Color("#f0f0f0")
	ColorMuted lipgloss.TerminalColor = lipgloss.Color("#6b7280")
	ColorSlate lipgloss.TerminalColor = lipgloss.Color("#94a3b8")
	ColorAccent  lipgloss.TerminalColor = lipgloss.Color("#5eead4")
	ColorPrimary lipgloss.TerminalColor = lipgloss.Color("#5c9eff")
	ColorPurple  lipgloss.TerminalColor = lipgloss.Color("#c4b5fd")
	ColorAmber   lipgloss.TerminalColor = lipgloss.Color("#fbbf24")
	ColorReady      lipgloss.TerminalColor = lipgloss.Color("#5eead4")
	ColorInProgress lipgloss.TerminalColor = lipgloss.Color("#fbbf24")
	ColorBlocked    lipgloss.TerminalColor = lipgloss.Color("#ef4444")
	ColorDone       lipgloss.TerminalColor = lipgloss.Color("#9ca3af")
	ColorCritical   lipgloss.TerminalColor = lipgloss.Color("#ef4444")
	ColorHigh       lipgloss.TerminalColor = lipgloss.Color("#f97316")
)

// Style presets (built in init after NO_COLOR check)
var (
	TitleStyle, TaskReadyStyle, TaskInProgressStyle, TaskBlockedStyle, TaskDoneStyle lipgloss.Style
	PriorityCriticalStyle, PriorityHighStyle, PriorityNormalStyle, PriorityLowStyle     lipgloss.Style
	PanelStyle, SelectedStyle, SelectedDescStyle, HelpStyle, KeyStyle, TagStyle       lipgloss.Style
	TimerRunningStyle, RuleStyle, FilterMatchStyle                                   lipgloss.Style
)

func init() {
	disabled := os.Getenv("NO_COLOR") != ""
	nc := lipgloss.NoColor{}
	if disabled {
		ColorBg, ColorFg, ColorMuted, ColorSlate = nc, nc, nc, nc
		ColorAccent, ColorPrimary, ColorPurple, ColorAmber = nc, nc, nc, nc
		ColorReady, ColorInProgress, ColorBlocked, ColorDone = nc, nc, nc, nc
		ColorCritical, ColorHigh = nc, nc
	}

	// Inline colors for SelectedStyle, SelectedDescStyle, TagStyle
	var bg, fg, descFg, tagBg lipgloss.TerminalColor
	if disabled {
		bg, fg, descFg, tagBg = nc, nc, nc, nc
	} else {
		bg = lipgloss.Color("#2d4a7c")
		fg = lipgloss.Color("#ffffff")
		descFg = lipgloss.Color("#c0c0c0")
		tagBg = lipgloss.Color("#2d2d2d")
	}

	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).MarginBottom(1)
	TaskReadyStyle = lipgloss.NewStyle().Foreground(ColorReady)
	TaskInProgressStyle = lipgloss.NewStyle().Foreground(ColorInProgress).Bold(true)
	TaskBlockedStyle = lipgloss.NewStyle().Foreground(ColorBlocked)
	TaskDoneStyle = lipgloss.NewStyle().Foreground(ColorDone).Strikethrough(true)
	PriorityCriticalStyle = lipgloss.NewStyle().Foreground(ColorCritical).Bold(true)
	PriorityHighStyle = lipgloss.NewStyle().Foreground(ColorHigh).Bold(true)
	PriorityNormalStyle = lipgloss.NewStyle().Foreground(ColorFg)
	PriorityLowStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	PanelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorSlate).Padding(1, 2)
	SelectedStyle = lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(true)
	SelectedDescStyle = lipgloss.NewStyle().Background(bg).Foreground(descFg)
	HelpStyle = lipgloss.NewStyle().Foreground(ColorSlate)
	KeyStyle = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	TagStyle = lipgloss.NewStyle().Foreground(ColorPurple).Background(tagBg).Padding(0, 1)
	TimerRunningStyle = lipgloss.NewStyle().Foreground(ColorAmber).Bold(true)
	RuleStyle = lipgloss.NewStyle().Foreground(ColorAmber).Bold(true)
	FilterMatchStyle = lipgloss.NewStyle().Foreground(ColorAmber).Bold(true).Underline(true)
}

// progressColorA and progressColorB return hex strings for the progress bar gradient.
// When NO_COLOR is set, returns neutral grays so the bar remains visible.
func progressColorA() string {
	if _, ok := ColorMuted.(lipgloss.NoColor); ok {
		return "#6b7280"
	}
	if c, ok := ColorMuted.(lipgloss.Color); ok {
		return string(c)
	}
	return "#6b7280"
}
func progressColorB() string {
	if _, ok := ColorAccent.(lipgloss.NoColor); ok {
		return "#94a3b8"
	}
	if c, ok := ColorAccent.(lipgloss.Color); ok {
		return string(c)
	}
	return "#94a3b8"
}

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
