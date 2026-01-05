// Package http provides an HTTP REST API server for Tasuku.
package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/iheanyi/tasuku/internal/store"
	"github.com/iheanyi/tasuku/internal/task"
)

// Server is the HTTP API server.
type Server struct {
	store *store.Store
	mux   *http.ServeMux
}

// New creates a new HTTP server.
func New(s *store.Store) *Server {
	srv := &Server{
		store: s,
		mux:   http.NewServeMux(),
	}
	srv.registerRoutes()
	return srv
}

// ServeHTTP implements http.Handler for testing.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Run starts the HTTP server on the given address.
func (s *Server) Run(addr string) error {
	fmt.Printf("Starting HTTP server on %s\n", addr)
	fmt.Println("Endpoints:")
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
	s.mux.HandleFunc("/tasks", s.handleTasks)
	s.mux.HandleFunc("/tasks/", s.handleTask)
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

	id := req.ID
	if id == "" {
		id = generateID(req.Description)
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

	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
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

// generateID creates a kebab-case ID from description.
func generateID(desc string) string {
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
	if len(result) > 32 {
		result = result[:32]
	}
	return strings.TrimSuffix(result, "-")
}
