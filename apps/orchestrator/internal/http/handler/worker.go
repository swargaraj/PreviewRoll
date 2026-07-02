package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/worker"
)

// WorkerHandler handles worker requests
type WorkerHandler struct {
	db   *database.DB
	pool *worker.Pool
}

// NewWorkerHandler creates a new worker handler
func NewWorkerHandler(db *database.DB, pool *worker.Pool) *WorkerHandler {
	return &WorkerHandler{db: db, pool: pool}
}

// List returns a list of workers
func (h *WorkerHandler) List(w http.ResponseWriter, r *http.Request) {
	workers := h.pool.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

// GetByID returns a worker by ID
func (h *WorkerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid worker ID", http.StatusBadRequest)
		return
	}

	worker := h.pool.Get(id)
	if worker == nil {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(worker)
}

// Register registers a new worker
func (h *WorkerHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req worker.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" || req.Address == "" {
		http.Error(w, "Name and address are required", http.StatusBadRequest)
		return
	}

	if req.Capacity <= 0 {
		req.Capacity = 5
	}

	// Create worker
	wrk := worker.NewWorker(req.Name, req.Address, req.Capacity, req.Capabilities)

	// Insert into database
	err := h.db.QueryRow(`
		INSERT INTO workers (name, address, state, capacity, current_load, capabilities)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id
	`, wrk.Name, wrk.Address, wrk.State, wrk.Capacity, wrk.CurrentLoad, "{}").Scan(&wrk.ID)
	if err != nil {
		http.Error(w, "Failed to register worker", http.StatusInternalServerError)
		return
	}

	// Register with pool
	h.pool.Register(wrk)

	// Return response
	resp := worker.RegisterResponse{
		WorkerID: wrk.ID,
		Status:   "success",
		Message:  "Worker registered successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Heartbeat handles worker heartbeat
func (h *WorkerHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req worker.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update worker in database
	_, err := h.db.Exec(`
		UPDATE workers SET current_load = ?, last_seen_at = datetime('now'), updated_at = datetime('now')
		WHERE id = ?
	`, req.CurrentLoad, req.WorkerID)
	if err != nil {
		http.Error(w, "Failed to update worker", http.StatusInternalServerError)
		return
	}

	// Update pool
	h.pool.UpdateLoad(req.WorkerID, req.CurrentLoad)

	// Return response
	resp := worker.HeartbeatResponse{
		Status:    "ok",
		Timestamp: 0, // TODO: Use actual timestamp
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Deregister deregisters a worker
func (h *WorkerHandler) Deregister(w http.ResponseWriter, r *http.Request) {
	var req worker.DeregisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update worker state in database
	_, err := h.db.Exec(`
		UPDATE workers SET state = 'offline', updated_at = datetime('now')
		WHERE id = ?
	`, req.WorkerID)
	if err != nil {
		http.Error(w, "Failed to deregister worker", http.StatusInternalServerError)
		return
	}

	// Remove from pool
	h.pool.Deregister(req.WorkerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Worker deregistered",
	})
}
