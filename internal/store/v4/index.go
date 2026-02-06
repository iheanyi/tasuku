package v4

import (
	"encoding/json"
	"strings"
	"time"
)

// Index represents the auto-generated index.json file for fast agent queries.
type Index struct {
	Version        string              `json:"version"`
	Tasks          map[string]TaskMeta `json:"tasks"`
	ArchivedCount  int                 `json:"archived_count"`
	LearningsCount int                 `json:"learnings_count"`
	DecisionsCount int                 `json:"decisions_count"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// TaskMeta contains the metadata for a task in the index.
// Only frontmatter fields are included (not full content).
type TaskMeta struct {
	Status      string     `json:"status"`
	Priority    *int       `json:"priority,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	BlockedBy   []string   `json:"blocked_by,omitempty"`
	ParentID    string     `json:"parent_id,omitempty"`
	Owner       string     `json:"owner,omitempty"`
	ClaimedBy   string     `json:"claimed_by,omitempty"`
	Description string     `json:"description,omitempty"`
	File        string     `json:"file"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	TimerStart  *time.Time `json:"timer_start,omitempty"`
}

// NewIndex creates a new empty index.
func NewIndex() *Index {
	return &Index{
		Version:   "v4",
		Tasks:     make(map[string]TaskMeta),
		UpdatedAt: time.Now().UTC(),
	}
}

// AddTask adds or updates a task in the index from frontmatter only.
// Description is not available from frontmatter alone; use AddTaskWithDescription
// when the full task content has been parsed.
func (idx *Index) AddTask(id string, fm TaskFrontmatter) {
	// Preserve existing description if we're updating and it's already set
	existing, hasExisting := idx.Tasks[id]
	desc := ""
	if hasExisting {
		desc = existing.Description
	}
	idx.Tasks[id] = TaskMeta{
		Status:      fm.Status,
		Priority:    fm.Priority,
		Tags:        fm.Tags,
		BlockedBy:   fm.BlockedBy,
		ParentID:    fm.ParentID,
		Owner:       fm.Owner,
		ClaimedBy:   fm.ClaimedBy,
		Description: desc,
		File:        "tasks/" + id + ".md",
		CreatedAt:   fm.CreatedAt,
		UpdatedAt:   fm.UpdatedAt,
		TimerStart:  fm.TimerStart,
	}
	idx.UpdatedAt = time.Now().UTC()
}

// AddTaskWithDescription adds or updates a task in the index with an explicit description.
// Use this when the full task content (including body/description) is available.
func (idx *Index) AddTaskWithDescription(id, description string, fm TaskFrontmatter) {
	idx.Tasks[id] = TaskMeta{
		Status:      fm.Status,
		Priority:    fm.Priority,
		Tags:        fm.Tags,
		BlockedBy:   fm.BlockedBy,
		ParentID:    fm.ParentID,
		Owner:       fm.Owner,
		ClaimedBy:   fm.ClaimedBy,
		Description: truncateDescription(description, 200),
		File:        "tasks/" + id + ".md",
		CreatedAt:   fm.CreatedAt,
		UpdatedAt:   fm.UpdatedAt,
		TimerStart:  fm.TimerStart,
	}
	idx.UpdatedAt = time.Now().UTC()
}

// truncateDescription truncates a description to maxLen characters at a word boundary.
func truncateDescription(desc string, maxLen int) string {
	if len(desc) <= maxLen {
		return desc
	}
	// Find last space before maxLen
	truncated := desc[:maxLen]
	if idx := strings.LastIndex(truncated, " "); idx > maxLen/2 {
		truncated = truncated[:idx]
	}
	return truncated + "..."
}

// RemoveTask removes a task from the index.
func (idx *Index) RemoveTask(id string) {
	delete(idx.Tasks, id)
	idx.UpdatedAt = time.Now().UTC()
}

// Marshal returns the JSON representation of the index.
func (idx *Index) Marshal() ([]byte, error) {
	return json.MarshalIndent(idx, "", "  ")
}

// ParseIndex parses an index.json file.
func ParseIndex(data []byte) (*Index, error) {
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	if idx.Tasks == nil {
		idx.Tasks = make(map[string]TaskMeta)
	}
	return &idx, nil
}
