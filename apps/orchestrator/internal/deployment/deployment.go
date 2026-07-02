package deployment

import (
	"fmt"
	"time"
)

// Deployment represents a preview deployment
type Deployment struct {
	ID            int64     `json:"id"`
	ProjectID     int64     `json:"project_id"`
	WorkerID      *int64    `json:"worker_id,omitempty"`
	PRNumber      int       `json:"pr_number"`
	CommitSHA     string    `json:"commit_sha"`
	Branch        string    `json:"branch"`
	State         State     `json:"state"`
	PreviewURL    string    `json:"preview_url,omitempty"`
	ContainerID   string    `json:"container_id,omitempty"`
	BuildLogPath  string    `json:"build_log_path,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Attempt       int       `json:"attempt"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// CreateDeploymentRequest represents the request to create a deployment
type CreateDeploymentRequest struct {
	ProjectID  int64  `json:"project_id"`
	PRNumber   int    `json:"pr_number"`
	CommitSHA  string `json:"commit_sha"`
	Branch     string `json:"branch"`
}

// UpdateDeploymentStateRequest represents the request to update deployment state
type UpdateDeploymentStateRequest struct {
	State        State  `json:"state"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ListDeploymentsRequest represents the request to list deployments
type ListDeploymentsRequest struct {
	ProjectID *int64  `json:"project_id,omitempty"`
	WorkerID  *int64  `json:"worker_id,omitempty"`
	State     *State  `json:"state,omitempty"`
	Limit     int     `json:"limit,omitempty"`
	Offset    int     `json:"offset,omitempty"`
}

// DeploymentFilter represents filters for listing deployments
type DeploymentFilter struct {
	ProjectID *int64
	WorkerID  *int64
	State     *State
	Limit     int
	Offset    int
}

// NewDeployment creates a new deployment
func NewDeployment(projectID int64, prNumber int, commitSHA, branch string) *Deployment {
	now := time.Now()
	return &Deployment{
		ProjectID: projectID,
		PRNumber:  prNumber,
		CommitSHA: commitSHA,
		Branch:    branch,
		State:     StateQueued,
		Attempt:   0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TransitionTo transitions the deployment to a new state
func (d *Deployment) TransitionTo(newState State, reason string) error {
	if err := ValidateTransition(d.State, newState); err != nil {
		return err
	}

	d.State = newState
	d.UpdatedAt = time.Now()

	// Set timestamps based on state
	if newState == StateBuilding && d.StartedAt == nil {
		now := time.Now()
		d.StartedAt = &now
	}

	if newState == StateRunning || newState == StateFailed || newState == StateStopped || newState == StateDestroyed {
		now := time.Now()
		d.CompletedAt = &now
	}

	return nil
}

// CanRetry checks if the deployment can be retried
func (d *Deployment) CanRetry(maxAttempts int) bool {
	return IsRetryable(d.State) && d.Attempt < maxAttempts
}

// Retry retries the deployment
func (d *Deployment) Retry() error {
	if !d.CanRetry(3) {
		return fmt.Errorf("deployment cannot be retried")
	}

	d.State = StateQueued
	d.Attempt++
	d.WorkerID = nil
	d.ContainerID = ""
	d.ErrorMessage = ""
	d.UpdatedAt = time.Now()

	return nil
}
