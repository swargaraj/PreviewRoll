package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/scheduler"
)

// Pool manages a pool of workers
type Pool struct {
	mu      sync.RWMutex
	workers map[int64]*Worker
	logger  *slog.Logger
}

// NewPool creates a new worker pool
func NewPool(logger *slog.Logger) *Pool {
	return &Pool{
		workers: make(map[int64]*Worker),
		logger:  logger,
	}
}

// Register registers a worker in the pool
func (p *Pool) Register(worker *Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.workers[worker.ID] = worker
	p.logger.Info("worker registered", "worker_id", worker.ID, "name", worker.Name)
}

// Deregister removes a worker from the pool
func (p *Pool) Deregister(workerID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.workers, workerID)
	p.logger.Info("worker deregistered", "worker_id", workerID)
}

// Get returns a worker by ID
func (p *Pool) Get(workerID int64) *Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.workers[workerID]
}

// GetAll returns all workers
func (p *Pool) GetAll() []*Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workers := make([]*Worker, 0, len(p.workers))
	for _, w := range p.workers {
		workers = append(workers, w)
	}

	return workers
}

// GetOnline returns all online workers
func (p *Pool) GetOnline() []*Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var workers []*Worker
	for _, w := range p.workers {
		if w.State == StateOnline {
			workers = append(workers, w)
		}
	}

	return workers
}

// GetAvailable returns all available workers
func (p *Pool) GetAvailable() []*Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var workers []*Worker
	for _, w := range p.workers {
		if w.IsAvailable() {
			workers = append(workers, w)
		}
	}

	return workers
}

// SelectBest selects the best worker for a job
func (p *Pool) SelectBest() *scheduler.Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var best *Worker
	bestLoad := -1

	for _, w := range p.workers {
		if !w.IsAvailable() {
			continue
		}

		if best == nil || w.CurrentLoad < bestLoad {
			best = w
			bestLoad = w.CurrentLoad
		}
	}

	if best == nil {
		return nil
	}

	return &scheduler.Worker{
		ID:            best.ID,
		Name:          best.Name,
		Address:       best.Address,
		MaxConcurrent: best.Capacity,
		CurrentLoad:   best.CurrentLoad,
		LastSeenAt:    best.LastSeenAt,
	}
}

// IncrementLoad increments the load of a worker
func (p *Pool) IncrementLoad(workerID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if w, exists := p.workers[workerID]; exists {
		w.CurrentLoad++
		w.LastSeenAt = time.Now()
	}
}

// DecrementLoad decrements the load of a worker
func (p *Pool) DecrementLoad(workerID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if w, exists := p.workers[workerID]; exists {
		if w.CurrentLoad > 0 {
			w.CurrentLoad--
		}
		w.LastSeenAt = time.Now()
	}
}

// UpdateLoad updates the load of a worker
func (p *Pool) UpdateLoad(workerID int64, load int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if w, exists := p.workers[workerID]; exists {
		w.UpdateLoad(load)
	}
}

// CheckHealth checks the health of all workers
func (p *Pool) CheckHealth() []int64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	var unhealthy []int64

	for _, w := range p.workers {
		if !w.IsHealthy() {
			p.logger.Warn("worker unhealthy", "worker_id", w.ID, "last_seen", w.LastSeenAt)
			unhealthy = append(unhealthy, w.ID)
		}
	}

	return unhealthy
}

// MarkOffline marks workers as offline
func (p *Pool) MarkOffline(workerIDs []int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, id := range workerIDs {
		if w, exists := p.workers[id]; exists {
			w.SetState(StateOffline)
			p.logger.Info("worker marked offline", "worker_id", id)
		}
	}
}

// Count returns the number of workers
func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.workers)
}

// StartHealthCheck starts a background health check goroutine
func (p *Pool) StartHealthCheck(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				unhealthy := p.CheckHealth()
				if len(unhealthy) > 0 {
					p.MarkOffline(unhealthy)
					// TODO: Trigger recovery for deployments on unhealthy workers
				}
			}
		}
	}()
}
