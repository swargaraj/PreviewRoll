package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/deployment"
)

// DeploymentHandler handles deployment requests
type DeploymentHandler struct {
	db *database.DB
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(db *database.DB) *DeploymentHandler {
	return &DeploymentHandler{db: db}
}

// List returns a list of deployments
func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	projectID := r.URL.Query().Get("project_id")
	state := r.URL.Query().Get("state")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Build query
	query := "SELECT id, project_id, worker_id, pr_number, commit_sha, branch, state, preview_url, container_id, build_log_path, error_message, attempt, created_at, updated_at, started_at, completed_at FROM deployments WHERE 1=1"
	args := []interface{}{}

	if projectID != "" {
		query += " AND project_id = ?"
		args = append(args, projectID)
	}

	if state != "" {
		query += " AND state = ?"
		args = append(args, state)
	}

	query += " ORDER BY created_at DESC"

	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil {
			query += " LIMIT ?"
			args = append(args, limit)
		}
	}

	if offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err == nil {
			query += " OFFSET ?"
			args = append(args, offset)
		}
	}

	// Execute query
	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to query deployments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Scan results
	var deployments []*deployment.Deployment
	for rows.Next() {
		d := &deployment.Deployment{}
		err := rows.Scan(
			&d.ID, &d.ProjectID, &d.WorkerID, &d.PRNumber, &d.CommitSHA,
			&d.Branch, &d.State, &d.PreviewURL, &d.ContainerID,
			&d.BuildLogPath, &d.ErrorMessage, &d.Attempt,
			&d.CreatedAt, &d.UpdatedAt, &d.StartedAt, &d.CompletedAt,
		)
		if err != nil {
			http.Error(w, "Failed to scan deployment", http.StatusInternalServerError)
			return
		}
		deployments = append(deployments, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}

// GetByID returns a deployment by ID
func (h *DeploymentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	d := &deployment.Deployment{}
	err = h.db.QueryRow(`
		SELECT id, project_id, worker_id, pr_number, commit_sha, branch, state, preview_url, container_id, build_log_path, error_message, attempt, created_at, updated_at, started_at, completed_at
		FROM deployments WHERE id = ?
	`, id).Scan(
		&d.ID, &d.ProjectID, &d.WorkerID, &d.PRNumber, &d.CommitSHA,
		&d.Branch, &d.State, &d.PreviewURL, &d.ContainerID,
		&d.BuildLogPath, &d.ErrorMessage, &d.Attempt,
		&d.CreatedAt, &d.UpdatedAt, &d.StartedAt, &d.CompletedAt,
	)
	if err != nil {
		http.Error(w, "Deployment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

// Stop stops a deployment
func (h *DeploymentHandler) Stop(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	// Update deployment state
	_, err = h.db.Exec(`
		UPDATE deployments SET state = 'stopped', updated_at = datetime('now'), completed_at = datetime('now')
		WHERE id = ? AND state NOT IN ('stopped', 'destroyed')
	`, id)
	if err != nil {
		http.Error(w, "Failed to stop deployment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Deployment stopped",
	})
}

// Restart restarts a deployment
func (h *DeploymentHandler) Restart(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	// Reset deployment state to queued
	_, err = h.db.Exec(`
		UPDATE deployments SET state = 'queued', worker_id = NULL, container_id = NULL, error_message = NULL, attempt = attempt + 1, updated_at = datetime('now')
		WHERE id = ? AND state IN ('failed', 'stopped')
	`, id)
	if err != nil {
		http.Error(w, "Failed to restart deployment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Deployment restarted",
	})
}

// Logs returns the logs for a deployment
func (h *DeploymentHandler) Logs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	_, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	// TODO: Implement SSE log streaming
	// For now, return placeholder
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Log streaming not yet implemented",
	})
}
