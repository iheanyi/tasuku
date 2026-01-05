// Package task defines the core task domain types.
package task

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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

// Duration wraps time.Duration for JSON serialization as a string (e.g., "2h30m5s").
type Duration time.Duration

// MarshalJSON implements json.Marshaler for Duration.
func (d Duration) MarshalJSON() ([]byte, error) {
	if d == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler for Duration.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		*d = 0
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// String returns the duration as a string.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// TimeDuration returns the underlying time.Duration.
func (d Duration) TimeDuration() time.Duration {
	return time.Duration(d)
}

// FormatHumanReadable returns a human-friendly duration string.
func (d Duration) FormatHumanReadable() string {
	td := time.Duration(d)
	if td == 0 {
		return "0s"
	}

	var parts []string
	hours := int(td.Hours())
	minutes := int(td.Minutes()) % 60
	seconds := int(td.Seconds()) % 60

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, "")
}

// IsTimerRunning returns true if the task has an active timer.
func (t Task) IsTimerRunning() bool {
	return t.TimerStart != nil
}

// CurrentDuration returns the total duration including any active timer.
func (t Task) CurrentDuration() time.Duration {
	total := time.Duration(t.Duration)
	if t.TimerStart != nil {
		total += time.Since(*t.TimerStart)
	}
	return total
}

// Task represents a single task.
type Task struct {
	Status      Status     `json:"status"`
	Description string     `json:"description"`
	Priority    *int       `json:"priority,omitempty"`   // 0=critical, 1=high, 2=normal (default), 3=low, 4=backlog
	ParentID    *string    `json:"parent_id,omitempty"`  // V3.0: Parent task ID for subtasks
	BlockedBy   []string   `json:"blocked_by"`
	Owner       *string    `json:"owner"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"` // When the task was claimed by an agent
	Tags        []string          `json:"tags,omitempty"`        // V2.0: Tags for filtering and grouping
	Fields      map[string]string `json:"fields,omitempty"`      // V2.0: Custom key-value metadata
	TimerStart  *time.Time        `json:"timer_start,omitempty"` // V2.0: When timer started, nil if not running
	Duration    Duration   `json:"duration,omitempty"`    // V2.0: Accumulated time spent on task
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// DefaultClaimTimeout is the duration after which a claim is considered stale.
const DefaultClaimTimeout = 2 * time.Hour

// IsClaimStale returns true if the claim is older than the timeout duration.
func (t Task) IsClaimStale(timeout time.Duration) bool {
	if t.ClaimedAt == nil {
		return false
	}
	return time.Since(*t.ClaimedAt) > timeout
}

// HasTag returns true if the task has the specified tag.
func (t Task) HasTag(tag string) bool {
	for _, tg := range t.Tags {
		if tg == tag {
			return true
		}
	}
	return false
}

// IsSubtask returns true if this task has a parent.
func (t Task) IsSubtask() bool {
	return t.ParentID != nil && *t.ParentID != ""
}

// GetParentID returns the parent ID or empty string if not a subtask.
func (t Task) GetParentID() string {
	if t.ParentID == nil {
		return ""
	}
	return *t.ParentID
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
	ID        string    `json:"id"`
	Chose     string    `json:"chose"`
	Over      []string  `json:"over"`
	Because   string    `json:"because"`
	CreatedAt time.Time `json:"created_at"`
}

// Note represents a note attached to a task with timestamp.
type Note struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Learning represents a recorded learning with stable ID.
type Learning struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	IsRule    bool      `json:"is_rule,omitempty"` // V2.0: True if learning is a never/always rule
	CreatedAt time.Time `json:"created_at"`
}

// IsRuleLearning detects if a learning text matches never/always patterns.
// It returns true if the text:
// - Starts with "Never" or "Always" (case-insensitive)
// - Contains "never" or "always" as key instruction words
func IsRuleLearning(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))

	// Check if starts with never/always
	if strings.HasPrefix(lower, "never ") || strings.HasPrefix(lower, "always ") {
		return true
	}

	// Check for never/always as key words in the text
	// Look for patterns like "you should never", "must always", etc.
	words := strings.Fields(lower)
	for _, word := range words {
		// Clean punctuation from word
		word = strings.Trim(word, ".,;:!?\"'")
		if word == "never" || word == "always" {
			return true
		}
	}

	return false
}

// GenerateShortID creates a 6-character hex ID for notes and learnings.
func GenerateShortID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateTaskID creates a task ID from a description, adding a suffix only if needed.
// Pass existingIDs to check for collisions - if nil, always generates clean ID.
// Format: kebab-case-description (clean) or kebab-case-description-xyz (with suffix on collision)
func GenerateTaskID(desc string, existingIDs map[string]struct{}) string {
	baseID := generateBaseID(desc)

	// If no existing IDs provided or no collision, return clean ID
	if existingIDs == nil {
		return baseID
	}

	if _, exists := existingIDs[baseID]; !exists {
		return baseID
	}

	// Collision detected - add random suffix
	for i := 0; i < 100; i++ { // Try up to 100 times
		suffix := GenerateShortID()[:3]
		newID := baseID + "-" + suffix
		if _, exists := existingIDs[newID]; !exists {
			return newID
		}
	}

	// Fallback to longer suffix if still colliding (very unlikely)
	return baseID + "-" + GenerateShortID()
}

// generateBaseID creates a deterministic kebab-case ID from description.
func generateBaseID(desc string) string {
	result := ""
	for _, r := range desc {
		if r >= 'a' && r <= 'z' {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32) // lowercase
		} else if r == ' ' && len(result) > 0 && result[len(result)-1] != '-' {
			result += "-"
		}
	}
	result = strings.TrimSuffix(result, "-")

	// Handle empty description
	if result == "" {
		return "task-" + GenerateShortID()[:3]
	}

	// Truncate to 32 chars for reasonable length
	if len(result) > 32 {
		result = result[:32]
		result = strings.TrimSuffix(result, "-")
	}

	return result
}

// Context holds agent learnings and decisions.
type Context struct {
	Learnings []Learning        `json:"learnings"`
	Decisions []Decision        `json:"decisions"`
	Notes     map[string][]Note `json:"notes"`
}

// UnmarshalJSON implements custom unmarshaling for Context to handle
// backward compatibility with old learnings format ([]string vs []Learning)
// and old notes format ([]string vs []Note).
func (c *Context) UnmarshalJSON(data []byte) error {
	// Use a raw structure to detect the format of each field
	type rawContext struct {
		Learnings json.RawMessage     `json:"learnings"`
		Decisions []Decision          `json:"decisions"`
		Notes     json.RawMessage     `json:"notes"`
	}
	var raw rawContext
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Decisions = raw.Decisions

	// Handle learnings - try new format first, then old string format
	if len(raw.Learnings) > 0 {
		var newLearnings []Learning
		if err := json.Unmarshal(raw.Learnings, &newLearnings); err == nil {
			c.Learnings = newLearnings
			// Ensure all learnings have IDs (backward compatibility)
			for i := range c.Learnings {
				if c.Learnings[i].ID == "" {
					c.Learnings[i].ID = GenerateShortID()
				}
			}
		} else {
			// Try old format: []string
			var oldLearnings []string
			if err := json.Unmarshal(raw.Learnings, &oldLearnings); err != nil {
				return err
			}
			c.Learnings = make([]Learning, len(oldLearnings))
			for i, text := range oldLearnings {
				c.Learnings[i] = Learning{
					ID:        GenerateShortID(),
					Text:      text,
					CreatedAt: time.Time{}, // Zero time for migrated learnings
				}
			}
		}
	} else {
		c.Learnings = []Learning{}
	}

	// Handle notes - try new format first, then old string format
	if len(raw.Notes) > 0 {
		var newNotes map[string][]Note
		if err := json.Unmarshal(raw.Notes, &newNotes); err == nil {
			c.Notes = newNotes
			// Ensure all notes have IDs (backward compatibility)
			for taskID, notes := range c.Notes {
				for i := range notes {
					if notes[i].ID == "" {
						notes[i].ID = GenerateShortID()
					}
				}
				c.Notes[taskID] = notes
			}
		} else {
			// Try old format: map[string][]string
			var oldNotes map[string][]string
			if err := json.Unmarshal(raw.Notes, &oldNotes); err != nil {
				return err
			}
			c.Notes = make(map[string][]Note)
			for taskID, notes := range oldNotes {
				for _, text := range notes {
					c.Notes[taskID] = append(c.Notes[taskID], Note{
						ID:        GenerateShortID(),
						Text:      text,
						CreatedAt: time.Time{}, // Zero time for migrated notes
					})
				}
			}
		}
	} else {
		c.Notes = make(map[string][]Note)
	}

	return nil
}

// ArchivedTask represents a completed task that has been archived.
// It preserves the original task data plus archival metadata.
type ArchivedTask struct {
	Task
	ArchivedAt time.Time `json:"archived_at"`           // When the task was archived
	Summary    string    `json:"summary,omitempty"`     // Optional AI-generated summary
	TotalTime  Duration  `json:"total_time,omitempty"`  // Final time spent on task
}

// File represents the complete .tasuku.json structure.
type File struct {
	Version int                     `json:"version"`
	Tasks   map[string]Task         `json:"tasks"`
	Context Context                 `json:"context"`
	Archive map[string]ArchivedTask `json:"archive,omitempty"` // Archived completed tasks
}

// NewFile creates an empty task file with defaults.
func NewFile() *File {
	return &File{
		Version: 1,
		Tasks:   make(map[string]Task),
		Context: Context{
			Learnings: []Learning{},
			Decisions: []Decision{},
			Notes:     make(map[string][]Note),
		},
		Archive: make(map[string]ArchivedTask),
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
