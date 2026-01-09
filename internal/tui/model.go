package tui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

// View represents the current screen
type View int

const (
	ViewDashboard View = iota
	ViewTaskDetail
	ViewLearnings
	ViewDecisions
	ViewNotes
	ViewHelp
	ViewConfirm
	ViewCreate
	ViewEdit
)

// SortMode represents how tasks are sorted
type SortMode int

const (
	SortByStatus   SortMode = iota // Default: status then priority
	SortByPriority                 // Priority first, then status
)

// StatusFilter represents which statuses to show
type StatusFilter int

const (
	FilterAll        StatusFilter = iota // Show all tasks
	FilterReady                          // Only ready tasks
	FilterInProgress                     // Only in_progress tasks
	FilterBlocked                        // Only blocked tasks
	FilterDone                           // Only done tasks
)

// Model is the main TUI model
type Model struct {
	store        store.Storage
	file         *task.File
	view         View
	prevView     View // previous view before help overlay
	taskList     list.Model
	progress     progress.Model
	selected     string // selected task ID
	width        int
	height       int
	err          error
	sortMode     SortMode     // current sort mode
	statusFilter StatusFilter // current status filter

	// Async state
	loading         bool   // true if an action is in progress
	lastSelectedID  string // ID selected before refresh
	lastSelectedIdx int    // Index selected before refresh

	// Markdown renderer for rich content
	mdRenderer *glamour.TermRenderer

	// Confirmation dialog state
	confirmAction  string // what action to confirm (e.g., "archive", "bulk_archive")
	confirmTaskID  string // which task (for single task actions)
	confirmMessage string // message to show in the dialog

	// Task creation state
	createInput textarea.Model // textarea for new task description

	// Task editing state
	editInput  textarea.Model // textarea for editing task
	editTaskID string         // ID of task being edited
}

// TaskItem implements list.Item for the task list
type TaskItem struct {
	ID   string
	Task task.Task
}

func (i TaskItem) Title() string {
	// Use plain unicode symbols - styling applied by delegate
	symbol := statusSymbolPlain(string(i.Task.Status))
	priority := prioritySymbolPlain(i.Task.GetPriority())
	return fmt.Sprintf("%s %s %s", symbol, priority, i.ID)
}

// Plain symbols without styling for list filtering compatibility
func statusSymbolPlain(status string) string {
	switch status {
	case "ready":
		return "○"
	case "in_progress":
		return "●"
	case "blocked":
		return "◌"
	case "done":
		return "✓"
	default:
		return "?"
	}
}

func prioritySymbolPlain(priority int) string {
	switch priority {
	case 0: // Critical
		return "!!!"
	case 1: // High
		return "!!"
	case 2, 3, 4: // Normal, Low, Backlog
		return "·"
	default:
		return ""
	}
}

func (i TaskItem) Description() string {
	desc := safeTruncate(i.Task.Description, 60)

	var extras []string
	if len(i.Task.Tags) > 0 {
		extras = append(extras, fmt.Sprintf("[%s]", strings.Join(i.Task.Tags, ",")))
	}
	if i.Task.IsTimerRunning() {
		extras = append(extras, "⏱️")
	}
	if len(extras) > 0 {
		desc += " " + strings.Join(extras, " ")
	}
	return desc
}

func (i TaskItem) FilterValue() string {
	return i.ID + " " + i.Task.Description
}

// TaskDelegate is a custom delegate that colors items based on status
type TaskDelegate struct {
	list.DefaultDelegate
}

func NewTaskDelegate() TaskDelegate {
	d := TaskDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	return d
}

func (d TaskDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti, ok := item.(TaskItem)
	if !ok {
		d.DefaultDelegate.Render(w, m, index, item)
		return
	}

	// Determine if this item is selected
	isSelected := index == m.Index()

	// Get base styles
	var titleStyle, descStyle lipgloss.Style

	if isSelected {
		titleStyle = SelectedStyle
		descStyle = SelectedDescStyle
	} else {
		// Apply status-based colors to non-selected items
		switch ti.Task.Status {
		case task.StatusReady:
			titleStyle = lipgloss.NewStyle().Foreground(ColorReady)
		case task.StatusInProgress:
			titleStyle = lipgloss.NewStyle().Foreground(ColorInProgress).Bold(true)
		case task.StatusBlocked:
			titleStyle = lipgloss.NewStyle().Foreground(ColorBlocked)
		case task.StatusDone:
			titleStyle = lipgloss.NewStyle().Foreground(ColorDone).Strikethrough(true)
		default:
			titleStyle = lipgloss.NewStyle().Foreground(ColorFg)
		}
		descStyle = lipgloss.NewStyle().Foreground(ColorSlate) // #94a3b8 - brighter for readability
	}

	// Get plain text for title and description (without ANSI codes)
	titleText := ti.Title()
	descText := ti.Description()

	// Check if filter is active and highlight matched characters
	var title, desc string
	matches := m.MatchesForItem(index)

	if len(matches) > 0 && !isSelected {
		// Highlight matched characters in title
		title = highlightMatches(titleText, matches, titleStyle, FilterMatchStyle)
		// Description doesn't get filter matches highlighted in standard usage
		desc = descStyle.Render(descText)
	} else {
		title = titleStyle.Render(titleText)
		desc = descStyle.Render(descText)
	}

	// Add spacing and cursor
	cursor := "  "
	if isSelected {
		cursor = "> "
	}

	fmt.Fprintf(w, "%s%s\n%s%s", cursor, title, "  ", desc)
}

// highlightMatches renders text with matched positions highlighted
func highlightMatches(text string, matches []int, baseStyle, matchStyle lipgloss.Style) string {
	if len(matches) == 0 {
		return baseStyle.Render(text)
	}

	// Convert matches to a set for O(1) lookup
	matchSet := make(map[int]bool)
	for _, m := range matches {
		matchSet[m] = true
	}

	// Build the styled string character by character
	var result strings.Builder
	runes := []rune(text)

	for i, r := range runes {
		char := string(r)
		if matchSet[i] {
			result.WriteString(matchStyle.Render(char))
		} else {
			result.WriteString(baseStyle.Render(char))
		}
	}

	return result.String()
}

// TasksLoadedMsg carries the loaded file
type TasksLoadedMsg struct {
	File *task.File
	Err  error
}

// ActionMsg carries the result of an action
type ActionMsg struct {
	Err error
}

// Command constructors
func loadTasksCmd(s store.Storage) tea.Cmd {
	return func() tea.Msg {
		f, err := s.Read()
		return TasksLoadedMsg{File: f, Err: err}
	}
}

func setStatusCmd(s store.Storage, id string, status task.Status) tea.Cmd {
	return func() tea.Msg {
		return ActionMsg{Err: s.SetStatus(id, status)}
	}
}

func archiveTaskCmd(s store.Storage, id string) tea.Cmd {
	return func() tea.Msg {
		return ActionMsg{Err: s.ArchiveTask(id, "")}
	}
}

func bulkArchiveCmd(s store.Storage, doneIDs []string) tea.Cmd {
	return func() tea.Msg {
		for _, id := range doneIDs {
			if err := s.ArchiveTask(id, ""); err != nil {
				return ActionMsg{Err: err}
			}
		}
		return ActionMsg{Err: nil}
	}
}

func deleteTaskCmd(s store.Storage, id string) tea.Cmd {
	return func() tea.Msg {
		return ActionMsg{Err: s.DeleteTask(id)}
	}
}

func addTaskCmd(s store.Storage, id, desc string) tea.Cmd {
	return func() tea.Msg {
		return ActionMsg{Err: s.AddTask(id, desc)}
	}
}

func setDescriptionCmd(s store.Storage, id, desc string) tea.Cmd {
	return func() tea.Msg {
		return ActionMsg{Err: s.SetDescription(id, desc)}
	}
}

func startTimerCmd(s store.Storage, id string) tea.Cmd {
	return func() tea.Msg {
		return ActionMsg{Err: s.StartTimer(id)}
	}
}

func stopTimerCmd(s store.Storage, id string) tea.Cmd {
	return func() tea.Msg {
		_, err := s.StopTimer(id)
		return ActionMsg{Err: err}
	}
}

func unblockTaskCmd(s store.Storage, id string) tea.Cmd {
	return func() tea.Msg {
		_ = s.UnblockTask(id)
		return ActionMsg{Err: s.SetStatus(id, task.StatusReady)}
	}
}

func pauseTaskCmd(s store.Storage, id string) tea.Cmd {
	return func() tea.Msg {
		// Best effort stop timer
		s.StopTimerIfRunning(id)
		return ActionMsg{Err: s.SetStatus(id, task.StatusReady)}
	}
}

// New creates a new TUI model
func New(s store.Storage) (*Model, error) {
	// Initialize progress bar with theme colors
	prog := progress.New(
		progress.WithScaledGradient(string(ColorMuted), string(ColorAccent)),
		progress.WithoutPercentage(),
	)

	// Initialize markdown renderer for rich content display
	mdRenderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	m := &Model{
		store:      s,
		file:       task.NewFile(), // Start empty
		view:       ViewDashboard,
		progress:   prog,
		mdRenderer: mdRenderer,
		loading:    true, // Start in loading state
	}

	// We don't load synchronously anymore. Init() will trigger the load.
	return m, nil
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return loadTasksCmd(m.store)
}

func (m *Model) initTaskList() {
	// Convert tasks to list items, applying filter and sort
	var items []list.Item

	type sortableTask struct {
		id   string
		task task.Task
	}

	var sortable []sortableTask
	for id, t := range m.file.Tasks {
		// Apply status filter
		if m.statusFilter != FilterAll {
			switch m.statusFilter {
			case FilterReady:
				if t.Status != task.StatusReady {
					continue
				}
			case FilterInProgress:
				if t.Status != task.StatusInProgress {
					continue
				}
			case FilterBlocked:
				if t.Status != task.StatusBlocked {
					continue
				}
			case FilterDone:
				if t.Status != task.StatusDone {
					continue
				}
			}
		}
		sortable = append(sortable, sortableTask{id: id, task: t})
	}

	// Sort based on current sort mode
	statusOrder := map[task.Status]int{
		task.StatusInProgress: 0,
		task.StatusReady:      1,
		task.StatusBlocked:    2,
		task.StatusDone:       3,
	}

	sort.Slice(sortable, func(i, j int) bool {
		if m.sortMode == SortByPriority {
			// Priority first, then status
			pi, pj := sortable[i].task.GetPriority(), sortable[j].task.GetPriority()
			if pi != pj {
				return pi < pj
			}
			return statusOrder[sortable[i].task.Status] < statusOrder[sortable[j].task.Status]
		}
		// Default: status first, then priority
		si, sj := statusOrder[sortable[i].task.Status], statusOrder[sortable[j].task.Status]
		if si != sj {
			return si < sj
		}
		return sortable[i].task.GetPriority() < sortable[j].task.GetPriority()
	})

	for _, st := range sortable {
		items = append(items, TaskItem{ID: st.id, Task: st.task})
	}

	delegate := NewTaskDelegate()

	m.taskList = list.New(items, delegate, 0, 0)
	m.taskList.Title = m.getListTitle()
	m.taskList.SetShowStatusBar(true)
	m.taskList.SetFilteringEnabled(true)
	m.taskList.Styles.Title = TitleStyle

	// Restore size if already set
	if m.width > 0 && m.height > 0 {
		m.taskList.SetSize(m.width-4, m.height-8)
		m.progress.Width = m.width - 4
	}
}

func (m *Model) initCreateInput() {
	ta := textarea.New()
	ta.Placeholder = "Enter task description...\n(Shift+Enter for newlines, Enter to submit)"
	ta.CharLimit = 2000 // Allow much longer descriptions
	ta.SetWidth(60)
	ta.SetHeight(5) // Multi-line input
	ta.ShowLineNumbers = false
	m.createInput = ta
}

func (m *Model) initEditInput(taskID string, currentDesc string) {
	ta := textarea.New()
	ta.Placeholder = "Edit task description...\n(Shift+Enter for newlines, Enter to submit)"
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(5)
	ta.ShowLineNumbers = false
	ta.SetValue(currentDesc)
	m.editInput = ta
	m.editTaskID = taskID
}

func (m *Model) getListTitle() string {
	title := "Tasks"
	switch m.statusFilter {
	case FilterReady:
		title = "Tasks (Ready)"
	case FilterInProgress:
		title = "Tasks (In Progress)"
	case FilterBlocked:
		title = "Tasks (Blocked)"
	case FilterDone:
		title = "Tasks (Done)"
	}
	if m.sortMode == SortByPriority {
		title += " [Priority↑]"
	}
	return title
}

// safeTruncate truncates a string to maxLen runes to handle multi-byte chars safely.
func safeTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen-3]) + "..."
}

// KeyMap defines the key bindings
type KeyMap struct {
	Quit        key.Binding
	Refresh     key.Binding
	Start       key.Binding
	Done        key.Binding
	Enter       key.Binding
	Back        key.Binding
	Timer       key.Binding
	Archive     key.Binding
	BulkArchive key.Binding
	Help        key.Binding
	FilterAll   key.Binding
	FilterReady key.Binding
	FilterProg  key.Binding
	FilterBlock key.Binding
	FilterDone  key.Binding
	ToggleSort  key.Binding
	ConfirmYes  key.Binding
	ConfirmNo   key.Binding
	NewTask     key.Binding
	EditTask    key.Binding
	Notes       key.Binding
	Learnings   key.Binding
	Decisions   key.Binding
	Delete      key.Binding
	Block       key.Binding
	Unblock     key.Binding
	Pause       key.Binding
}

var keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start task"),
	),
	Done: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "mark done"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view details"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Timer: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle timer"),
	),
	Archive: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "archive"),
	),
	BulkArchive: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "archive all done"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	FilterAll: key.NewBinding(
		key.WithKeys("0"),
		key.WithHelp("0", "all tasks"),
	),
	FilterReady: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "ready"),
	),
	FilterProg: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "in progress"),
	),
	FilterBlock: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "blocked"),
	),
	FilterDone: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "done"),
	),
	ToggleSort: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "toggle priority sort"),
	),
	ConfirmYes: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm"),
	),
	ConfirmNo: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "cancel"),
	),
	NewTask: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new task"),
	),
	EditTask: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit task"),
	),
	Notes: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "view notes"),
	),
	Learnings: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "view learnings"),
	),
	Decisions: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "view decisions"),
	),
	Delete: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete task"),
	),
	Block: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "block task"),
	),
	Unblock: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unblock task"),
	),
	Pause: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "pause task"),
	),
}

// countDoneTasks returns the number of tasks with done status
func (m Model) countDoneTasks() int {
	count := 0
	for _, t := range m.file.Tasks {
		if t.Status == task.StatusDone {
			count++
		}
	}
	return count
}

// runAction executes a command and sets loading state
func (m *Model) runAction(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.loading = true
	return m, cmd
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Handle Async Messages
	case TasksLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.file = msg.File
		m.initTaskList()
		m.restoreSelection(m.lastSelectedID, m.lastSelectedIdx)
		return m, nil

	case ActionMsg:
		if msg.Err != nil {
			m.loading = false
			m.err = msg.Err
			return m, nil
		}
		// Action success, refresh data
		return m.refresh()

	case tea.KeyMsg:
		// Handle confirmation dialog
		if m.view == ViewConfirm {
			switch {
			case key.Matches(msg, keys.ConfirmYes):
				// Execute the confirmed action
				m.view = ViewDashboard
				action := m.confirmAction
				id := m.confirmTaskID
				
				// Reset confirm state
				m.confirmAction = ""
				m.confirmTaskID = ""
				m.confirmMessage = ""

				switch action {
				case "archive":
					return m.runAction(archiveTaskCmd(m.store, id))
				case "bulk_archive":
					// Collect done task IDs
					var doneIDs []string
					for id, t := range m.file.Tasks {
						if t.Status == task.StatusDone {
							doneIDs = append(doneIDs, id)
						}
					}
					return m.runAction(bulkArchiveCmd(m.store, doneIDs))
				case "delete":
					return m.runAction(deleteTaskCmd(m.store, id))
				}
				return m, nil

			case key.Matches(msg, keys.ConfirmNo), key.Matches(msg, keys.Back):
				// Cancel the action
				m.view = ViewDashboard
				m.confirmAction = ""
				m.confirmTaskID = ""
				m.confirmMessage = ""
				return m, nil
			}
			return m, nil
		}

		// Handle ViewCreate mode - textarea for new task
		if m.view == ViewCreate {
			// Ctrl+S or Ctrl+Enter to submit
			if msg.String() == "ctrl+s" || (msg.Type == tea.KeyEnter && msg.Alt) {
				// Create the task
				desc := strings.TrimSpace(m.createInput.Value())
				m.createInput.Reset()
				m.view = ViewDashboard
				if desc != "" {
					id := generateTaskID(desc)
					return m.runAction(addTaskCmd(m.store, id, desc))
				}
				return m, nil
			}
			if msg.Type == tea.KeyEsc {
				// Cancel creation
				m.createInput.Reset()
				m.view = ViewDashboard
				return m, nil
			}
			// Update textarea
			var cmd tea.Cmd
			m.createInput, cmd = m.createInput.Update(msg)
			return m, cmd
		}

		// Handle ViewEdit mode - textarea for editing task description
		if m.view == ViewEdit {
			// Ctrl+S or Alt+Enter to submit
			if msg.String() == "ctrl+s" || (msg.Type == tea.KeyEnter && msg.Alt) {
				// Update the task description
				desc := strings.TrimSpace(m.editInput.Value())
				id := m.editTaskID
				m.editInput.Reset()
				m.editTaskID = ""
				m.view = ViewDashboard
				if desc != "" && id != "" {
					return m.runAction(setDescriptionCmd(m.store, id, desc))
				}
				return m, nil
			}
			if msg.Type == tea.KeyEsc {
				// Cancel editing
				m.editInput.Reset()
				m.editTaskID = ""
				m.view = ViewDashboard
				return m, nil
			}
			// Update textarea
			var cmd tea.Cmd
			m.editInput, cmd = m.editInput.Update(msg)
			return m, cmd
		}

		// When filtering is active, only handle quit/escape - let list handle the rest
		if m.view == ViewDashboard && m.taskList.SettingFilter() {
			switch {
			case key.Matches(msg, keys.Quit):
				// ctrl+c should still quit the app
				if msg.String() == "ctrl+c" {
					return m, tea.Quit
				}
				// 'q' should go to filter input, not quit
			}
			// Pass all other keys to the list for filter input
			var cmd tea.Cmd
			m.taskList, cmd = m.taskList.Update(msg)
			return m, cmd
		}

		// Normal keybinding handling when NOT filtering
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Refresh):
			return m.refresh()

		case key.Matches(msg, keys.Help):
			if m.view == ViewHelp {
				// Close help overlay, return to previous view
				m.view = m.prevView
				return m, nil
			}
			// Open help overlay
			m.prevView = m.view
			m.view = ViewHelp
			return m, nil

		case key.Matches(msg, keys.Back):
			if m.view == ViewHelp {
				// Close help overlay
				m.view = m.prevView
				return m, nil
			}
			if m.view != ViewDashboard {
				m.view = ViewDashboard
				return m, nil
			}
			// On dashboard, escape does nothing - prevent passing to list
			// which would clear filter state and cause visual glitches
			return m, nil

		case key.Matches(msg, keys.Enter):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					m.selected = item.ID
					m.view = ViewTaskDetail
					return m, nil
				}
			}

		case key.Matches(msg, keys.Start):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.Status == task.StatusReady {
						return m.runAction(setStatusCmd(m.store, item.ID, task.StatusInProgress))
					}
				}
			}

		case key.Matches(msg, keys.Done):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.Status == task.StatusInProgress {
						return m.runAction(setStatusCmd(m.store, item.ID, task.StatusDone))
					}
				}
			}

		case key.Matches(msg, keys.Timer):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.IsTimerRunning() {
						return m.runAction(stopTimerCmd(m.store, item.ID))
					} else {
						return m.runAction(startTimerCmd(m.store, item.ID))
					}
				}
			}

		case key.Matches(msg, keys.Archive):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.Status == task.StatusDone {
						// Show confirmation dialog
						m.confirmAction = "archive"
						m.confirmTaskID = item.ID
						m.confirmMessage = fmt.Sprintf("Archive task '%s'? (y/n)", item.ID)
						m.view = ViewConfirm
						return m, nil
					}
				}
			}

		case key.Matches(msg, keys.BulkArchive):
			if m.view == ViewDashboard {
				doneCount := m.countDoneTasks()
				if doneCount > 0 {
					// Show confirmation dialog for bulk archive
					m.confirmAction = "bulk_archive"
					m.confirmTaskID = ""
					m.confirmMessage = fmt.Sprintf("Archive all %d done tasks? (y/n)", doneCount)
					m.view = ViewConfirm
					return m, nil
				}
			}

		// Status filter keys (0-4)
		case key.Matches(msg, keys.FilterAll):
			if m.view == ViewDashboard {
				m.statusFilter = FilterAll
				m.initTaskList()
				return m, nil
			}
		case key.Matches(msg, keys.FilterReady):
			if m.view == ViewDashboard {
				m.statusFilter = FilterReady
				m.initTaskList()
				return m, nil
			}
		case key.Matches(msg, keys.FilterProg):
			if m.view == ViewDashboard {
				m.statusFilter = FilterInProgress
				m.initTaskList()
				return m, nil
			}
		case key.Matches(msg, keys.FilterBlock):
			if m.view == ViewDashboard {
				m.statusFilter = FilterBlocked
				m.initTaskList()
				return m, nil
			}
		case key.Matches(msg, keys.FilterDone):
			if m.view == ViewDashboard {
				m.statusFilter = FilterDone
				m.initTaskList()
				return m, nil
			}

		// Sort toggle
		case key.Matches(msg, keys.ToggleSort):
			if m.view == ViewDashboard {
				if m.sortMode == SortByStatus {
					m.sortMode = SortByPriority
				} else {
					m.sortMode = SortByStatus
				}
				m.initTaskList()
				return m, nil
			}

		// New task
		case key.Matches(msg, keys.NewTask):
			if m.view == ViewDashboard {
				m.initCreateInput()
				m.view = ViewCreate
				return m, m.createInput.Focus()
			}

		// Edit task
		case key.Matches(msg, keys.EditTask):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					m.initEditInput(item.ID, item.Task.Description)
					m.view = ViewEdit
					return m, m.editInput.Focus()
				}
			}

		// View notes
		case key.Matches(msg, keys.Notes):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					m.selected = item.ID
					m.prevView = m.view
					m.view = ViewNotes
					return m, nil
				}
			}

		// View learnings
		case key.Matches(msg, keys.Learnings):
			if m.view == ViewDashboard {
				m.prevView = m.view
				m.view = ViewLearnings
				return m, nil
			}

		// View decisions
		case key.Matches(msg, keys.Decisions):
			if m.view == ViewDashboard {
				m.prevView = m.view
				m.view = ViewDecisions
				return m, nil
			}

		// Delete task
		case key.Matches(msg, keys.Delete):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					// Show confirmation dialog
					m.confirmAction = "delete"
					m.confirmTaskID = item.ID
					m.confirmMessage = fmt.Sprintf("Delete task '%s'? (y/n)", item.ID)
					m.view = ViewConfirm
					return m, nil
				}
			}

		// Block task
		case key.Matches(msg, keys.Block):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.Status == task.StatusReady || item.Task.Status == task.StatusInProgress {
						return m.runAction(setStatusCmd(m.store, item.ID, task.StatusBlocked))
					}
				}
			}

		// Unblock task
		case key.Matches(msg, keys.Unblock):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.Status == task.StatusBlocked {
						return m.runAction(unblockTaskCmd(m.store, item.ID))
					}
				}
			}

		// Pause task
		case key.Matches(msg, keys.Pause):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.Status == task.StatusInProgress {
						return m.runAction(pauseTaskCmd(m.store, item.ID))
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.taskList.SetSize(msg.Width-4, msg.Height-8)
		m.progress.Width = msg.Width - 4
	}

	// Update task list
	if m.view == ViewDashboard {
		var cmd tea.Cmd
		m.taskList, cmd = m.taskList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) refresh() (tea.Model, tea.Cmd) {
	// Save current selection before refresh
	if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
		m.lastSelectedID = item.ID
		m.lastSelectedIdx = m.taskList.Index()
	} else {
		m.lastSelectedID = ""
		m.lastSelectedIdx = 0
	}

	m.loading = true
	return m, loadTasksCmd(m.store)
}

// restoreSelection tries to select a task by ID, or falls back to maintaining index position
func (m *Model) restoreSelection(taskID string, previousIdx int) {
	itemCount := len(m.taskList.Items())
	if itemCount == 0 {
		return
	}

	if taskID == "" {
		// No previous selection, select first item
		m.taskList.Select(0)
		return
	}

	// Try to find and select the task by ID
	for i, item := range m.taskList.Items() {
		if ti, ok := item.(TaskItem); ok && ti.ID == taskID {
			m.taskList.Select(i)
			return
		}
	}

	// Task no longer exists (deleted/archived), maintain position or go to last item
	if previousIdx >= itemCount {
		m.taskList.Select(itemCount - 1)
	} else {
		m.taskList.Select(previousIdx)
	}
}

// View implements tea.Model
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	switch m.view {
	case ViewHelp:
		return m.viewHelpOverlay()
	case ViewConfirm:
		return m.viewConfirmOverlay()
	case ViewCreate:
		return m.viewCreate()
	case ViewEdit:
		return m.viewEdit()
	case ViewTaskDetail:
		return m.viewTaskDetail()
	case ViewNotes:
		return m.viewNotes()
	case ViewLearnings:
		return m.viewLearnings()
	case ViewDecisions:
		return m.viewDecisions()
	default:
		return m.viewDashboard()
	}
}

func (m Model) viewDashboard() string {
	// Stats header
	var ready, inProgress, blocked, done int
	for _, t := range m.file.Tasks {
		switch t.Status {
		case task.StatusReady:
			ready++
		case task.StatusInProgress:
			inProgress++
		case task.StatusBlocked:
			blocked++
		case task.StatusDone:
			done++
		}
	}

	total := ready + inProgress + blocked + done

	statsStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginBottom(0)

	stats := statsStyle.Render(fmt.Sprintf(
		"Ready: %s  In Progress: %s  Blocked: %s  Done: %s",
		TaskReadyStyle.Render(fmt.Sprintf("%d", ready)),
		TaskInProgressStyle.Render(fmt.Sprintf("%d", inProgress)),
		TaskBlockedStyle.Render(fmt.Sprintf("%d", blocked)),
		TaskDoneStyle.Render(fmt.Sprintf("%d", done)),
	))

	// Progress bar showing done/total
	var progressPercent float64
	if total > 0 {
		progressPercent = float64(done) / float64(total)
	}
	progressLabel := HelpStyle.Render(fmt.Sprintf("Progress: %d/%d ", done, total))
	progressBar := m.progress.ViewAs(progressPercent)
	progressLine := lipgloss.NewStyle().MarginBottom(1).Render(progressLabel + progressBar)

	help1 := HelpStyle.Render("n:new  e:edit  s:start  d:done  P:pause  b:block  u:unblock  x:delete  t:timer  a:archive")
	help2 := HelpStyle.Render("0-4:status  p:priority  N:notes  L:learnings  D:decisions  /:filter  r:refresh  ?:help  q:quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		stats,
		progressLine,
		m.taskList.View(),
		help1,
		help2,
	)
}

func (m Model) viewConfirmOverlay() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAmber).
		Padding(1, 3)

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorAmber).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(ColorFg).
		MarginTop(1).
		MarginBottom(1)

	modalContent := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Confirm"),
		messageStyle.Render(m.confirmMessage),
		HelpStyle.Render("y: confirm  n/esc: cancel"),
	)

	modal := modalStyle.Render(modalContent)

	// Center the modal on screen using lipgloss.Place (handles ANSI correctly)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) viewHelpOverlay() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent).
		Padding(1, 3)

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	// Group help items by category for better readability
	helpContent := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Keyboard Shortcuts"),
		"",
		HelpStyle.Render("Tasks"),
		"  "+KeyStyle.Render("n")+"      New task",
		"  "+KeyStyle.Render("e")+"      Edit task description",
		"  "+KeyStyle.Render("s")+"      Start ready task",
		"  "+KeyStyle.Render("d")+"      Mark done",
		"  "+KeyStyle.Render("P")+"      Pause in-progress task",
		"  "+KeyStyle.Render("b")+"      Block task",
		"  "+KeyStyle.Render("u")+"      Unblock task",
		"  "+KeyStyle.Render("x")+"      Delete task",
		"  "+KeyStyle.Render("t")+"      Toggle timer",
		"  "+KeyStyle.Render("a")+"      Archive done task",
		"  "+KeyStyle.Render("A")+"      Archive all done",
		"  "+KeyStyle.Render("enter")+"  View details",
		"",
		HelpStyle.Render("Navigation"),
		"  "+KeyStyle.Render("/")+"      Filter tasks",
		"  "+KeyStyle.Render("0-4")+"    Status filter",
		"  "+KeyStyle.Render("p")+"      Priority sort",
		"  "+KeyStyle.Render("N")+"      Notes",
		"  "+KeyStyle.Render("L")+"      Learnings",
		"  "+KeyStyle.Render("D")+"      Decisions",
		"",
		HelpStyle.Render("General"),
		"  "+KeyStyle.Render("r")+"      Refresh",
		"  "+KeyStyle.Render("?")+"      Help",
		"  "+KeyStyle.Render("q")+"      Quit",
		"",
		HelpStyle.Render("?/esc: close"),
	)

	modal := modalStyle.Render(helpContent)

	// Center the modal on screen using lipgloss.Place (handles ANSI correctly)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) viewTaskDetail() string {
	t, exists := m.file.Tasks[m.selected]
	if !exists {
		errorPanel := PanelStyle.Width(m.width - 4).Render(
			HelpStyle.Render("Task not found.\n\nPress esc to go back"),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errorPanel)
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		MarginBottom(1)
	b.WriteString(titleStyle.Render(m.selected))
	b.WriteString("\n\n")

	// Status and Priority
	b.WriteString(fmt.Sprintf("Status: %s %s\n", StatusSymbol(string(t.Status)), t.Status))
	b.WriteString(fmt.Sprintf("Priority: %s %s\n", PrioritySymbol(t.GetPriority()), task.PriorityName(t.GetPriority())))
	b.WriteString("\n")

	// Description (render as Markdown if renderer available)
	sectionStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)
	b.WriteString(sectionStyle.Render("Description"))
	b.WriteString("\n")
	if m.mdRenderer != nil && containsMarkdown(t.Description) {
		rendered, err := m.mdRenderer.Render(t.Description)
		if err == nil {
			b.WriteString(strings.TrimSpace(rendered))
		} else {
			b.WriteString(t.Description)
		}
	} else {
		b.WriteString(t.Description)
	}
	b.WriteString("\n\n")

	// Timer
	if t.Duration > 0 || t.IsTimerRunning() {
		b.WriteString(sectionStyle.Render("Time Tracking"))
		b.WriteString("\n")
		if t.IsTimerRunning() {
			b.WriteString(TimerRunningStyle.Render("Timer running"))
			b.WriteString(" - ")
		}
		b.WriteString(fmt.Sprintf("Total: %s\n", task.Duration(t.CurrentDuration()).FormatHumanReadable()))
		b.WriteString("\n")
	}

	// Tags
	if len(t.Tags) > 0 {
		b.WriteString(sectionStyle.Render("Tags"))
		b.WriteString("\n")
		for _, tag := range t.Tags {
			b.WriteString(TagStyle.Render(tag))
			b.WriteString(" ")
		}
		b.WriteString("\n\n")
	}

	// Custom fields
	if len(t.Fields) > 0 {
		b.WriteString(sectionStyle.Render("Custom Fields"))
		b.WriteString("\n")
		for k, v := range t.Fields {
			b.WriteString(fmt.Sprintf("  %s: %s\n", KeyStyle.Render(k), v))
		}
		b.WriteString("\n")
	}

	// Notes
	if notes, exists := m.file.Context.Notes[m.selected]; exists && len(notes) > 0 {
		b.WriteString(sectionStyle.Render("Notes"))
		b.WriteString("\n")
		for _, note := range notes {
			b.WriteString(fmt.Sprintf("  - %s\n", note.Text))
		}
		b.WriteString("\n")
	}

	// Blocked by (filter out empty strings)
	var blockers []string
	for _, blocker := range t.BlockedBy {
		if blocker != "" {
			blockers = append(blockers, blocker)
		}
	}
	if len(blockers) > 0 {
		b.WriteString(TaskBlockedStyle.Render("Blocked by: "))
		b.WriteString(strings.Join(blockers, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString(HelpStyle.Render("esc: back"))

	panel := PanelStyle.Width(m.width - 4).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

// renderEmptyState creates a consistent empty state message with a CLI hint.
func (m Model) renderEmptyState(message string, cliCommand string) string {
	var b strings.Builder
	b.WriteString(HelpStyle.Render(message))
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("Use '"))
	b.WriteString(KeyStyle.Render(cliCommand))
	b.WriteString(HelpStyle.Render("' to add entries."))
	return b.String()
}

// renderModal creates a consistent modal dialog with title, content, and help text.
// Set useBorder to false for textarea content to avoid double-border artifacts.
func (m Model) renderModal(title string, borderColor lipgloss.Color, content string, helpText string, useBorder bool) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 3)

	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	// Apply inner border only when requested (not for textareas which have their own styling)
	var styledContent string
	if useBorder {
		inputStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorSlate).
			Padding(0, 1).
			Width(50)
		styledContent = inputStyle.Render(content)
	} else {
		// No border, just minimal padding for textareas
		styledContent = lipgloss.NewStyle().Padding(0, 1).Render(content)
	}

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		"",
		styledContent,
		"",
		HelpStyle.Render(helpText),
	)

	return modalStyle.Render(modalContent)
}

// generateTaskID creates a kebab-case ID from description.
func generateTaskID(desc string) string {
	result := ""
	for _, r := range desc {
		if r >= 'a' && r <= 'z' {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else if r == ' ' && len(result) > 0 && result[len(result)-1] != '-' {
			result += "-"
		}
	}
	result = strings.TrimSuffix(result, "-")
	if len(result) > 32 {
		result = result[:32]
	}
	return result
}

// containsMarkdown checks if text contains Markdown formatting.
// Returns true if there are indicators like headers, code blocks, emphasis, etc.
func containsMarkdown(text string) bool {
	// Check for common Markdown patterns
	patterns := []string{
		"```",  // Code blocks
		"**",   // Bold
		"__",   // Bold
		"*",    // Italic (if single)
		"_",    // Italic (if single)
		"# ",   // Headers
		"## ",  // Headers
		"- ",   // Lists
		"* ",   // Lists
		"1. ",  // Ordered lists
		"[",    // Links
		"`",    // Inline code
		"> ",   // Blockquotes
		"---",  // Horizontal rule
		"***",  // Horizontal rule
	}

	for _, p := range patterns {
		if strings.Contains(text, p) {
			return true
		}
	}
	return false
}

func (m Model) viewCreate() string {
	modal := m.renderModal(
		"New Task",
		ColorAccent,
		m.createInput.View(),
		"ctrl+s: create  esc: cancel  (Enter adds newlines)",
		false, // no inner border for textarea
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) viewEdit() string {
	idLabel := HelpStyle.Render(m.editTaskID)
	content := lipgloss.JoinVertical(lipgloss.Left, idLabel, "", m.editInput.View())
	modal := m.renderModal(
		"Edit Task",
		ColorPrimary,
		content,
		"ctrl+s: save  esc: cancel  (Enter adds newlines)",
		false, // no inner border for textarea
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) viewNotes() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	b.WriteString(titleStyle.Render(fmt.Sprintf("Notes for %s", m.selected)))
	b.WriteString("\n\n")

	notes, exists := m.file.Context.Notes[m.selected]
	if !exists || len(notes) == 0 {
		b.WriteString(m.renderEmptyState(
			"No notes recorded for this task.",
			"tk note <task-id> \"note text\"",
		))
	} else {
		for i, n := range notes {
			// Render note text as Markdown if it contains formatting
			noteText := n.Text
			if m.mdRenderer != nil && containsMarkdown(noteText) {
				rendered, err := m.mdRenderer.Render(noteText)
				if err == nil {
					noteText = strings.TrimSpace(rendered)
				}
			}
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, noteText))
			b.WriteString(fmt.Sprintf("     %s\n", HelpStyle.Render(task.FormatLocalTime(n.CreatedAt))))
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("esc: back"))

	panel := PanelStyle.Width(m.width - 4).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func (m Model) viewLearnings() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	sectionStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	b.WriteString(titleStyle.Render("Learnings"))
	b.WriteString("\n\n")

	if len(m.file.Context.Learnings) == 0 {
		b.WriteString(m.renderEmptyState(
			"No learnings recorded yet.",
			"tk learn \"insight\"",
		))
	} else {
		// Separate rules from regular learnings
		var rules, regular []task.Learning
		for _, l := range m.file.Context.Learnings {
			if l.IsRule {
				rules = append(rules, l)
			} else {
				regular = append(regular, l)
			}
		}

		// Show rules first (never/always)
		if len(rules) > 0 {
			b.WriteString(RuleStyle.Render("Rules (Never/Always)"))
			b.WriteString("\n")
			for _, r := range rules {
				b.WriteString(fmt.Sprintf("  %s %s\n", RuleStyle.Render("!"), r.Text))
			}
			b.WriteString("\n")
		}

		// Then regular learnings
		if len(regular) > 0 {
			b.WriteString(sectionStyle.Render("Insights"))
			b.WriteString("\n")
			for _, l := range regular {
				b.WriteString(fmt.Sprintf("  - %s\n", l.Text))
				b.WriteString(fmt.Sprintf("    %s\n", HelpStyle.Render(task.FormatLocalTime(l.CreatedAt))))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("esc: back"))

	panel := PanelStyle.Width(m.width - 4).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}

func (m Model) viewDecisions() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)

	choseStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	overStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Strikethrough(true)

	reasonStyle := lipgloss.NewStyle().
		Foreground(ColorFg).
		Italic(true)

	b.WriteString(titleStyle.Render("Decisions"))
	b.WriteString("\n\n")

	if len(m.file.Context.Decisions) == 0 {
		b.WriteString(m.renderEmptyState(
			"No decisions recorded yet.",
			"tk decide --chose X --over Y,Z --because \"reason\"",
		))
	} else {
		for _, d := range m.file.Context.Decisions {
			b.WriteString(fmt.Sprintf("  %s %s\n", choseStyle.Render("*"), choseStyle.Render(d.Chose)))
			if len(d.Over) > 0 {
				b.WriteString(fmt.Sprintf("    over: %s\n", overStyle.Render(strings.Join(d.Over, ", "))))
			}
			if d.Because != "" {
				b.WriteString(fmt.Sprintf("    %s\n", reasonStyle.Render("\""+d.Because+"\"")))
			}
			b.WriteString(fmt.Sprintf("    %s\n", HelpStyle.Render(task.FormatLocalTime(d.CreatedAt))))
			b.WriteString("\n")
		}
	}

	b.WriteString(HelpStyle.Render("esc: back"))

	panel := PanelStyle.Width(m.width - 4).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
