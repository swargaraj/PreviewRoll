package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database/sqlc"
)

// ProjectHandler handles project requests
type ProjectHandler struct {
	db *database.DB
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(db *database.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

// List returns a list of projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(int64)

	projects, err := h.db.Q().ListProjectsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to query projects", http.StatusInternalServerError)
		return
	}

	if projects == nil {
		projects = []sqlc.Project{}
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

	project, err := h.db.Q().GetProjectByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name          string `json:"name"`
	GithubRepo    string `json:"github_repo"`
	GithubRepoID  int64  `json:"github_repo_id"`
	WebhookSecret string `json:"webhook_secret"`
}

// Create creates a new project
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(UserIDKey).(int64)

	if req.Name == "" || req.GithubRepo == "" {
		http.Error(w, "Name and github_repo are required", http.StatusBadRequest)
		return
	}

	project, err := h.db.Q().CreateProject(r.Context(), sqlc.CreateProjectParams{
		UserID:        userID,
		Name:          req.Name,
		GithubRepo:    req.GithubRepo,
		GithubRepoID:  req.GithubRepoID,
		WebhookSecret: req.WebhookSecret,
	})
	if err != nil {
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": project.ID,
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

	err = h.db.Q().DeleteProject(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Project deleted",
	})
}
