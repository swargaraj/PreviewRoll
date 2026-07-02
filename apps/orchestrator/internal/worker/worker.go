package worker

import (
	"time"
)

// State represents the worker state
type State string

const (
	StateOnline  State = "online"
	StateOffline State = "offline"
	StateBusy    State = "busy"
)

// Worker represents a worker instance
type Worker struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	State        State     `json:"state"`
	Capacity     int       `json:"capacity"`
	CurrentLoad  int       `json:"current_load"`
	Capabilities Capabilities `json:"capabilities"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Capabilities represents worker capabilities
type Capabilities struct {
	Platforms []string `json:"platforms"`
	Languages []string `json:"languages"`
	Features  []string `json:"features"`
}

// RegisterRequest represents the worker registration request
type RegisterRequest struct {
	Name         string       `json:"name"`
	Address      string       `json:"address"`
	Capacity     int          `json:"capacity"`
	Capabilities Capabilities `json:"capabilities"`
}

// RegisterResponse represents the worker registration response
type RegisterResponse struct {
	WorkerID int64  `json:"worker_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// HeartbeatRequest represents the worker heartbeat request
type HeartbeatRequest struct {
	WorkerID    int64 `json:"worker_id"`
	CurrentLoad int   `json:"current_load"`
}

// HeartbeatResponse represents the worker heartbeat response
type HeartbeatResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// DeregisterRequest represents the worker deregistration request
type DeregisterRequest struct {
	WorkerID int64  `json:"worker_id"`
	Reason   string `json:"reason"`
}

// NewWorker creates a new worker
func NewWorker(name, address string, capacity int, capabilities Capabilities) *Worker {
	now := time.Now()
	return &Worker{
		Name:         name,
		Address:      address,
		State:        StateOnline,
		Capacity:     capacity,
		CurrentLoad:  0,
		Capabilities: capabilities,
		LastSeenAt:   now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsAvailable checks if the worker can accept new jobs
func (w *Worker) IsAvailable() bool {
	return w.State == StateOnline && w.CurrentLoad < w.Capacity
}

// IsHealthy checks if the worker is healthy (seen within 45 seconds)
func (w *Worker) IsHealthy() bool {
	return time.Since(w.LastSeenAt) < 45*time.Second
}

// UpdateLoad updates the worker's load
func (w *Worker) UpdateLoad(load int) {
	w.CurrentLoad = load
	w.LastSeenAt = time.Now()
	w.UpdatedAt = time.Now()
}

// SetState sets the worker state
func (w *Worker) SetState(state State) {
	w.State = state
	w.UpdatedAt = time.Now()
}
