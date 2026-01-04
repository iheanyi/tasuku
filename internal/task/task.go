// Package task defines the core task domain types.
package task

import "time"

// Status represents the state of a task.
type Status string

const (
	StatusReady      Status = "ready"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusDone       Status = "done"
)

// Priority levels for tasks.
const (
	PriorityCritical = 0
	PriorityHigh     = 1
	PriorityNormal   = 2
	PriorityLow      = 3
	PriorityBacklog  = 4
)

// Task represents a single task.
type Task struct {
	Status      Status    `json:"status"`
	Description string    `json:"description"`
	Priority    *int      `json:"priority,omitempty"` // 0=critical, 1=high, 2=normal (default), 3=low, 4=backlog
	BlockedBy   []string  `json:"blocked_by"`
	Owner       *string   `json:"owner"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetPriority returns the task's priority, defaulting to Normal (2) if not set.
func (t Task) GetPriority() int {
	if t.Priority == nil {
		return PriorityNormal
	}
	return *t.Priority
}

// PriorityName returns a human-readable name for the priority level.
func PriorityName(p int) string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	case PriorityBacklog:
		return "backlog"
	default:
		return "unknown"
	}
}

// Decision represents an architectural decision.
type Decision struct {
	ID      string   `json:"id"`
	Chose   string   `json:"chose"`
	Over    []string `json:"over"`
	Because string   `json:"because"`
}

// Context holds agent learnings and decisions.
type Context struct {
	Learnings []string            `json:"learnings"`
	Decisions []Decision          `json:"decisions"`
	Notes     map[string][]string `json:"notes"`
}

// File represents the complete .tasuku.json structure.
type File struct {
	Version int             `json:"version"`
	Tasks   map[string]Task `json:"tasks"`
	Context Context         `json:"context"`
}

// NewFile creates an empty task file with defaults.
func NewFile() *File {
	return &File{
		Version: 1,
		Tasks:   make(map[string]Task),
		Context: Context{
			Learnings: []string{},
			Decisions: []Decision{},
			Notes:     make(map[string][]string),
		},
	}
}

// NewTask creates a task with sensible defaults.
func NewTask(description string) Task {
	now := time.Now().UTC()
	return Task{
		Status:      StatusReady,
		Description: description,
		BlockedBy:   []string{},
		Owner:       nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ValidTransition checks if a status transition is allowed.
func ValidTransition(from, to Status) bool {
	switch from {
	case StatusReady:
		// Allow ready -> done as shortcut for small tasks
		return to == StatusInProgress || to == StatusBlocked || to == StatusDone
	case StatusInProgress:
		return to == StatusDone || to == StatusBlocked || to == StatusReady
	case StatusBlocked:
		return to == StatusReady
	case StatusDone:
		return to == StatusReady // Allow reopening
	default:
		return false
	}
}
