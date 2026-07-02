package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
)

// ProjectHandler handles project requests
type ProjectHandler struct {
	db *database.DB
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(db *database.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

// Project represents a project
type Project struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	Name          string `json:"name"`
	GithubRepo    string `json:"github_repo"`
	GithubRepoID  int64  `json:"github_repo_id"`
	WebhookSecret string `json:"-"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name          string `json:"name"`
	GithubRepo    string `json:"github_repo"`
	GithubRepoID  int64  `json:"github_repo_id"`
	WebhookSecret string `json:"webhook_secret"`
}

// List returns a list of projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.db.Query(`
		SELECT id, user_id, name, github_repo, github_repo_id, created_at, updated_at
		FROM projects WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		http.Error(w, "Failed to query projects", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		p := &Project{}
		err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.GithubRepo, &p.GithubRepoID, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			http.Error(w, "Failed to scan project", http.StatusInternalServerError)
			return
		}
		projects = append(projects, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// GetByID returns a project by ID
func (h *ProjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	p := &Project{}
	err = h.db.QueryRow(`
		SELECT id, user_id, name, github_repo, github_repo_id, created_at, updated_at
		FROM projects WHERE id = ?
	`, id).Scan(&p.ID, &p.UserID, &p.Name, &p.GithubRepo, &p.GithubRepoID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// Create creates a new project
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from context
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate input
	if req.Name == "" || req.GithubRepo == "" {
		http.Error(w, "Name and github_repo are required", http.StatusBadRequest)
		return
	}

	// Insert project
	var projectID int64
	err := h.db.QueryRow(`
		INSERT INTO projects (user_id, name, github_repo, github_repo_id, webhook_secret)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`, userID, req.Name, req.GithubRepo, req.GithubRepoID, req.WebhookSecret).Scan(&projectID)
	if err != nil {
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": projectID,
	})
}

// Delete deletes a project
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	// Delete project
	result, err := h.db.Exec(`
		DELETE FROM projects WHERE id = ?
	`, id)
	if err != nil {
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Project deleted",
	})
}
