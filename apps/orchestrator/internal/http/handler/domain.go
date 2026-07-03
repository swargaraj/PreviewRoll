package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database/sqlc"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/domain"
)

// DomainHandler handles domain requests
type DomainHandler struct {
	db *database.DB
}

// NewDomainHandler creates a new domain handler
func NewDomainHandler(db *database.DB) *DomainHandler {
	return &DomainHandler{db: db}
}

// Create godoc
// @Summary      Create a domain
// @Description  Add a new domain
// @Tags         domains
// @Accept       json
// @Produce      json
// @Param        body  body  object{name=string}  true  "Domain name"
// @Success      201  {object}  map[string]int64
// @Failure      400  {object}  string
// @Failure      409  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/domains [post]
// Create adds a new domain
func (h *DomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(UserIDKey).(int64)

	if req.Name == "" {
		http.Error(w, "Domain name is required", http.StatusBadRequest)
		return
	}

	domainID, err := h.db.Q().CreateDomain(r.Context(), sqlc.CreateDomainParams{
		UserID: userID,
		Name:   req.Name,
	})
	if err != nil {
		http.Error(w, "Domain already exists or failed to create", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": domainID,
	})
}

// List godoc
// @Summary      List domains
// @Description  Get list of domains for the authenticated user
// @Tags         domains
// @Produce      json
// @Success      200  {array}   object
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/domains [get]
// List returns all domains for the current user
func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(int64)

	domains, err := h.db.Q().ListDomainsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to query domains", http.StatusInternalServerError)
		return
	}

	if domains == nil {
		domains = []sqlc.Domain{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

// Delete godoc
// @Summary      Delete a domain
// @Description  Delete a domain by ID
// @Tags         domains
// @Produce      json
// @Param        id   path  int  true  "Domain ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/domains/{id} [delete]
// Delete removes a domain
func (h *DomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid domain ID", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(UserIDKey).(int64)

	err = h.db.Q().DeleteDomainByIDAndUser(r.Context(), sqlc.DeleteDomainByIDAndUserParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Domain deleted",
	})
}

// ConnectProject godoc
// @Summary      Connect domain to project
// @Description  Connect a domain to a project with a prefix
// @Tags         domains
// @Accept       json
// @Produce      json
// @Param        id    path  int                       true  "Domain ID"
// @Param        body  body  domain.ConnectProjectRequest  true  "Project connection details"
// @Success      201  {object}  map[string]int64
// @Failure      400  {object}  string
// @Failure      404  {object}  string
// @Failure      409  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/domains/{id}/projects [post]
// ConnectProject connects a domain to a project with a prefix
func (h *DomainHandler) ConnectProject(w http.ResponseWriter, r *http.Request) {
	domainIDStr := r.PathValue("id")
	domainID, err := strconv.ParseInt(domainIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid domain ID", http.StatusBadRequest)
		return
	}

	var req domain.ConnectProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(UserIDKey).(int64)

	if req.ProjectID == 0 {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	domainOwnerID, err := h.db.Q().GetDomainByID(r.Context(), domainID)
	if err != nil || domainOwnerID != userID {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	projectOwnerID, err := h.db.Q().GetProjectOwnerByID(r.Context(), req.ProjectID)
	if err != nil || projectOwnerID != userID {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	prefix := req.Prefix
	if prefix == "" {
		prefix = "preview"
	}

	projectDomainID, err := h.db.Q().CreateProjectDomain(r.Context(), sqlc.CreateProjectDomainParams{
		ProjectID: req.ProjectID,
		DomainID:  domainID,
		Prefix:    prefix,
	})
	if err != nil {
		http.Error(w, "Domain already connected to this project or failed to connect", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": projectDomainID,
	})
}

// DisconnectProject godoc
// @Summary      Disconnect domain from project
// @Description  Remove a domain connection from a project
// @Tags         domains
// @Produce      json
// @Param        id         path  int  true  "Domain ID"
// @Param        projectId  path  int  true  "Project ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string
// @Failure      404  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/domains/{id}/projects/{projectId} [delete]
// DisconnectProject removes a domain from a project
func (h *DomainHandler) DisconnectProject(w http.ResponseWriter, r *http.Request) {
	domainIDStr := r.PathValue("id")
	domainID, err := strconv.ParseInt(domainIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid domain ID", http.StatusBadRequest)
		return
	}

	projectIDStr := r.PathValue("projectId")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(UserIDKey).(int64)

	domainOwnerID, err := h.db.Q().GetDomainByID(r.Context(), domainID)
	if err != nil || domainOwnerID != userID {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	err = h.db.Q().DeleteProjectDomain(r.Context(), sqlc.DeleteProjectDomainParams{
		DomainID:  domainID,
		ProjectID: projectID,
	})
	if err != nil {
		http.Error(w, "Domain not connected to this project", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Domain disconnected from project",
	})
}

// ListProjectDomains godoc
// @Summary      List project domains
// @Description  Get all domains connected to a project
// @Tags         domains
// @Produce      json
// @Param        projectId  path  int  true  "Project ID"
// @Success      200  {array}   object
// @Failure      400  {object}  string
// @Failure      404  {object}  string
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/projects/{projectId}/domains [get]
// ListProjectDomains returns all domains connected to a project
func (h *DomainHandler) ListProjectDomains(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.PathValue("projectId")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(UserIDKey).(int64)

	projectOwnerID, err := h.db.Q().GetProjectOwnerByID(r.Context(), projectID)
	if err != nil || projectOwnerID != userID {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	projectDomains, err := h.db.Q().ListProjectDomains(r.Context(), projectID)
	if err != nil {
		http.Error(w, "Failed to query project domains", http.StatusInternalServerError)
		return
	}

	if projectDomains == nil {
		projectDomains = []sqlc.ListProjectDomainsRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projectDomains)
}
