package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
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

// Create adds a new domain
func (h *DomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if req.Name == "" {
		http.Error(w, "Domain name is required", http.StatusBadRequest)
		return
	}

	var domainID int64
	err := h.db.QueryRow(`
		INSERT INTO domains (user_id, name)
		VALUES (?, ?)
		RETURNING id
	`, userID, req.Name).Scan(&domainID)
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

// List returns all domains for the current user
func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.db.Query(`
		SELECT id, user_id, name, created_at, updated_at
		FROM domains WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		http.Error(w, "Failed to query domains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var domains []*domain.Domain
	for rows.Next() {
		d := &domain.Domain{}
		err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			http.Error(w, "Failed to scan domain", http.StatusInternalServerError)
			return
		}
		domains = append(domains, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

// Delete removes a domain
func (h *DomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid domain ID", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	result, err := h.db.Exec(`
		DELETE FROM domains WHERE id = ? AND user_id = ?
	`, id, userID)
	if err != nil {
		http.Error(w, "Failed to delete domain", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Domain deleted",
	})
}

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

	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if req.ProjectID == 0 {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	// Verify domain belongs to user
	var count int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM domains WHERE id = ? AND user_id = ?
	`, domainID, userID).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	// Verify project belongs to user
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM projects WHERE id = ? AND user_id = ?
	`, req.ProjectID, userID).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Set default prefix
	prefix := req.Prefix
	if prefix == "" {
		prefix = "preview"
	}

	// Connect domain to project
	var projectDomainID int64
	err = h.db.QueryRow(`
		INSERT INTO project_domains (project_id, domain_id, prefix)
		VALUES (?, ?, ?)
		RETURNING id
	`, req.ProjectID, domainID, prefix).Scan(&projectDomainID)
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

	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify domain belongs to user
	var count int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM domains WHERE id = ? AND user_id = ?
	`, domainID, userID).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Domain not found", http.StatusNotFound)
		return
	}

	result, err := h.db.Exec(`
		DELETE FROM project_domains WHERE domain_id = ? AND project_id = ?
	`, domainID, projectID)
	if err != nil {
		http.Error(w, "Failed to disconnect domain from project", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Domain not connected to this project", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Domain disconnected from project",
	})
}

// ListProjectDomains returns all domains connected to a project
func (h *DomainHandler) ListProjectDomains(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.PathValue("projectId")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify project belongs to user
	var count int
	err = h.db.QueryRow(`
		SELECT COUNT(*) FROM projects WHERE id = ? AND user_id = ?
	`, projectID, userID).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	rows, err := h.db.Query(`
		SELECT pd.id, pd.project_id, pd.domain_id, d.name, pd.prefix
		FROM project_domains pd
		INNER JOIN domains d ON pd.domain_id = d.id
		WHERE pd.project_id = ?
	`, projectID)
	if err != nil {
		http.Error(w, "Failed to query project domains", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projectDomains []*domain.ProjectDomainDetail
	for rows.Next() {
		pd := &domain.ProjectDomainDetail{}
		err := rows.Scan(&pd.ID, &pd.ProjectID, &pd.DomainID, &pd.DomainName, &pd.Prefix)
		if err != nil {
			http.Error(w, "Failed to scan project domain", http.StatusInternalServerError)
			return
		}
		projectDomains = append(projectDomains, pd)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projectDomains)
}
