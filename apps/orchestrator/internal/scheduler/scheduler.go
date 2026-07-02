package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/queue"
)

// Worker represents a worker in the scheduler
type Worker struct {
	ID            int64
	Name          string
	Address       string
	MaxConcurrent int
	CurrentLoad   int
	LastSeenAt    time.Time
}

// Scheduler manages worker selection and job assignment
type Scheduler struct {
	mu      sync.RWMutex
	workers map[int64]*Worker
	logger  *slog.Logger
}

// NewScheduler creates a new scheduler
func NewScheduler(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		workers: make(map[int64]*Worker),
		logger:  logger,
	}
}

// RegisterWorker registers a worker with the scheduler
func (s *Scheduler) RegisterWorker(worker *Worker) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.workers[worker.ID] = worker
	s.logger.Info("worker registered", "worker_id", worker.ID, "name", worker.Name)
}

// RemoveWorker removes a worker from the scheduler
func (s *Scheduler) RemoveWorker(workerID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.workers, workerID)
	s.logger.Info("worker removed", "worker_id", workerID)
}

// UpdateWorkerLoad updates the current load of a worker
func (s *Scheduler) UpdateWorkerLoad(workerID int64, load int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if worker, exists := s.workers[workerID]; exists {
		worker.CurrentLoad = load
		worker.LastSeenAt = time.Now()
	}
}

// SelectWorker selects the best worker for a job
func (s *Scheduler) SelectWorker() *Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bestWorker *Worker
	bestLoad := -1

	for _, worker := range s.workers {
		// Check if worker is available
		if worker.CurrentLoad >= worker.MaxConcurrent {
			continue
		}

		// Check if worker is recently seen (within 45 seconds)
		if time.Since(worker.LastSeenAt) > 45*time.Second {
			continue
		}

		// Select worker with lowest load
		if bestWorker == nil || worker.CurrentLoad > bestLoad {
			bestWorker = worker
			bestLoad = worker.CurrentLoad
		}
	}

	return bestWorker
}

// GetWorker returns a worker by ID
func (s *Scheduler) GetWorker(workerID int64) *Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.workers[workerID]
}

// GetAvailableWorkers returns all available workers
func (s *Scheduler) GetAvailableWorkers() []*Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var workers []*Worker
	for _, worker := range s.workers {
		if worker.CurrentLoad < worker.MaxConcurrent {
			workers = append(workers, worker)
		}
	}

	return workers
}

// GetWorkerCount returns the number of registered workers
func (s *Scheduler) GetWorkerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.workers)
}

// IncrementLoad increments the load of a worker
func (s *Scheduler) IncrementLoad(workerID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if worker, exists := s.workers[workerID]; exists {
		worker.CurrentLoad++
	}
}

// DecrementLoad decrements the load of a worker
func (s *Scheduler) DecrementLoad(workerID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if worker, exists := s.workers[workerID]; exists {
		if worker.CurrentLoad > 0 {
			worker.CurrentLoad--
		}
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context, queue Queue, pool Pool) {
	s.logger.Info("scheduler started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.processQueue(queue, pool)
		case <-queue.Notify():
			s.processQueue(queue, pool)
		}
	}
}

// processQueue processes jobs in the queue
func (s *Scheduler) processQueue(queue Queue, pool Pool) {
	for {
		job := queue.Dequeue()
		if job == nil {
			break
		}

		// Select a worker
		worker := pool.SelectBest()
		if worker == nil {
			s.logger.Warn("no worker available for job", "deployment_id", job.DeploymentID)
			// Put the job back in the queue
			queue.Enqueue(job)
			break
		}

		s.logger.Info("assigning job to worker",
			"deployment_id", job.DeploymentID,
			"worker_id", worker.ID,
			"worker_name", worker.Name,
		)

		// TODO: Send job to worker via gRPC
		// For now, just increment the worker load
		pool.IncrementLoad(worker.ID)
	}
}

// Queue represents a job queue interface
type Queue interface {
	Enqueue(job *queue.Job)
	Dequeue() *queue.Job
	Size() int
	IsEmpty() bool
	Notify() <-chan struct{}
}

// Pool represents a worker pool interface
type Pool interface {
	SelectBest() *Worker
	IncrementLoad(workerID int64)
	DecrementLoad(workerID int64)
}

// Job represents a job in the queue
type Job struct {
	DeploymentID int64
	ProjectID    int64
}
