package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

// validateDBPath checks that the path doesn't contain query parameters
// that could override the read-only mode we append.
func validateDBPath(path string) error {
	if strings.ContainsAny(path, "?&#") {
		return fmt.Errorf("database path %q contains query parameters — refusing to open (possible mode override)", path)
	}
	return nil
}

// OverseerTask represents a task from Overseer's SQLite database.
type OverseerTask struct {
	ID          string
	ParentID    *string
	Description string
	Context     *string // Additional context/notes
	Result      *string // Outcome/result
	Priority    int     // 0=high, 1=medium, 2=low
	Completed   bool
	Cancelled   bool
	Archived    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	// VCS fields
	CommitSHA   *string
	Bookmark    *string
	StartCommit *string
}

// OverseerLearning represents a learning from Overseer's SQLite database.
type OverseerLearning struct {
	ID        string
	Content   string
	CreatedAt time.Time
}

// OverseerBlocker represents a blocker relationship from Overseer's SQLite database.
type OverseerBlocker struct {
	TaskID    string
	BlockerID string
}

// OverseerResult holds all data read from an Overseer SQLite database.
type OverseerResult struct {
	Tasks    []OverseerTask
	Learnings []OverseerLearning
	Blockers  []OverseerBlocker
	Metadata  map[string]string // task_id → JSON data
}

// maxTestedSchemaVersion is the highest Overseer schema version we've tested against.
const maxTestedSchemaVersion = 5

// findOverseerDB locates the Overseer database file.
// It checks: custom path → OVERSEER_DB_PATH env → .overseer/tasks.db in CWD.
func findOverseerDB(customPath string) (string, error) {
	if customPath != "" {
		if err := validateDBPath(customPath); err != nil {
			return "", err
		}
		if _, err := os.Stat(customPath); err != nil {
			return "", fmt.Errorf("overseer database not found at %s", customPath)
		}
		return customPath, nil
	}

	if envPath := os.Getenv("OVERSEER_DB_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return "", fmt.Errorf("overseer database not found at %s (from OVERSEER_DB_PATH)", envPath)
		}
		return envPath, nil
	}

	defaultPath := ".overseer/tasks.db"
	if _, err := os.Stat(defaultPath); err != nil {
		return "", fmt.Errorf(".overseer/tasks.db not found - use --path to specify location")
	}
	return defaultPath, nil
}

// parseTimestamp tries RFC3339 first, then SQLite datetime format.
func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp %q", s)
}

// scanNullableTime scans a nullable timestamp column.
func scanNullableTime(val *sql.NullString) *time.Time {
	if val == nil || !val.Valid || val.String == "" {
		return nil
	}
	t, err := parseTimestamp(val.String)
	if err != nil {
		return nil
	}
	return &t
}

// scanNullableString converts sql.NullString to *string.
func scanNullableString(val sql.NullString) *string {
	if !val.Valid {
		return nil
	}
	s := val.String
	return &s
}

// getTableColumns returns a set of column names for the given table using PRAGMA table_info.
func getTableColumns(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// checkSchemaVersion logs a warning if the database schema version exceeds our tested max.
func checkSchemaVersion(db *sql.DB) {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return
	}
	if version > maxTestedSchemaVersion {
		fmt.Fprintf(os.Stderr, "Warning: Overseer schema version %d is newer than tested version %d — some data may not be imported\n", version, maxTestedSchemaVersion)
	}
}

// tableExists checks whether a table exists in the database.
func tableExists(db *sql.DB, name string) bool {
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&tableName)
	return err == nil
}

// readOverseerDB opens the Overseer SQLite database in read-only mode
// and returns all tasks, learnings, blockers, and metadata.
func readOverseerDB(dbPath string) (*OverseerResult, error) {
	if err := validateDBPath(dbPath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database (is it a valid SQLite file?): %w", err)
	}

	checkSchemaVersion(db)

	tasks, err := readOverseerTasks(db)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks: %w", err)
	}

	learnings, err := readOverseerLearnings(db)
	if err != nil {
		return nil, fmt.Errorf("failed to read learnings: %w", err)
	}

	blockers, err := readOverseerBlockers(db)
	if err != nil {
		return nil, fmt.Errorf("failed to read blockers: %w", err)
	}

	metadata, err := readOverseerMetadata(db)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	return &OverseerResult{
		Tasks:     tasks,
		Learnings: learnings,
		Blockers:  blockers,
		Metadata:  metadata,
	}, nil
}

// knownTaskColumns lists all task columns we know how to handle, in SELECT order.
var knownTaskColumns = []string{
	"id", "parent_id", "description", "context", "result", "priority",
	"completed", "cancelled", "archived", "created_at", "updated_at",
	"started_at", "completed_at", "commit_sha", "bookmark", "start_commit",
	"cancelled_at", "archived_at",
}

func readOverseerTasks(db *sql.DB) ([]OverseerTask, error) {
	cols, err := getTableColumns(db, "tasks")
	if err != nil {
		return nil, fmt.Errorf("failed to discover tasks columns: %w", err)
	}

	// Hard requirements — can't produce a valid task without these
	for _, req := range []string{"id", "description"} {
		if !cols[req] {
			return nil, fmt.Errorf("tasks table missing required column %q", req)
		}
	}

	// Build SELECT from known columns that exist, warn about missing ones
	var selectCols, missingCols []string
	for _, name := range knownTaskColumns {
		if cols[name] {
			selectCols = append(selectCols, name)
		} else {
			missingCols = append(missingCols, name)
		}
	}
	if len(missingCols) > 0 {
		fmt.Fprintf(os.Stderr, "Note: tasks table missing columns %s — using defaults\n", strings.Join(missingCols, ", "))
	}

	query := fmt.Sprintf("SELECT %s FROM tasks", strings.Join(selectCols, ", "))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()

	var tasks []OverseerTask
	for rows.Next() {
		var t OverseerTask
		var parentID, context, result sql.NullString
		var createdAt, updatedAt sql.NullString
		var startedAt, completedAt sql.NullString
		var commitSHA, bookmark, startCommit sql.NullString
		var discarded sql.NullString // for columns we read but don't use (cancelled_at, archived_at)

		// Build scan destinations in the same order as selectCols
		scanDests := make([]any, 0, len(selectCols))
		for _, col := range selectCols {
			switch col {
			case "id":
				scanDests = append(scanDests, &t.ID)
			case "parent_id":
				scanDests = append(scanDests, &parentID)
			case "description":
				scanDests = append(scanDests, &t.Description)
			case "context":
				scanDests = append(scanDests, &context)
			case "result":
				scanDests = append(scanDests, &result)
			case "priority":
				scanDests = append(scanDests, &t.Priority)
			case "completed":
				scanDests = append(scanDests, &t.Completed)
			case "cancelled":
				scanDests = append(scanDests, &t.Cancelled)
			case "archived":
				scanDests = append(scanDests, &t.Archived)
			case "created_at":
				scanDests = append(scanDests, &createdAt)
			case "updated_at":
				scanDests = append(scanDests, &updatedAt)
			case "started_at":
				scanDests = append(scanDests, &startedAt)
			case "completed_at":
				scanDests = append(scanDests, &completedAt)
			case "commit_sha":
				scanDests = append(scanDests, &commitSHA)
			case "bookmark":
				scanDests = append(scanDests, &bookmark)
			case "start_commit":
				scanDests = append(scanDests, &startCommit)
			case "cancelled_at":
				scanDests = append(scanDests, &discarded)
			case "archived_at":
				scanDests = append(scanDests, &discarded)
			}
		}

		if err := rows.Scan(scanDests...); err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}

		t.ParentID = scanNullableString(parentID)
		t.Context = scanNullableString(context)
		t.Result = scanNullableString(result)
		t.CommitSHA = scanNullableString(commitSHA)
		t.Bookmark = scanNullableString(bookmark)
		t.StartCommit = scanNullableString(startCommit)

		// Apply defaults for missing timestamp columns
		if cols["created_at"] {
			if parsed, err := parseTimestamp(createdAt.String); createdAt.Valid && err == nil {
				t.CreatedAt = parsed
			} else {
				t.CreatedAt = now
			}
		} else {
			t.CreatedAt = now
		}
		if cols["updated_at"] {
			if parsed, err := parseTimestamp(updatedAt.String); updatedAt.Valid && err == nil {
				t.UpdatedAt = parsed
			} else {
				t.UpdatedAt = t.CreatedAt
			}
		} else {
			t.UpdatedAt = t.CreatedAt
		}

		t.StartedAt = scanNullableTime(&startedAt)
		t.CompletedAt = scanNullableTime(&completedAt)

		// Default priority if column was missing (zero-value 0 means "high" in Overseer,
		// which is a valid value, so we only override if the column itself was absent)
		if !cols["priority"] {
			t.Priority = 1 // medium
		}

		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

// knownLearningColumns lists learnings columns we know how to handle.
var knownLearningColumns = []string{"id", "task_id", "content", "source_task_id", "created_at"}

func readOverseerLearnings(db *sql.DB) ([]OverseerLearning, error) {
	if !tableExists(db, "learnings") {
		return nil, nil
	}

	cols, err := getTableColumns(db, "learnings")
	if err != nil {
		return nil, fmt.Errorf("failed to discover learnings columns: %w", err)
	}

	// Hard requirements
	for _, req := range []string{"id", "content"} {
		if !cols[req] {
			return nil, fmt.Errorf("learnings table missing required column %q", req)
		}
	}

	var selectCols, missingCols []string
	for _, name := range knownLearningColumns {
		if cols[name] {
			selectCols = append(selectCols, name)
		} else {
			missingCols = append(missingCols, name)
		}
	}
	if len(missingCols) > 0 {
		fmt.Fprintf(os.Stderr, "Note: learnings table missing columns %s — using defaults\n", strings.Join(missingCols, ", "))
	}

	query := fmt.Sprintf("SELECT %s FROM learnings", strings.Join(selectCols, ", "))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()

	var learnings []OverseerLearning
	for rows.Next() {
		var l OverseerLearning
		var discarded sql.NullString // for columns we read but don't use (task_id, source_task_id)
		var createdAt sql.NullString

		scanDests := make([]any, 0, len(selectCols))
		for _, col := range selectCols {
			switch col {
			case "id":
				scanDests = append(scanDests, &l.ID)
			case "task_id":
				scanDests = append(scanDests, &discarded)
			case "content":
				scanDests = append(scanDests, &l.Content)
			case "source_task_id":
				scanDests = append(scanDests, &discarded)
			case "created_at":
				scanDests = append(scanDests, &createdAt)
			}
		}

		if err := rows.Scan(scanDests...); err != nil {
			return nil, fmt.Errorf("failed to scan learning row: %w", err)
		}

		if cols["created_at"] {
			if parsed, err := parseTimestamp(createdAt.String); createdAt.Valid && err == nil {
				l.CreatedAt = parsed
			} else {
				l.CreatedAt = now
			}
		} else {
			l.CreatedAt = now
		}

		learnings = append(learnings, l)
	}

	return learnings, rows.Err()
}

func readOverseerBlockers(db *sql.DB) ([]OverseerBlocker, error) {
	// Overseer's actual table name is task_blockers, not blockers
	if !tableExists(db, "task_blockers") {
		return nil, nil
	}

	cols, err := getTableColumns(db, "task_blockers")
	if err != nil {
		return nil, fmt.Errorf("failed to discover task_blockers columns: %w", err)
	}

	// Both columns are hard requirements — a relationship without both sides is meaningless
	for _, req := range []string{"task_id", "blocker_id"} {
		if !cols[req] {
			return nil, fmt.Errorf("task_blockers table missing required column %q", req)
		}
	}

	rows, err := db.Query(`SELECT task_id, blocker_id FROM task_blockers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blockers []OverseerBlocker
	for rows.Next() {
		var b OverseerBlocker
		if err := rows.Scan(&b.TaskID, &b.BlockerID); err != nil {
			return nil, fmt.Errorf("failed to scan blocker row: %w", err)
		}
		blockers = append(blockers, b)
	}

	return blockers, rows.Err()
}

// readOverseerMetadata reads the task_metadata table and returns a map of task_id → JSON data.
func readOverseerMetadata(db *sql.DB) (map[string]string, error) {
	if !tableExists(db, "task_metadata") {
		return nil, nil
	}

	cols, err := getTableColumns(db, "task_metadata")
	if err != nil {
		return nil, fmt.Errorf("failed to discover task_metadata columns: %w", err)
	}
	if !cols["task_id"] || !cols["data"] {
		return nil, nil
	}

	rows, err := db.Query(`SELECT task_id, data FROM task_metadata`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata := make(map[string]string)
	for rows.Next() {
		var taskID, data string
		if err := rows.Scan(&taskID, &data); err != nil {
			return nil, fmt.Errorf("failed to scan metadata row: %w", err)
		}
		if data != "" {
			metadata[taskID] = data
		}
	}

	return metadata, rows.Err()
}
