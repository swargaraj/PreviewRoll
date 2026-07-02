package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/domain"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/queue"
)

// WebhookHandler handles GitHub webhook requests
type WebhookHandler struct {
	db    *database.DB
	queue *queue.MemoryQueue
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(db *database.DB, queue *queue.MemoryQueue) *WebhookHandler {
	return &WebhookHandler{db: db, queue: queue}
}

// WebhookPayload represents the GitHub webhook payload
type WebhookPayload struct {
	Action      string `json:"action"`
	PullRequest *struct {
		Number int `json:"number"`
		Head   struct {
			Sha string `json:"sha"`
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
	Repository *struct {
		FullName string `json:"full_name"`
		ID       int64  `json:"id"`
	} `json:"repository"`
}

// HandleGitHubWebhook handles GitHub webhook requests
func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get signature from header
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	// Verify signature
	if !h.verifySignature(body, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Handle ping event
	if r.Header.Get("X-GitHub-Event") == "ping" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		return
	}

	// Handle pull request events
	if r.Header.Get("X-GitHub-Event") == "pull_request" {
		h.handlePullRequest(w, payload)
		return
	}

	// Unknown event type
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handlePullRequest(w http.ResponseWriter, payload WebhookPayload) {
	if payload.PullRequest == nil || payload.Repository == nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	slog.Info("received pull request event",
		"action", payload.Action,
		"pr_number", payload.PullRequest.Number,
		"repo", payload.Repository.FullName,
		"sha", payload.PullRequest.Head.Sha,
		"branch", payload.PullRequest.Head.Ref,
	)

	switch payload.Action {
	case "opened", "synchronize", "reopened":
		// Create or update deployment
		h.createDeployment(payload)
	case "closed":
		// Stop deployment
		h.stopDeployment(payload)
	default:
		slog.Info("ignoring pull request action", "action", payload.Action)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func (h *WebhookHandler) createDeployment(payload WebhookPayload) {
	// Find project by repository
	var projectID int64
	err := h.db.QueryRow(`
		SELECT id FROM projects WHERE github_repo = ?
	`, payload.Repository.FullName).Scan(&projectID)
	if err != nil {
		slog.Error("project not found", "repo", payload.Repository.FullName)
		return
	}

	// Look up connected domain for preview URL
	previewURL := h.resolvePreviewURL(projectID, payload.PullRequest.Head.Sha)

	// Check if deployment already exists for this PR
	var existingID int64
	err = h.db.QueryRow(`
		SELECT id FROM deployments WHERE project_id = ? AND pr_number = ?
	`, projectID, payload.PullRequest.Number).Scan(&existingID)
	if err == nil {
		// Update existing deployment
		_, err = h.db.Exec(`
			UPDATE deployments SET commit_sha = ?, branch = ?, state = 'queued', preview_url = ?, updated_at = datetime('now')
			WHERE id = ?
		`, payload.PullRequest.Head.Sha, payload.PullRequest.Head.Ref, previewURL, existingID)
		if err != nil {
			slog.Error("failed to update deployment", "error", err)
			return
		}
		slog.Info("deployment updated", "deployment_id", existingID, "preview_url", previewURL)
	} else {
		// Create new deployment
		var deploymentID int64
		err = h.db.QueryRow(`
			INSERT INTO deployments (project_id, pr_number, commit_sha, branch, state, preview_url)
			VALUES (?, ?, ?, ?, 'queued', ?)
			RETURNING id
		`, projectID, payload.PullRequest.Number, payload.PullRequest.Head.Sha, payload.PullRequest.Head.Ref, previewURL).Scan(&deploymentID)
		if err != nil {
			slog.Error("failed to create deployment", "error", err)
			return
		}
		slog.Info("deployment created", "deployment_id", deploymentID, "preview_url", previewURL)

		// Add to queue
		h.queue.Enqueue(&queue.Job{
			DeploymentID: deploymentID,
			ProjectID:    projectID,
		})
	}
}

// resolvePreviewURL looks up the project's connected domain and generates a preview URL
func (h *WebhookHandler) resolvePreviewURL(projectID int64, commitSHA string) string {
	var prefix, domainName string
	err := h.db.QueryRow(`
		SELECT pd.prefix, d.name
		FROM project_domains pd
		INNER JOIN domains d ON pd.domain_id = d.id
		WHERE pd.project_id = ?
		LIMIT 1
	`, projectID).Scan(&prefix, &domainName)
	if err != nil {
		slog.Debug("no domain connected to project, skipping preview URL", "project_id", projectID)
		return ""
	}

	return domain.GeneratePreviewURL(prefix, domainName, commitSHA)
}

func (h *WebhookHandler) stopDeployment(payload WebhookPayload) {
	// Find project by repository
	var projectID int64
	err := h.db.QueryRow(`
		SELECT id FROM projects WHERE github_repo = ?
	`, payload.Repository.FullName).Scan(&projectID)
	if err != nil {
		slog.Error("project not found", "repo", payload.Repository.FullName)
		return
	}

	// Stop deployment for this PR
	_, err = h.db.Exec(`
		UPDATE deployments SET state = 'stopped', updated_at = datetime('now'), completed_at = datetime('now')
		WHERE project_id = ? AND pr_number = ? AND state NOT IN ('stopped', 'destroyed')
	`, projectID, payload.PullRequest.Number)
	if err != nil {
		slog.Error("failed to stop deployment", "error", err)
		return
	}

	slog.Info("deployment stopped",
		"project_id", projectID,
		"pr_number", payload.PullRequest.Number,
	)
}

func (h *WebhookHandler) verifySignature(payload []byte, signature string) bool {
	// Get webhook secret from database
	// For now, use a placeholder
	secret := "placeholder-secret"

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
