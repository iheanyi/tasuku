// Package v4 implements the V4 Markdown-based storage format.
// It provides parsers and writers for task files with YAML frontmatter.
package v4

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/iheanyi/tasuku/internal/task"
	"gopkg.in/yaml.v3"
)

// TaskFrontmatter represents the YAML frontmatter of a task file.
type TaskFrontmatter struct {
	Status     string            `yaml:"status"`
	Priority   *int              `yaml:"priority,omitempty"`
	Tags       []string          `yaml:"tags,omitempty,flow"`
	BlockedBy  []string          `yaml:"blocked_by,omitempty,flow"`
	ParentID   string            `yaml:"parent_id,omitempty"`
	Owner      string            `yaml:"owner,omitempty"`
	ClaimedBy  string            `yaml:"claimed_by,omitempty"`
	TimeSpent  int64             `yaml:"time_spent,omitempty"` // nanoseconds (matches task.Duration)
	Fields     map[string]string `yaml:"fields,omitempty"`
	CreatedAt  time.Time         `yaml:"created_at"`
	UpdatedAt  time.Time         `yaml:"updated_at"`
	ClaimedAt  *time.Time        `yaml:"claimed_at,omitempty"`
	TimerStart *time.Time        `yaml:"timer_start,omitempty"`
}

// ParsedTask represents a fully parsed task from a Markdown file.
type ParsedTask struct {
	ID          string
	Frontmatter TaskFrontmatter
	Title       string
	Description string
	Notes       []ParsedNote
	RawBody     string // Full markdown body after frontmatter
}

// ParsedNote represents a note parsed from the Notes section.
type ParsedNote struct {
	ID        string    // Generated from content hash or position
	Text      string
	Timestamp time.Time
}

var (
	// frontmatterDelim matches the YAML frontmatter delimiters
	frontmatterDelim = []byte("---")
	// noteHeaderRegex matches "### YYYY-MM-DD HH:MM [id]" format
	noteHeaderRegex = regexp.MustCompile(`^###\s+(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2})(?:\s+\[([a-z0-9]+)\])?$`)
)

// ParseTaskFile parses a Markdown task file with YAML frontmatter.
// Returns the parsed task or an error with line number information.
func ParseTaskFile(id string, content []byte) (*ParsedTask, error) {
	// Split frontmatter and body
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Parse YAML frontmatter
	var fm TaskFrontmatter
	if err := yaml.Unmarshal(frontmatter, &fm); err != nil {
		return nil, fmt.Errorf("invalid frontmatter: %w", err)
	}

	// Validate required fields
	if fm.Status == "" {
		return nil, fmt.Errorf("missing required field: status")
	}

	// Parse body content
	title, description, notes := parseBody(body)

	return &ParsedTask{
		ID:          id,
		Frontmatter: fm,
		Title:       title,
		Description: description,
		Notes:       notes,
		RawBody:     string(body),
	}, nil
}

// splitFrontmatter splits a Markdown file into frontmatter and body.
func splitFrontmatter(content []byte) ([]byte, []byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// First line must be ---
	if !scanner.Scan() || !bytes.Equal(bytes.TrimSpace(scanner.Bytes()), frontmatterDelim) {
		return nil, nil, fmt.Errorf("file must start with '---' frontmatter delimiter")
	}

	var frontmatterLines []string
	foundEnd := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundEnd = true
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	if !foundEnd {
		return nil, nil, fmt.Errorf("frontmatter not closed with '---'")
	}

	frontmatter := []byte(strings.Join(frontmatterLines, "\n"))

	// Remaining content is the body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	// Trim leading empty lines from body
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
		bodyLines = bodyLines[1:]
	}

	body := []byte(strings.Join(bodyLines, "\n"))

	return frontmatter, body, nil
}

// parseBody parses the Markdown body to extract title, description, and notes.
func parseBody(body []byte) (title, description string, notes []ParsedNote) {
	lines := strings.Split(string(body), "\n")
	if len(lines) == 0 {
		return "", "", nil
	}

	var descLines []string
	var notesSection bool
	var currentNote *ParsedNote

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for title (H1)
		if strings.HasPrefix(line, "# ") && title == "" {
			title = strings.TrimPrefix(line, "# ")
			continue
		}

		// Check for Notes section start
		if strings.TrimSpace(line) == "## Notes" {
			notesSection = true
			continue
		}

		// Parse notes in the Notes section
		if notesSection {
			if match := noteHeaderRegex.FindStringSubmatch(line); match != nil {
				// Save previous note if exists
				if currentNote != nil {
					currentNote.Text = strings.TrimSpace(currentNote.Text)
					notes = append(notes, *currentNote)
				}

				// Parse timestamp
				dateStr := match[1]
				timeStr := match[2]
				ts, _ := time.Parse("2006-01-02 15:04", dateStr+" "+timeStr)

				// Use ID from markdown if present, otherwise generate one
				noteID := match[3]
				if noteID == "" {
					noteID = task.GenerateShortID()
				}

				currentNote = &ParsedNote{
					ID:        noteID,
					Timestamp: ts,
				}
				continue
			}

			// Accumulate note content
			if currentNote != nil {
				if currentNote.Text != "" {
					currentNote.Text += "\n"
				}
				currentNote.Text += line
			}
			continue
		}

		// Skip other H2 sections for description
		if strings.HasPrefix(line, "## ") && !notesSection {
			// Don't include other sections in description
			continue
		}

		// Accumulate description (before Notes section)
		if !notesSection && !strings.HasPrefix(line, "## ") {
			descLines = append(descLines, line)
		}
	}

	// Save final note
	if currentNote != nil {
		currentNote.Text = strings.TrimSpace(currentNote.Text)
		notes = append(notes, *currentNote)
	}

	description = strings.TrimSpace(strings.Join(descLines, "\n"))

	return title, description, notes
}

// ToTask converts a ParsedTask to a task.Task.
func (p *ParsedTask) ToTask() task.Task {
	t := task.Task{
		Status:      task.Status(p.Frontmatter.Status),
		Description: p.Description,
		Priority:    p.Frontmatter.Priority,
		BlockedBy:   p.Frontmatter.BlockedBy,
		Tags:        p.Frontmatter.Tags,
		Fields:      p.Frontmatter.Fields,
		CreatedAt:   p.Frontmatter.CreatedAt,
		UpdatedAt:   p.Frontmatter.UpdatedAt,
		ClaimedAt:   p.Frontmatter.ClaimedAt,
		TimerStart:  p.Frontmatter.TimerStart,
	}

	// Handle optional pointer fields
	if p.Frontmatter.ParentID != "" {
		t.ParentID = &p.Frontmatter.ParentID
	}
	if p.Frontmatter.Owner != "" {
		t.Owner = &p.Frontmatter.Owner
	}

	// Convert time_spent to Duration (stored as nanoseconds)
	if p.Frontmatter.TimeSpent > 0 {
		t.Duration = task.Duration(p.Frontmatter.TimeSpent)
	}

	// Ensure non-nil slices
	if t.BlockedBy == nil {
		t.BlockedBy = []string{}
	}

	// Use title if description is empty
	if t.Description == "" && p.Title != "" {
		t.Description = p.Title
	}

	return t
}

// ToNotes converts parsed notes to task.Note slice.
func (p *ParsedTask) ToNotes() []task.Note {
	notes := make([]task.Note, len(p.Notes))
	for i, n := range p.Notes {
		notes[i] = task.Note{
			ID:        n.ID,
			Text:      n.Text,
			CreatedAt: n.Timestamp,
		}
	}
	return notes
}

// WriteTaskFile generates Markdown content for a task.
func WriteTaskFile(id string, t task.Task, notes []task.Note) ([]byte, error) {
	var buf bytes.Buffer

	// Build frontmatter
	fm := TaskFrontmatter{
		Status:     string(t.Status),
		Priority:   t.Priority,
		Tags:       t.Tags,
		BlockedBy:  t.BlockedBy,
		Fields:     t.Fields,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
		ClaimedAt:  t.ClaimedAt,
		TimerStart: t.TimerStart,
	}

	if t.ParentID != nil {
		fm.ParentID = *t.ParentID
	}
	if t.Owner != nil {
		fm.Owner = *t.Owner
	}
	if t.Duration > 0 {
		fm.TimeSpent = int64(t.Duration) // Store as nanoseconds
	}

	// Write frontmatter
	buf.WriteString("---\n")

	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
	}
	buf.Write(fmBytes)

	buf.WriteString("---\n\n")

	// Write title (H1)
	title := extractTitle(t.Description)
	buf.WriteString("# ")
	buf.WriteString(title)
	buf.WriteString("\n\n")

	// Write description (remaining content after title)
	desc := extractDescription(t.Description, title)
	if desc != "" {
		buf.WriteString(desc)
		buf.WriteString("\n")
	}

	// Write notes section if any
	if len(notes) > 0 {
		buf.WriteString("\n## Notes\n\n")

		// Notes in reverse chronological order (newest first)
		for i := len(notes) - 1; i >= 0; i-- {
			n := notes[i]
			ts := n.CreatedAt
			if ts.IsZero() {
				ts = time.Now()
			}
			// Include note ID in header for persistence: ### YYYY-MM-DD HH:MM [id]
			buf.WriteString(fmt.Sprintf("### %s [%s]\n", ts.Format("2006-01-02 15:04"), n.ID))
			buf.WriteString(n.Text)
			buf.WriteString("\n\n")
		}
	}

	return buf.Bytes(), nil
}

// extractTitle extracts a title from a description.
// If the description starts with a single line that looks like a title, use it.
// Otherwise, truncate the description to 80 chars.
func extractTitle(desc string) string {
	lines := strings.SplitN(desc, "\n", 2)
	firstLine := strings.TrimSpace(lines[0])

	// If first line is short enough and looks like a title, use it
	if len(firstLine) <= 80 {
		return firstLine
	}

	// Truncate to 80 chars at word boundary
	if len(firstLine) > 80 {
		truncated := firstLine[:80]
		if idx := strings.LastIndex(truncated, " "); idx > 40 {
			truncated = truncated[:idx]
		}
		return truncated + "..."
	}

	return firstLine
}

// extractDescription extracts the description body after the title.
func extractDescription(desc, title string) string {
	// If the description equals the title, no additional description
	if strings.TrimSpace(desc) == strings.TrimSpace(title) {
		return ""
	}

	// If description starts with title, return the rest
	if strings.HasPrefix(desc, title) {
		rest := strings.TrimPrefix(desc, title)
		return strings.TrimSpace(rest)
	}

	return desc
}
