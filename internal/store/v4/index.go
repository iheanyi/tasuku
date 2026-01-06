package v4

import (
	"encoding/json"
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
	Status    string    `json:"status"`
	Priority  *int      `json:"priority,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	BlockedBy []string  `json:"blocked_by,omitempty"`
	ParentID  string    `json:"parent_id,omitempty"`
	Owner     string    `json:"owner,omitempty"`
	ClaimedBy string    `json:"claimed_by,omitempty"`
	File      string    `json:"file"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewIndex creates a new empty index.
func NewIndex() *Index {
	return &Index{
		Version:   "v4",
		Tasks:     make(map[string]TaskMeta),
		UpdatedAt: time.Now().UTC(),
	}
}

// AddTask adds or updates a task in the index.
func (idx *Index) AddTask(id string, fm TaskFrontmatter) {
	idx.Tasks[id] = TaskMeta{
		Status:    fm.Status,
		Priority:  fm.Priority,
		Tags:      fm.Tags,
		BlockedBy: fm.BlockedBy,
		ParentID:  fm.ParentID,
		Owner:     fm.Owner,
		ClaimedBy: fm.ClaimedBy,
		File:      "tasks/" + id + ".md",
		UpdatedAt: fm.UpdatedAt,
	}
	idx.UpdatedAt = time.Now().UTC()
}

// RemoveTask removes a task from the index.
func (idx *Index) RemoveTask(id string) {
	delete(idx.Tasks, id)
	idx.UpdatedAt = time.Now().UTC()
}

// SetCounts updates the archive, learnings, and decisions counts.
func (idx *Index) SetCounts(archived, learnings, decisions int) {
	idx.ArchivedCount = archived
	idx.LearningsCount = learnings
	idx.DecisionsCount = decisions
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
