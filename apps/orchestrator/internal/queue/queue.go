package queue

import (
	"sync"
)

// Job represents a job in the queue
type Job struct {
	DeploymentID int64
	ProjectID    int64
}

// Queue represents a job queue interface
type Queue interface {
	Enqueue(job *Job)
	Dequeue() *Job
	Size() int
	IsEmpty() bool
	Notify() <-chan struct{}
}

// MemoryQueue is an in-memory implementation of the Queue interface
type MemoryQueue struct {
	mu     sync.RWMutex
	jobs   []*Job
	notify chan struct{}
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		jobs:   make([]*Job, 0),
		notify: make(chan struct{}, 1),
	}
}

// Enqueue adds a job to the queue
func (q *MemoryQueue) Enqueue(job *Job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs = append(q.jobs, job)

	// Signal that there's a new job
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

// Dequeue removes and returns the next job from the queue
func (q *MemoryQueue) Dequeue() *Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.jobs) == 0 {
		return nil
	}

	job := q.jobs[0]
	q.jobs[0] = nil
	q.jobs = q.jobs[1:]

	return job
}

// Size returns the number of jobs in the queue
func (q *MemoryQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.jobs)
}

// IsEmpty checks if the queue is empty
func (q *MemoryQueue) IsEmpty() bool {
	return q.Size() == 0
}

// Notify returns a channel that receives when there are new jobs
func (q *MemoryQueue) Notify() <-chan struct{} {
	return q.notify
}
