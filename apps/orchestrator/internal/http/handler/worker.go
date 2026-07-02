package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database/sqlc"
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

	if req.Name == "" || req.Address == "" {
		http.Error(w, "Name and address are required", http.StatusBadRequest)
		return
	}

	if req.Capacity <= 0 {
		req.Capacity = 5
	}

	wrk := worker.NewWorker(req.Name, req.Address, req.Capacity, req.Capabilities)

	created, err := h.db.Q().CreateWorker(r.Context(), sqlc.CreateWorkerParams{
		Name:         wrk.Name,
		Address:      wrk.Address,
		State:        string(wrk.State),
		Capacity:     int64(wrk.Capacity),
		CurrentLoad:  int64(wrk.CurrentLoad),
		Capabilities: "{}",
	})
	if err != nil {
		http.Error(w, "Failed to register worker", http.StatusInternalServerError)
		return
	}

	wrk.ID = created.ID
	h.pool.Register(wrk)

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

	err := h.db.Q().UpdateWorkerLoad(r.Context(), sqlc.UpdateWorkerLoadParams{
		CurrentLoad: int64(req.CurrentLoad),
		ID:          req.WorkerID,
	})
	if err != nil {
		http.Error(w, "Failed to update worker", http.StatusInternalServerError)
		return
	}

	h.pool.UpdateLoad(req.WorkerID, req.CurrentLoad)

	resp := worker.HeartbeatResponse{
		Status:    "ok",
		Timestamp: 0,
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

	err := h.db.Q().UpdateWorkerState(r.Context(), sqlc.UpdateWorkerStateParams{
		State: "offline",
		ID:    req.WorkerID,
	})
	if err != nil {
		http.Error(w, "Failed to deregister worker", http.StatusInternalServerError)
		return
	}

	h.pool.Deregister(req.WorkerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Worker deregistered",
	})
}
