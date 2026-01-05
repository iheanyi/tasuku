package tui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

	// Confirmation dialog state
	confirmAction  string // what action to confirm (e.g., "archive", "bulk_archive")
	confirmTaskID  string // which task (for single task actions)
	confirmMessage string // message to show in the dialog

	// Task creation state
	createInput textinput.Model // text input for new task description

	// Task editing state
	editInput  textinput.Model // text input for editing task
	editTaskID string          // ID of task being edited
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
	desc := i.Task.Description
	if len(desc) > 60 {
		desc = desc[:57] + "..."
	}

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

// New creates a new TUI model
func New(s store.Storage) (*Model, error) {
	f, err := s.Read()
	if err != nil {
		return nil, err
	}

	// Initialize progress bar with theme colors
	prog := progress.New(
		progress.WithScaledGradient(string(ColorMuted), string(ColorAccent)),
		progress.WithoutPercentage(),
	)

	m := &Model{
		store:    s,
		file:     f,
		view:     ViewDashboard,
		progress: prog,
	}

	m.initTaskList()
	return m, nil
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
	ti := textinput.New()
	ti.Placeholder = "Enter task description..."
	ti.CharLimit = 256
	ti.Width = 50
	m.createInput = ti
}

func (m *Model) initEditInput(taskID string, currentDesc string) {
	ti := textinput.New()
	ti.Placeholder = "Edit task description..."
	ti.CharLimit = 256
	ti.Width = 50
	ti.SetValue(currentDesc)
	m.editInput = ti
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

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return nil
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

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle confirmation dialog
		if m.view == ViewConfirm {
			switch {
			case key.Matches(msg, keys.ConfirmYes):
				// Execute the confirmed action
				switch m.confirmAction {
				case "archive":
					_ = m.store.ArchiveTask(m.confirmTaskID, "")
				case "bulk_archive":
					// Archive all done tasks
					for id, t := range m.file.Tasks {
						if t.Status == task.StatusDone {
							_ = m.store.ArchiveTask(id, "")
						}
					}
				}
				m.view = ViewDashboard
				m.confirmAction = ""
				m.confirmTaskID = ""
				m.confirmMessage = ""
				return m.refresh()

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

		// Handle ViewCreate mode - text input for new task
		if m.view == ViewCreate {
			switch msg.Type {
			case tea.KeyEnter:
				// Create the task
				desc := m.createInput.Value()
				if desc != "" {
					id := generateTaskID(desc)
					if err := m.store.AddTask(id, desc); err != nil {
						m.err = err
					}
				}
				// Reset and return to dashboard
				m.createInput.Reset()
				m.view = ViewDashboard
				return m.refresh()
			case tea.KeyEsc:
				// Cancel creation
				m.createInput.Reset()
				m.view = ViewDashboard
				return m, nil
			}
			// Update text input
			var cmd tea.Cmd
			m.createInput, cmd = m.createInput.Update(msg)
			return m, cmd
		}

		// Handle ViewEdit mode - text input for editing task description
		if m.view == ViewEdit {
			switch msg.Type {
			case tea.KeyEnter:
				// Update the task description
				desc := m.editInput.Value()
				if desc != "" && m.editTaskID != "" {
					if err := m.store.SetDescription(m.editTaskID, desc); err != nil {
						m.err = err
					}
				}
				// Reset and return to dashboard
				m.editInput.Reset()
				m.editTaskID = ""
				m.view = ViewDashboard
				return m.refresh()
			case tea.KeyEsc:
				// Cancel editing
				m.editInput.Reset()
				m.editTaskID = ""
				m.view = ViewDashboard
				return m, nil
			}
			// Update text input
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
			f, err := m.store.Read()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.file = f
			m.initTaskList()
			return m, nil

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
						_ = m.store.SetStatus(item.ID, task.StatusInProgress)
						return m.refresh()
					}
				}
			}

		case key.Matches(msg, keys.Done):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.Status == task.StatusInProgress {
						_ = m.store.SetStatus(item.ID, task.StatusDone)
						return m.refresh()
					}
				}
			}

		case key.Matches(msg, keys.Timer):
			if m.view == ViewDashboard {
				if item, ok := m.taskList.SelectedItem().(TaskItem); ok {
					if item.Task.IsTimerRunning() {
						_, _ = m.store.StopTimer(item.ID)
					} else {
						_ = m.store.StartTimer(item.ID)
					}
					return m.refresh()
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

func (m Model) refresh() (tea.Model, tea.Cmd) {
	f, err := m.store.Read()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.file = f
	m.initTaskList()
	return m, nil
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

	help1 := HelpStyle.Render("n:new  e:edit  s:start  d:done  t:timer  a:archive  A:archive all  enter:details  /:filter  r:refresh  q:quit")
	help2 := HelpStyle.Render("0-4:status  p:priority  N:notes  L:learnings  D:decisions  ?:help")

	return lipgloss.JoinVertical(lipgloss.Left,
		stats,
		progressLine,
		m.taskList.View(),
		help1,
		help2,
	)
}

func (m Model) viewConfirmOverlay() string {
	background := m.viewDashboard()

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAmber).
		Padding(1, 3).
		Background(ColorBg)

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

	// Center the modal on the screen
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)
	x := max((m.width-modalWidth)/2, 0)
	y := max((m.height-modalHeight)/2, 0)

	return placeOverlay(x, y, modal, background, m.width, m.height)
}

func (m Model) viewHelpOverlay() string {
	background := m.viewDashboard()

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent).
		Padding(1, 3).
		Background(ColorBg)

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

	// Center the modal on the screen
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)
	x := max((m.width-modalWidth)/2, 0)
	y := max((m.height-modalHeight)/2, 0)

	return placeOverlay(x, y, modal, background, m.width, m.height)
}

// placeOverlay places a foreground string over a background string at the given position
func placeOverlay(x, y int, fg, bg string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	// Ensure we have enough lines
	for len(bgLines) < height {
		bgLines = append(bgLines, "")
	}

	// Overlay foreground onto background
	for i, fgLine := range fgLines {
		bgY := y + i
		if bgY < 0 || bgY >= len(bgLines) {
			continue
		}

		bgLine := bgLines[bgY]
		// Pad bgLine to be wide enough
		for len(bgLine) < x+len(fgLine) {
			bgLine += " "
		}

		// Replace portion of bgLine with fgLine
		runes := []rune(bgLine)
		fgRunes := []rune(fgLine)

		// Build new line
		var newLine []rune
		for j := 0; j < len(runes) || j < x+len(fgRunes); j++ {
			if j >= x && j < x+len(fgRunes) {
				newLine = append(newLine, fgRunes[j-x])
			} else if j < len(runes) {
				newLine = append(newLine, runes[j])
			} else {
				newLine = append(newLine, ' ')
			}
		}
		bgLines[bgY] = string(newLine)
	}

	return strings.Join(bgLines, "\n")
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

	// Description
	sectionStyle := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)
	b.WriteString(sectionStyle.Render("Description"))
	b.WriteString("\n")
	b.WriteString(t.Description)
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

	// Blocked by
	if len(t.BlockedBy) > 0 {
		b.WriteString(TaskBlockedStyle.Render("Blocked by: "))
		b.WriteString(strings.Join(t.BlockedBy, ", "))
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
func (m Model) renderModal(title string, borderColor lipgloss.Color, content string, helpText string) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 3)

	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorSlate).
		Padding(0, 1).
		Width(50)

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		"",
		inputStyle.Render(content),
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

func (m Model) viewCreate() string {
	modal := m.renderModal(
		"New Task",
		ColorAccent,
		m.createInput.View(),
		"enter: create  esc: cancel",
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
		"enter: save  esc: cancel",
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
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, n.Text))
			b.WriteString(fmt.Sprintf("     %s\n", HelpStyle.Render(n.CreatedAt.Format("2006-01-02 15:04"))))
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
				b.WriteString(fmt.Sprintf("    %s\n", HelpStyle.Render(l.CreatedAt.Format("2006-01-02 15:04"))))
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
			b.WriteString(fmt.Sprintf("    %s\n", HelpStyle.Render(d.CreatedAt.Format("2006-01-02 15:04"))))
			b.WriteString("\n")
		}
	}

	b.WriteString(HelpStyle.Render("esc: back"))

	panel := PanelStyle.Width(m.width - 4).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
