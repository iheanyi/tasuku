package v4

import (
	"testing"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
)

func TestParseTaskFile(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		content string
		want    *ParsedTask
		wantErr bool
	}{
		{
			name: "basic task",
			id:   "test-task",
			content: `---
status: ready
created_at: 2024-01-05T10:00:00Z
updated_at: 2024-01-05T11:00:00Z
---

# Implement JWT authentication

Add token-based authentication to protect API endpoints.
`,
			want: &ParsedTask{
				ID: "test-task",
				Frontmatter: TaskFrontmatter{
					Status:    "ready",
					CreatedAt: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC),
				},
				Title:       "Implement JWT authentication",
				Description: "Add token-based authentication to protect API endpoints.",
			},
		},
		{
			name: "task with all fields",
			id:   "auth-jwt",
			content: `---
status: in_progress
priority: 2
tags: [backend, api]
blocked_by: [auth-setup]
parent_id: epic-123
owner: claude
claimed_by: agent-1
time_spent: 3600
fields:
  estimate: 2h
  pr: https://github.com/org/repo/pull/123
created_at: 2024-01-05T10:00:00Z
updated_at: 2024-01-05T11:00:00Z
---

# Implement JWT authentication

Add token-based authentication to protect API endpoints.

Support **rich formatting**, ` + "`inline code`" + `, and code blocks:

` + "```go" + `
func ValidateToken(token string) (*Claims, error) {
    // Implementation
}
` + "```" + `
`,
			want: &ParsedTask{
				ID: "auth-jwt",
				Frontmatter: TaskFrontmatter{
					Status:    "in_progress",
					Priority:  intPtr(2),
					Tags:      []string{"backend", "api"},
					BlockedBy: []string{"auth-setup"},
					ParentID:  "epic-123",
					Owner:     "claude",
					ClaimedBy: "agent-1",
					TimeSpent: 3600,
					Fields: map[string]string{
						"estimate": "2h",
						"pr":       "https://github.com/org/repo/pull/123",
					},
					CreatedAt: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC),
				},
				Title: "Implement JWT authentication",
			},
		},
		{
			name: "task with notes",
			id:   "bug-fix",
			content: `---
status: in_progress
created_at: 2024-01-05T10:00:00Z
updated_at: 2024-01-05T11:00:00Z
---

# Fix authentication bug

## Notes

### 2024-01-05 11:00
Found root cause - race condition in auth middleware. Need to add mutex.

### 2024-01-05 10:30
Started investigating the bug. Reproduced locally with concurrent requests.
`,
			want: &ParsedTask{
				ID: "bug-fix",
				Frontmatter: TaskFrontmatter{
					Status:    "in_progress",
					CreatedAt: time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC),
				},
				Title:       "Fix authentication bug",
				Description: "",
			},
		},
		{
			name:    "missing frontmatter delimiter",
			id:      "bad-task",
			content: "# Just a title\n\nNo frontmatter here",
			wantErr: true,
		},
		{
			name: "missing status",
			id:   "bad-task",
			content: `---
priority: 2
created_at: 2024-01-05T10:00:00Z
updated_at: 2024-01-05T11:00:00Z
---

# Missing status
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskFile(tt.id, []byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Frontmatter.Status != tt.want.Frontmatter.Status {
				t.Errorf("Status = %v, want %v", got.Frontmatter.Status, tt.want.Frontmatter.Status)
			}
			if got.Title != tt.want.Title {
				t.Errorf("Title = %v, want %v", got.Title, tt.want.Title)
			}
		})
	}
}

func TestWriteTaskFile(t *testing.T) {
	now := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
	priority := 2

	tests := []struct {
		name    string
		id      string
		task    task.Task
		notes   []task.Note
		wantErr bool
	}{
		{
			name: "basic task",
			id:   "test-task",
			task: task.Task{
				Status:      task.StatusReady,
				Description: "Implement JWT authentication",
				CreatedAt:   now,
				UpdatedAt:   now,
				BlockedBy:   []string{},
			},
			notes: nil,
		},
		{
			name: "task with all fields",
			id:   "auth-jwt",
			task: task.Task{
				Status:      task.StatusInProgress,
				Description: "Implement JWT authentication\n\nAdd token-based auth to protect endpoints.",
				Priority:    &priority,
				Tags:        []string{"backend", "api"},
				BlockedBy:   []string{"auth-setup"},
				Fields: map[string]string{
					"estimate": "2h",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			notes: nil,
		},
		{
			name: "task with notes",
			id:   "bug-fix",
			task: task.Task{
				Status:      task.StatusInProgress,
				Description: "Fix authentication bug",
				CreatedAt:   now,
				UpdatedAt:   now,
				BlockedBy:   []string{},
			},
			notes: []task.Note{
				{ID: "abc123", Text: "Started investigating", CreatedAt: now},
				{ID: "def456", Text: "Found the root cause", CreatedAt: now.Add(30 * time.Minute)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := WriteTaskFile(tt.id, tt.task, tt.notes)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteTaskFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Parse it back
			parsed, err := ParseTaskFile(tt.id, content)
			if err != nil {
				t.Errorf("Failed to parse written content: %v\nContent:\n%s", err, string(content))
				return
			}

			if parsed.Frontmatter.Status != string(tt.task.Status) {
				t.Errorf("Roundtrip Status = %v, want %v", parsed.Frontmatter.Status, tt.task.Status)
			}
		})
	}
}

func TestParsedTaskToTask(t *testing.T) {
	priority := 2
	now := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)

	parsed := &ParsedTask{
		ID: "test-task",
		Frontmatter: TaskFrontmatter{
			Status:    "in_progress",
			Priority:  &priority,
			Tags:      []string{"backend"},
			BlockedBy: []string{"dep-1"},
			ParentID:  "parent-task",
			Owner:     "claude",
			TimeSpent: int64(time.Hour), // nanoseconds
			CreatedAt: now,
			UpdatedAt: now,
		},
		Title:       "Test Task",
		Description: "Test description",
	}

	got := parsed.ToTask()

	if got.Status != task.StatusInProgress {
		t.Errorf("Status = %v, want %v", got.Status, task.StatusInProgress)
	}
	if got.Description != "Test description" {
		t.Errorf("Description = %v, want %v", got.Description, "Test description")
	}
	if got.Priority == nil || *got.Priority != 2 {
		t.Errorf("Priority = %v, want %v", got.Priority, &priority)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "backend" {
		t.Errorf("Tags = %v, want [backend]", got.Tags)
	}
	if got.ParentID == nil || *got.ParentID != "parent-task" {
		t.Errorf("ParentID = %v, want parent-task", got.ParentID)
	}
	if got.Owner == nil || *got.Owner != "claude" {
		t.Errorf("Owner = %v, want claude", got.Owner)
	}
	if time.Duration(got.Duration) != time.Hour {
		t.Errorf("Duration = %v, want 1h", got.Duration)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantFM  string
		wantErr bool
	}{
		{
			name: "valid frontmatter",
			content: `---
status: ready
---

# Title`,
			wantFM: "status: ready",
		},
		{
			name:    "no opening delimiter",
			content: "status: ready\n---\n# Title",
			wantErr: true,
		},
		{
			name:    "no closing delimiter",
			content: "---\nstatus: ready\n# Title",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, err := splitFrontmatter([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("splitFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if string(fm) != tt.wantFM {
				t.Errorf("frontmatter = %q, want %q", string(fm), tt.wantFM)
			}
		})
	}
}

func TestNotesParsing(t *testing.T) {
	content := `---
status: ready
created_at: 2024-01-05T10:00:00Z
updated_at: 2024-01-05T11:00:00Z
---

# Task with notes

## Notes

### 2024-01-05 11:00
Second note with multiple lines.
Line 2 of note.

### 2024-01-05 10:30
First note.
`

	parsed, err := ParseTaskFile("test", []byte(content))
	if err != nil {
		t.Fatalf("ParseTaskFile() error = %v", err)
	}

	if len(parsed.Notes) != 2 {
		t.Fatalf("Notes count = %d, want 2", len(parsed.Notes))
	}

	// Notes should be in document order (11:00 first, 10:30 second)
	if !parsed.Notes[0].Timestamp.Equal(time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC)) {
		t.Errorf("First note timestamp = %v, want 2024-01-05 11:00", parsed.Notes[0].Timestamp)
	}
	if !parsed.Notes[1].Timestamp.Equal(time.Date(2024, 1, 5, 10, 30, 0, 0, time.UTC)) {
		t.Errorf("Second note timestamp = %v, want 2024-01-05 10:30", parsed.Notes[1].Timestamp)
	}
}

func intPtr(i int) *int {
	return &i
}
