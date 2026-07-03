package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"math"
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

// PaginatedResponse represents a paginated list response
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// List godoc
// @Summary      List projects
// @Description  Get paginated list of projects for the authenticated user
// @Tags         projects
// @Produce      json
// @Param        page       query  int     false  "Page number"     default(1)
// @Param        page_size  query  int     false  "Items per page"  default(10)
// @Param        search     query  string false  "Search term"
// @Success      200  {object}  PaginatedResponse
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/projects [get]
// List returns a paginated list of projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(int64)

	page := 1
	pageSize := 10
	search := r.URL.Query().Get("search")

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	var total int64
	var err error

	if search != "" {
		total, err = h.db.Q().CountProjectsByUserIDSearch(r.Context(), sqlc.CountProjectsByUserIDSearchParams{
			UserID:  userID,
			Column2: sql.NullString{String: search, Valid: true},
			Column3: sql.NullString{String: search, Valid: true},
		})
	} else {
		total, err = h.db.Q().CountProjectsByUserID(r.Context(), userID)
	}
	if err != nil {
		slog.Error("failed to count projects", "error", err, "user_id", userID)
		http.Error(w, "Failed to count projects", http.StatusInternalServerError)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	offset := int64((page - 1) * pageSize)

	var projects []sqlc.Project

	if search != "" {
		projects, err = h.db.Q().ListProjectsByUserIDPaginatedSearch(r.Context(), sqlc.ListProjectsByUserIDPaginatedSearchParams{
			UserID:  userID,
			Column2: sql.NullString{String: search, Valid: true},
			Column3: sql.NullString{String: search, Valid: true},
			Limit:   int64(pageSize),
			Offset:  offset,
		})
	} else {
		projects, err = h.db.Q().ListProjectsByUserIDPaginated(r.Context(), sqlc.ListProjectsByUserIDPaginatedParams{
			UserID: userID,
			Limit:  int64(pageSize),
			Offset: offset,
		})
	}
	if err != nil {
		slog.Error("failed to query projects", "error", err, "user_id", userID)
		http.Error(w, "Failed to query projects", http.StatusInternalServerError)
		return
	}

	if projects == nil {
		projects = []sqlc.Project{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PaginatedResponse{
		Items:      projects,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// GetByID godoc
// @Summary      Get project by ID
// @Description  Returns a single project
// @Tags         projects
// @Produce      json
// @Param        id   path  int  true  "Project ID"
// @Success      200  {object}  object
// @Failure      400  {object}  string
// @Failure      404  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/projects/{id} [get]
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
	RepoURL       string `json:"repo_url"`
	VcsProvider   string `json:"vcs_provider"`
	WebhookSecret string `json:"webhook_secret"`
}

// Create godoc
// @Summary      Create a project
// @Description  Create a new project
// @Tags         projects
// @Accept       json
// @Produce      json
// @Param        body  body  CreateProjectRequest  true  "Project details"
// @Success      201  {object}  map[string]int64
// @Failure      400  {object}  string
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/projects [post]
// Create creates a new project
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(UserIDKey).(int64)

	if req.Name == "" || req.RepoURL == "" {
		http.Error(w, "Name and repo_url are required", http.StatusBadRequest)
		return
	}

	if req.VcsProvider == "" {
		req.VcsProvider = "github"
	}

	project, err := h.db.Q().CreateProject(r.Context(), sqlc.CreateProjectParams{
		UserID:        userID,
		Name:          req.Name,
		RepoUrl:       req.RepoURL,
		VcsProvider:   req.VcsProvider,
		WebhookSecret: req.WebhookSecret,
	})
	if err != nil {
		slog.Error("failed to create project", "error", err, "user_id", userID, "name", req.Name)
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": project.ID,
	})
}

// Delete godoc
// @Summary      Delete a project
// @Description  Delete a project by ID
// @Tags         projects
// @Produce      json
// @Param        id   path  int  true  "Project ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/projects/{id} [delete]
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
		slog.Error("failed to delete project", "error", err, "project_id", id)
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Project deleted",
	})
}
