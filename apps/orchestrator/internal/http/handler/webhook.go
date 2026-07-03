package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database/sqlc"
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

// HandleGitHubWebhook godoc
// @Summary      GitHub webhook
// @Description  Handle incoming GitHub webhook events (push, pull_request, ping)
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        X-Hub-Signature-256  header  string  true  "HMAC-SHA256 signature"
// @Param        X-GitHub-Event       header  string  true  "Event type"
// @Param        body                 body    WebhookPayload  true  "GitHub webhook payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  string
// @Failure      401  {object}  string
// @Router       /webhook/github [post]
// HandleGitHubWebhook handles GitHub webhook requests
func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	if !h.verifySignature(body, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if r.Header.Get("X-GitHub-Event") == "ping" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
		return
	}

	if r.Header.Get("X-GitHub-Event") == "pull_request" {
		h.handlePullRequest(w, payload)
		return
	}

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
		h.createDeployment(payload)
	case "closed":
		h.stopDeployment(payload)
	default:
		slog.Info("ignoring pull request action", "action", payload.Action)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func (h *WebhookHandler) createDeployment(payload WebhookPayload) {
	ctx := context.Background()

	projectID, err := h.db.Q().GetProjectByRepoURL(ctx, "https://github.com/"+payload.Repository.FullName)
	if err != nil {
		slog.Error("project not found", "repo", payload.Repository.FullName)
		return
	}

	previewURL := h.resolvePreviewURL(projectID, payload.PullRequest.Head.Sha)

	existing, err := h.db.Q().GetDeploymentByProjectAndPR(ctx, sqlc.GetDeploymentByProjectAndPRParams{
		ProjectID: projectID,
		PrNumber:  int64(payload.PullRequest.Number),
	})
	if err == nil {
		err = h.db.Q().UpdateDeploymentFromWebhook(ctx, sqlc.UpdateDeploymentFromWebhookParams{
			CommitSha: payload.PullRequest.Head.Sha,
			Branch:    payload.PullRequest.Head.Ref,
			PreviewUrl: sql.NullString{
				String: previewURL,
				Valid:  previewURL != "",
			},
			ID: existing,
		})
		if err != nil {
			slog.Error("failed to update deployment", "error", err)
			return
		}
		slog.Info("deployment updated", "deployment_id", existing, "preview_url", previewURL)
	} else {
		deploymentID, err := h.db.Q().CreateDeploymentFromWebhook(ctx, sqlc.CreateDeploymentFromWebhookParams{
			ProjectID: projectID,
			PrNumber:  int64(payload.PullRequest.Number),
			CommitSha: payload.PullRequest.Head.Sha,
			Branch:    payload.PullRequest.Head.Ref,
			PreviewUrl: sql.NullString{
				String: previewURL,
				Valid:  previewURL != "",
			},
		})
		if err != nil {
			slog.Error("failed to create deployment", "error", err)
			return
		}
		slog.Info("deployment created", "deployment_id", deploymentID, "preview_url", previewURL)

		h.queue.Enqueue(&queue.Job{
			DeploymentID: deploymentID,
			ProjectID:    projectID,
		})
	}
}

func (h *WebhookHandler) resolvePreviewURL(projectID int64, commitSHA string) string {
	pd, err := h.db.Q().GetProjectDomainByProjectID(context.Background(), projectID)
	if err != nil {
		slog.Debug("no domain connected to project, skipping preview URL", "project_id", projectID)
		return ""
	}

	return domain.GeneratePreviewURL(pd.Prefix, pd.Name, commitSHA)
}

func (h *WebhookHandler) stopDeployment(payload WebhookPayload) {
	ctx := context.Background()

	projectID, err := h.db.Q().GetProjectByRepoURL(ctx, "https://github.com/"+payload.Repository.FullName)
	if err != nil {
		slog.Error("project not found", "repo", payload.Repository.FullName)
		return
	}

	err = h.db.Q().StopDeploymentsByProjectAndPR(ctx, sqlc.StopDeploymentsByProjectAndPRParams{
		ProjectID: projectID,
		PrNumber:  int64(payload.PullRequest.Number),
	})
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
	secret := "placeholder-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
