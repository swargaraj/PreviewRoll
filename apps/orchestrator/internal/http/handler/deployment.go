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

// List godoc
// @Summary      List deployments
// @Description  Get list of deployments with optional filters
// @Tags         deployments
// @Produce      json
// @Param        project_id  query  string  false  "Filter by project ID"
// @Param        state       query  string  false  "Filter by state"
// @Param        limit       query  string  false  "Limit results"
// @Param        offset      query  string  false  "Offset results"
// @Success      200  {array}   deployment.Deployment
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/deployments [get]
// List returns a list of deployments
func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	state := r.URL.Query().Get("state")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

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

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to query deployments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

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

	if deployments == nil {
		deployments = []*deployment.Deployment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deployments)
}

// GetByID godoc
// @Summary      Get deployment by ID
// @Description  Returns a single deployment
// @Tags         deployments
// @Produce      json
// @Param        id   path  int  true  "Deployment ID"
// @Success      200  {object}  deployment.Deployment
// @Failure      400  {object}  string
// @Failure      404  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/deployments/{id} [get]
// GetByID returns a deployment by ID
func (h *DeploymentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	d, err := h.db.Q().GetDeploymentByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Deployment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

// Stop godoc
// @Summary      Stop a deployment
// @Description  Stop a running deployment
// @Tags         deployments
// @Produce      json
// @Param        id   path  int  true  "Deployment ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/deployments/{id}/stop [post]
// Stop stops a deployment
func (h *DeploymentHandler) Stop(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	err = h.db.Q().UpdateDeploymentForStop(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to stop deployment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Deployment stopped",
	})
}

// Restart godoc
// @Summary      Restart a deployment
// @Description  Restart a stopped or failed deployment
// @Tags         deployments
// @Produce      json
// @Param        id   path  int  true  "Deployment ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string
// @Failure      500  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/deployments/{id}/restart [post]
// Restart restarts a deployment
func (h *DeploymentHandler) Restart(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	err = h.db.Q().UpdateDeploymentForRestart(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to restart deployment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Deployment restarted",
	})
}

// Logs godoc
// @Summary      Get deployment logs
// @Description  Returns logs for a deployment (not yet implemented)
// @Tags         deployments
// @Produce      json
// @Param        id   path  int  true  "Deployment ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string
// @Security     CookieAuth
// @Router       /api/v1/deployments/{id}/logs [get]
// Logs returns the logs for a deployment
func (h *DeploymentHandler) Logs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	_, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Log streaming not yet implemented",
	})
}
