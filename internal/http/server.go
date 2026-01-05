// Package http provides an HTTP REST API server for Tasuku.
package http

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

//go:embed templates/*.html
var templateFS embed.FS

// Server is the HTTP API server.
type Server struct {
	store     store.Storage
	mux       *http.ServeMux
	templates *template.Template
}

// New creates a new HTTP server.
func New(s store.Storage) *Server {
	// Parse templates
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		panic(fmt.Sprintf("failed to parse templates: %v", err))
	}

	srv := &Server{
		store:     s,
		mux:       http.NewServeMux(),
		templates: tmpl,
	}
	srv.registerRoutes()
	return srv
}

// ServeHTTP implements http.Handler for testing.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// checkPortAvailable checks if a port is available and returns info about what's using it if not.
func checkPortAvailable(addr string) error {
	// Try to listen on the address
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// Port is in use, try to find out what's using it
		processInfo := getProcessUsingPort(addr)
		if processInfo != "" {
			return fmt.Errorf("port %s is already in use by: %s", addr, processInfo)
		}
		return fmt.Errorf("port %s is already in use (couldn't identify process)", addr)
	}
	ln.Close()
	return nil
}

// getProcessUsingPort tries to find what process is using a port.
func getProcessUsingPort(addr string) string {
	// Extract port from address (e.g., ":8080" -> "8080", "localhost:8080" -> "8080")
	port := addr
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		port = addr[idx+1:]
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		// Use lsof on Unix-like systems
		cmd = exec.Command("lsof", "-i", ":"+port, "-sTCP:LISTEN", "-n", "-P")
	default:
		return ""
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse lsof output - second line contains the process info
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return ""
	}

	// Parse the second line to get process name and PID
	fields := strings.Fields(lines[1])
	if len(fields) >= 2 {
		return fmt.Sprintf("%s (PID %s)", fields[0], fields[1])
	}
	return strings.TrimSpace(lines[1])
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	// Check if port is available before starting
	if err := checkPortAvailable(addr); err != nil {
		return fmt.Errorf("cannot start server: %w\n\nTo fix this:\n  1. Stop the existing process, or\n  2. Use a different port: tk serve --http :8081", err)
	}

	fmt.Printf("Starting HTTP server on %s\n", addr)
	fmt.Println("")
	fmt.Println("Web Dashboard:")
	fmt.Println("  GET    /                          - Dashboard with task list")
	fmt.Println("")
	fmt.Println("API Endpoints:")
	fmt.Println("  GET    /tasks                     - List all tasks (?status=X, ?tag=X)")
	fmt.Println("  POST   /tasks                     - Create a task")
	fmt.Println("  GET    /tasks/{id}                - Get task details")
	fmt.Println("  PUT    /tasks/{id}                - Update task")
	fmt.Println("  DELETE /tasks/{id}                - Delete task")
	fmt.Println("  POST   /tasks/{id}/timer/start    - Start timer on task")
	fmt.Println("  POST   /tasks/{id}/timer/stop     - Stop timer on task")
	fmt.Println("  PUT    /tasks/{id}/fields/{key}   - Set custom field")
	fmt.Println("  DELETE /tasks/{id}/fields/{key}   - Remove custom field")
	fmt.Println("  PUT    /tasks/{id}/tags/{tag}     - Add tag to task")
	fmt.Println("  DELETE /tasks/{id}/tags/{tag}     - Remove tag from task")
	fmt.Println("  POST   /tasks/{id}/archive        - Archive a done task")
	fmt.Println("  GET    /archive                   - List all archived tasks")
	fmt.Println("  POST   /archive/{id}/restore      - Restore an archived task")
	fmt.Println("  DELETE /archive                   - Clear all archived tasks")
	fmt.Println("  GET    /ready                     - List ready tasks")
	fmt.Println("  GET    /timers                    - List active timers")
	fmt.Println("  GET    /context                   - Get full context")
	fmt.Println("  POST   /learnings                 - Add a learning")
	fmt.Println("  POST   /decisions                 - Add a decision")
	fmt.Println("  GET    /schema                    - Get JSON schema")
	fmt.Println("  GET    /health                    - Health check")
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) registerRoutes() {
	// Web dashboard routes
	s.mux.HandleFunc("/", s.handleDashboard)
	s.mux.HandleFunc("/partials/tasks", s.handlePartialTasks)
	s.mux.HandleFunc("/partials/stats", s.handlePartialStats)
	s.mux.HandleFunc("/partials/progress", s.handlePartialProgress)
	s.mux.HandleFunc("/partials/task/", s.handlePartialTask)

	// API routes
	s.mux.HandleFunc("/tasks", s.handleTasks)
	s.mux.HandleFunc("/tasks/", s.handleTask)
	s.mux.HandleFunc("/archive", s.handleArchive)
	s.mux.HandleFunc("/archive/", s.handleArchiveItem)
	s.mux.HandleFunc("/ready", s.handleReady)
	s.mux.HandleFunc("/context", s.handleContext)
	s.mux.HandleFunc("/learnings", s.handleLearnings)
	s.mux.HandleFunc("/decisions", s.handleDecisions)
	s.mux.HandleFunc("/timers", s.handleTimers)
	s.mux.HandleFunc("/schema", s.handleSchema)
	s.mux.HandleFunc("/health", s.handleHealth)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case "GET":
		s.listTasks(w, r)
	case "POST":
		s.createTask(w, r)
	default:
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	f, err := s.store.Read()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	status := r.URL.Query().Get("status")
	tag := r.URL.Query().Get("tag")

	type taskResponse struct {
		ID           string            `json:"id"`
		Status       string            `json:"status"`
		Description  string            `json:"description"`
		Priority     int               `json:"priority"`
		BlockedBy    []string          `json:"blocked_by"`
		Owner        *string           `json:"owner"`
		Tags         []string          `json:"tags,omitempty"`
		Fields       map[string]string `json:"fields,omitempty"`
		TimerRunning bool              `json:"timer_running"`
		Duration     string            `json:"duration,omitempty"`
		CreatedAt    time.Time         `json:"created_at"`
		UpdatedAt    time.Time         `json:"updated_at"`
	}

	var tasks []taskResponse
	for id, t := range f.Tasks {
		if status != "" && string(t.Status) != status {
			continue
		}
		if tag != "" && !t.HasTag(tag) {
			continue
		}
		var durationStr string
		if t.Duration > 0 {
			durationStr = t.Duration.FormatHumanReadable()
		}
		tasks = append(tasks, taskResponse{
			ID:           id,
			Status:       string(t.Status),
			Description:  t.Description,
			Priority:     t.GetPriority(),
			BlockedBy:    t.BlockedBy,
			Owner:        t.Owner,
			Tags:         t.Tags,
			Fields:       t.Fields,
			TimerRunning: t.IsTimerRunning(),
			Duration:     durationStr,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}

	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          string   `json:"id"`
		Description string   `json:"description"`
		Priority    *int     `json:"priority"`
		Tags        []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Description == "" {
		http.Error(w, `{"error": "description required"}`, http.StatusBadRequest)
		return
	}

	// Generate ID if not provided, checking for collisions
	id := req.ID
	if id == "" {
		existingIDs := make(map[string]struct{})
		if f, err := s.store.Read(); err == nil {
			for taskID := range f.Tasks {
				existingIDs[taskID] = struct{}{}
			}
		}
		id = task.GenerateTaskID(req.Description, existingIDs)
	}

	if err := s.store.AddTaskWithTags(id, req.Description, req.Priority, req.Tags); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "created"})
}

func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract task ID and sub-path: /tasks/{id}/timer/start, /tasks/{id}/fields/{key}, etc.
	path := strings.TrimPrefix(r.URL.Path, "/tasks/")
	parts := strings.Split(path, "/")
	taskID := parts[0]

	if taskID == "" {
		http.Error(w, `{"error": "task ID required"}`, http.StatusBadRequest)
		return
	}

	// Handle sub-routes
	if len(parts) >= 2 {
		switch parts[1] {
		case "timer":
			if len(parts) >= 3 {
				s.handleTaskTimer(w, r, taskID, parts[2])
				return
			}
		case "fields":
			if len(parts) >= 3 {
				s.handleTaskField(w, r, taskID, parts[2])
				return
			}
		case "tags":
			if len(parts) >= 3 {
				s.handleTaskTag(w, r, taskID, parts[2])
				return
			}
		case "archive":
			s.handleTaskArchive(w, r, taskID)
			return
		}
	}

	switch r.Method {
	case "GET":
		s.getTask(w, r, taskID)
	case "PUT":
		s.updateTask(w, r, taskID)
	case "DELETE":
		s.deleteTask(w, r, taskID)
	default:
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request, taskID string) {
	f, err := s.store.Read()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	t, exists := f.Tasks[taskID]
	if !exists {
		http.Error(w, `{"error": "task not found"}`, http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":            taskID,
		"status":        t.Status,
		"description":   t.Description,
		"priority":      t.GetPriority(),
		"blocked_by":    t.BlockedBy,
		"owner":         t.Owner,
		"tags":          t.Tags,
		"fields":        t.Fields,
		"timer_running": t.IsTimerRunning(),
		"created_at":    t.CreatedAt,
		"updated_at":    t.UpdatedAt,
	}

	if t.Duration > 0 {
		response["duration"] = t.Duration.FormatHumanReadable()
	}

	if t.TimerStart != nil {
		response["timer_start"] = t.TimerStart
	}

	if notes, ok := f.Context.Notes[taskID]; ok {
		response["notes"] = notes
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request, taskID string) {
	var req struct {
		Status   *string `json:"status"`
		Priority *int    `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Status != nil {
		status := task.Status(*req.Status)
		if err := s.store.SetStatus(taskID, status); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
	}

	if req.Priority != nil {
		if err := s.store.SetPriority(taskID, *req.Priority); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
	}

	// Return HTML partial for HTMX requests
	if strings.Contains(r.Header.Get("Accept"), "text/html") || r.Header.Get("HX-Request") == "true" {
		s.renderTaskPartial(w, taskID)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// renderTaskPartial renders a single task row partial for HTMX.
func (s *Server) renderTaskPartial(w http.ResponseWriter, taskID string) {
	f, err := s.store.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t, exists := f.Tasks[taskID]
	if !exists {
		// Task was archived or deleted, return empty div that will be removed
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(""))
		return
	}

	var durationStr string
	if t.Duration > 0 {
		durationStr = t.Duration.FormatHumanReadable()
	}

	var priorityName string
	if t.Priority != nil {
		priorityName = task.PriorityName(*t.Priority)
	}

	view := TaskView{
		ID:           taskID,
		Status:       string(t.Status),
		Description:  t.Description,
		Priority:     t.Priority,
		PriorityName: priorityName,
		Tags:         t.Tags,
		TimerRunning: t.IsTimerRunning(),
		Duration:     durationStr,
		CreatedAt:    t.CreatedAt,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "task_row", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request, taskID string) {
	err := s.store.Update(func(f *task.File) error {
		if _, exists := f.Tasks[taskID]; !exists {
			return fmt.Errorf("task not found")
		}
		delete(f.Tasks, taskID)
		delete(f.Context.Notes, taskID)
		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	f, err := s.store.Read()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	type taskResponse struct {
		ID          string    `json:"id"`
		Description string    `json:"description"`
		Priority    int       `json:"priority"`
		CreatedAt   time.Time `json:"created_at"`
	}

	var tasks []taskResponse
	for id, t := range f.Tasks {
		if t.Status == task.StatusReady {
			blocked := false
			for _, blockerID := range t.BlockedBy {
				if blocker, exists := f.Tasks[blockerID]; exists && blocker.Status != task.StatusDone {
					blocked = true
					break
				}
			}
			if !blocked {
				tasks = append(tasks, taskResponse{
					ID:          id,
					Description: t.Description,
					Priority:    t.GetPriority(),
					CreatedAt:   t.CreatedAt,
				})
			}
		}
	}

	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	f, err := s.store.Read()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(f)
}

func (s *Server) handleLearnings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Learning string `json:"learning"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Learning == "" {
		http.Error(w, `{"error": "learning required"}`, http.StatusBadRequest)
		return
	}

	id, err := s.store.AddLearning(req.Learning)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "added"})
}

func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID      string   `json:"id"`
		Chose   string   `json:"chose"`
		Over    []string `json:"over"`
		Because string   `json:"because"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Chose == "" || req.Because == "" {
		http.Error(w, `{"error": "id, chose, and because are required"}`, http.StatusBadRequest)
		return
	}

	d := task.Decision{
		ID:        req.ID,
		Chose:     req.Chose,
		Over:      req.Over,
		Because:   req.Because,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.store.AddDecision(d); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added", "id": req.ID})
}

func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	schema := `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/iheanyi/tasuku/schema.json",
  "title": "Tasuku File",
  "description": "Schema for .tasuku.json task management file",
  "type": "object",
  "required": ["version", "tasks", "context"]
}`
	fmt.Fprint(w, schema)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleTimers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	timers, err := s.store.GetActiveTimers()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	type timerInfo struct {
		ID          string    `json:"id"`
		Description string    `json:"description"`
		StartedAt   time.Time `json:"started_at"`
		Elapsed     string    `json:"elapsed"`
	}

	var results []timerInfo
	for id, t := range timers {
		results = append(results, timerInfo{
			ID:          id,
			Description: t.Description,
			StartedAt:   *t.TimerStart,
			Elapsed:     time.Since(*t.TimerStart).Truncate(time.Second).String(),
		})
	}

	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleTaskTimer(w http.ResponseWriter, r *http.Request, taskID, action string) {
	if r.Method != "POST" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	switch action {
	case "start":
		if err := s.store.StartTimer(taskID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": taskID, "status": "timer_started"})

	case "stop":
		elapsed, err := s.store.StopTimer(taskID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      taskID,
			"status":  "timer_stopped",
			"elapsed": elapsed.String(),
		})

	default:
		http.Error(w, `{"error": "unknown timer action"}`, http.StatusBadRequest)
	}
}

func (s *Server) handleTaskField(w http.ResponseWriter, r *http.Request, taskID, key string) {
	switch r.Method {
	case "PUT":
		var req struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if err := s.store.SetField(taskID, key, req.Value); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": taskID, "key": key, "value": req.Value, "status": "set"})

	case "DELETE":
		if err := s.store.RemoveField(taskID, key); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": taskID, "key": key, "status": "removed"})

	default:
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTaskTag(w http.ResponseWriter, r *http.Request, taskID, tag string) {
	switch r.Method {
	case "PUT":
		if err := s.store.AddTag(taskID, tag); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": taskID, "tag": tag, "status": "added"})

	case "DELETE":
		if err := s.store.RemoveTag(taskID, tag); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": taskID, "tag": tag, "status": "removed"})

	default:
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleTaskArchive handles POST /tasks/{id}/archive
func (s *Server) handleTaskArchive(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != "POST" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Summary string `json:"summary"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := s.store.ArchiveTask(taskID, req.Summary); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Return empty HTML for HTMX to remove the task row
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(""))
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"id": taskID, "status": "archived"})
}

// handleArchive handles GET /archive and DELETE /archive
func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case "GET":
		s.listArchivedTasks(w, r)
	case "DELETE":
		s.clearArchive(w, r)
	default:
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) listArchivedTasks(w http.ResponseWriter, r *http.Request) {
	archived, err := s.store.GetArchivedTasks()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	type archivedResponse struct {
		ID          string    `json:"id"`
		Description string    `json:"description"`
		Summary     string    `json:"summary,omitempty"`
		ArchivedAt  time.Time `json:"archived_at"`
		TotalTime   string    `json:"total_time,omitempty"`
	}

	var tasks []archivedResponse
	for id, t := range archived {
		var totalTime string
		if t.TotalTime > 0 {
			totalTime = t.TotalTime.FormatHumanReadable()
		}
		tasks = append(tasks, archivedResponse{
			ID:          id,
			Description: t.Description,
			Summary:     t.Summary,
			ArchivedAt:  t.ArchivedAt,
			TotalTime:   totalTime,
		})
	}

	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) clearArchive(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.ClearArchive()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"status": "cleared", "count": count})
}

// handleArchiveItem handles POST /archive/{id}/restore
func (s *Server) handleArchiveItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract archive ID and action: /archive/{id}/restore
	path := strings.TrimPrefix(r.URL.Path, "/archive/")
	parts := strings.Split(path, "/")
	archiveID := parts[0]

	if archiveID == "" {
		http.Error(w, `{"error": "archive ID required"}`, http.StatusBadRequest)
		return
	}

	if len(parts) >= 2 && parts[1] == "restore" {
		if r.Method != "POST" {
			http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		if err := s.store.RestoreTask(archiveID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"id": archiveID, "status": "restored"})
		return
	}

	http.Error(w, `{"error": "not found"}`, http.StatusNotFound)
}

// generateID creates a kebab-case ID from description (without collision check).
// Used for tests. Production code should use task.GenerateTaskID with existingIDs.
func generateID(desc string) string {
	return task.GenerateTaskID(desc, nil)
}

// Template data types for the web dashboard

// DashboardStats holds statistics about tasks.
type DashboardStats struct {
	Total             int
	Ready             int
	InProgress        int
	Blocked           int
	Done              int
	ReadyPercent      float64
	InProgressPercent float64
	BlockedPercent    float64
	DonePercent       float64
}

// TaskView is a view model for rendering a task in templates.
type TaskView struct {
	ID           string
	Status       string
	Description  string
	Priority     *int
	PriorityName string
	Tags         []string
	TimerRunning bool
	Duration     string
	CreatedAt    time.Time
}

// ArchivedView is a view model for rendering an archived task.
type ArchivedView struct {
	ID          string
	Description string
	Summary     string
	TotalTime   string
	ArchivedAt  time.Time
}

// DashboardData holds all data for the dashboard template.
type DashboardData struct {
	Title    string
	Stats    DashboardStats
	Tasks    []TaskView
	Archived []ArchivedView
	Filter   string
}

// calculateStats computes task statistics from the store.
func (s *Server) calculateStats() (DashboardStats, error) {
	f, err := s.store.Read()
	if err != nil {
		return DashboardStats{}, err
	}

	stats := DashboardStats{}
	for _, t := range f.Tasks {
		stats.Total++
		switch t.Status {
		case task.StatusReady:
			stats.Ready++
		case task.StatusInProgress:
			stats.InProgress++
		case task.StatusBlocked:
			stats.Blocked++
		case task.StatusDone:
			stats.Done++
		}
	}

	if stats.Total > 0 {
		stats.ReadyPercent = float64(stats.Ready) / float64(stats.Total) * 100
		stats.InProgressPercent = float64(stats.InProgress) / float64(stats.Total) * 100
		stats.BlockedPercent = float64(stats.Blocked) / float64(stats.Total) * 100
		stats.DonePercent = float64(stats.Done) / float64(stats.Total) * 100
	}

	return stats, nil
}

// getTaskViews returns task views, optionally filtered by status.
func (s *Server) getTaskViews(statusFilter string) ([]TaskView, error) {
	f, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	var tasks []TaskView
	for id, t := range f.Tasks {
		if statusFilter != "" && string(t.Status) != statusFilter {
			continue
		}

		var durationStr string
		if t.Duration > 0 {
			durationStr = t.Duration.FormatHumanReadable()
		}

		var priorityName string
		if t.Priority != nil {
			priorityName = task.PriorityName(*t.Priority)
		}

		tasks = append(tasks, TaskView{
			ID:           id,
			Status:       string(t.Status),
			Description:  t.Description,
			Priority:     t.Priority,
			PriorityName: priorityName,
			Tags:         t.Tags,
			TimerRunning: t.IsTimerRunning(),
			Duration:     durationStr,
			CreatedAt:    t.CreatedAt,
		})
	}

	// Sort by priority (lower is higher priority), then by creation date
	sort.Slice(tasks, func(i, j int) bool {
		pi, pj := 2, 2 // default priority
		if tasks[i].Priority != nil {
			pi = *tasks[i].Priority
		}
		if tasks[j].Priority != nil {
			pj = *tasks[j].Priority
		}
		if pi != pj {
			return pi < pj
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	return tasks, nil
}

// getArchivedViews returns archived task views.
func (s *Server) getArchivedViews() ([]ArchivedView, error) {
	archived, err := s.store.GetArchivedTasks()
	if err != nil {
		return nil, err
	}

	var views []ArchivedView
	for id, t := range archived {
		var totalTime string
		if t.TotalTime > 0 {
			totalTime = t.TotalTime.FormatHumanReadable()
		}
		views = append(views, ArchivedView{
			ID:          id,
			Description: t.Description,
			Summary:     t.Summary,
			TotalTime:   totalTime,
			ArchivedAt:  t.ArchivedAt,
		})
	}

	// Sort by archived date descending
	sort.Slice(views, func(i, j int) bool {
		return views[i].ArchivedAt.After(views[j].ArchivedAt)
	})

	return views, nil
}

// handleDashboard serves the main dashboard page.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Only handle root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	stats, err := s.calculateStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tasks, err := s.getTaskViews("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	archived, err := s.getArchivedViews()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := DashboardData{
		Title:    "Dashboard",
		Stats:    stats,
		Tasks:    tasks,
		Archived: archived,
		Filter:   "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Note: We can't use http.Error after template execution starts since
	// headers are already sent once we begin writing the template.
	// Errors here will be logged but not displayed to user.
	_ = s.templates.ExecuteTemplate(w, "layout.html", data)
}

// handlePartialTasks serves a partial HTML fragment for HTMX updates.
func (s *Server) handlePartialTasks(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	tasks, err := s.getTaskViews(statusFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(tasks) == 0 {
		w.Write([]byte(`<div class="empty-state"><h3>No tasks found</h3><p>Create a task with <code>tk add "description"</code></p></div>`))
		return
	}

	for _, t := range tasks {
		if err := s.templates.ExecuteTemplate(w, "task_row", t); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// handlePartialStats serves the stats partial for HTMX updates.
func (s *Server) handlePartialStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.calculateStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "stats", stats); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handlePartialProgress serves the progress bar partial for HTMX updates.
func (s *Server) handlePartialProgress(w http.ResponseWriter, r *http.Request) {
	stats, err := s.calculateStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := DashboardData{Stats: stats}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "progress", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handlePartialTask serves a single task row for HTMX updates after actions.
func (s *Server) handlePartialTask(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/partials/task/")
	if taskID == "" {
		http.Error(w, "task ID required", http.StatusBadRequest)
		return
	}

	f, err := s.store.Read()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t, exists := f.Tasks[taskID]
	if !exists {
		// Task was archived or deleted, return empty
		w.WriteHeader(http.StatusOK)
		return
	}

	var durationStr string
	if t.Duration > 0 {
		durationStr = t.Duration.FormatHumanReadable()
	}

	var priorityName string
	if t.Priority != nil {
		priorityName = task.PriorityName(*t.Priority)
	}

	view := TaskView{
		ID:           taskID,
		Status:       string(t.Status),
		Description:  t.Description,
		Priority:     t.Priority,
		PriorityName: priorityName,
		Tags:         t.Tags,
		TimerRunning: t.IsTimerRunning(),
		Duration:     durationStr,
		CreatedAt:    t.CreatedAt,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "task_row", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
