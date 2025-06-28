package daemon

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobQueue implements QueueInterface with in-memory storage
type JobQueue struct {
	// mutex for thread-safe operations
	mu sync.RWMutex

	// jobs stores all jobs by ID
	jobs map[string]*Job

	// pending queue for jobs waiting to be processed
	pending []*Job

	// running jobs currently being processed
	running map[string]*Job

	// completed jobs (for history)
	completed []*Job

	// failed jobs (for history)
	failed []*Job

	// cancelled jobs (for history)
	cancelled []*Job

	// configuration
	maxQueueSize int
	maxHistory   int

	// channels for blocking operations
	newJobCh chan struct{}

	// statistics
	stats struct {
		totalAdded     int
		totalCompleted int
		totalFailed    int
		totalCancelled int
	}
}

// NewJobQueue creates a new job queue with the specified configuration
func NewJobQueue(maxQueueSize, maxHistory int) *JobQueue {
	if maxQueueSize <= 0 {
		maxQueueSize = DefaultQueueSize
	}
	if maxHistory <= 0 {
		maxHistory = 1000 // Default to keep 1000 completed jobs
	}

	return &JobQueue{
		jobs:         make(map[string]*Job),
		pending:      make([]*Job, 0),
		running:      make(map[string]*Job),
		completed:    make([]*Job, 0),
		failed:       make([]*Job, 0),
		cancelled:    make([]*Job, 0),
		maxQueueSize: maxQueueSize,
		maxHistory:   maxHistory,
		newJobCh:     make(chan struct{}, 1),
	}
}

// Add adds a job to the queue
func (q *JobQueue) Add(job *Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if job already exists
	if _, exists := q.jobs[job.ID]; exists {
		return fmt.Errorf("job with ID %s already exists", job.ID)
	}

	// Check queue size limit
	if len(q.pending) >= q.maxQueueSize {
		return fmt.Errorf("queue is full (max size: %d)", q.maxQueueSize)
	}

	// Check for duplicate paths in pending jobs
	for _, pendingJob := range q.pending {
		if pendingJob.Path == job.Path {
			return fmt.Errorf("job for path %s is already pending", job.Path)
		}
	}

	// Check for duplicate paths in running jobs
	for _, runningJob := range q.running {
		if runningJob.Path == job.Path {
			return fmt.Errorf("job for path %s is already running", job.Path)
		}
	}

	// Set job status and add to queue
	job.Status = JobStatusPending
	job.CreatedAt = time.Now()

	q.jobs[job.ID] = job
	q.pending = append(q.pending, job)
	q.stats.totalAdded++

	// Notify waiting workers
	select {
	case q.newJobCh <- struct{}{}:
	default:
		// Channel is full, that's ok
	}

	return nil
}

// Next gets the next job from the queue (blocking)
func (q *JobQueue) Next(ctx context.Context) (*Job, error) {
	for {
		q.mu.Lock()

		// Check if there are pending jobs
		if len(q.pending) > 0 {
			// Get the first job (FIFO)
			job := q.pending[0]
			q.pending = q.pending[1:]

			// Move to running
			job.Status = JobStatusRunning
			now := time.Now()
			job.StartedAt = &now
			q.running[job.ID] = job

			q.mu.Unlock()
			return job, nil
		}

		q.mu.Unlock()

		// Wait for new jobs or context cancellation
		select {
		case <-q.newJobCh:
			// Try again
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Get retrieves a job by ID
func (q *JobQueue) Get(jobID string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, exists := q.jobs[jobID]
	if !exists {
		return nil, ErrJobNotFound
	}

	// Return a copy to prevent external modification
	jobCopy := *job
	return &jobCopy, nil
}

// Update updates a job's status and data
func (q *JobQueue) Update(job *Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	existingJob, exists := q.jobs[job.ID]
	if !exists {
		return ErrJobNotFound
	}

	// Update the job in place
	existingJob.Status = job.Status
	existingJob.Stats = job.Stats
	existingJob.Error = job.Error
	existingJob.Progress = job.Progress

	// Handle status transitions
	switch job.Status {
	case JobStatusCompleted:
		if existingJob.CompletedAt == nil {
			now := time.Now()
			existingJob.CompletedAt = &now
		}

		// Move from running to completed
		delete(q.running, job.ID)
		q.completed = append(q.completed, existingJob)
		q.stats.totalCompleted++

		// Cleanup old completed jobs
		q.cleanupHistory(&q.completed)

	case JobStatusFailed:
		if existingJob.CompletedAt == nil {
			now := time.Now()
			existingJob.CompletedAt = &now
		}

		// Move from running to failed
		delete(q.running, job.ID)
		q.failed = append(q.failed, existingJob)
		q.stats.totalFailed++

		// Cleanup old failed jobs
		q.cleanupHistory(&q.failed)

	case JobStatusCancelled:
		if existingJob.CompletedAt == nil {
			now := time.Now()
			existingJob.CompletedAt = &now
		}

		// Remove from running or pending
		delete(q.running, job.ID)
		q.removeFromPending(job.ID)
		q.cancelled = append(q.cancelled, existingJob)
		q.stats.totalCancelled++

		// Cleanup old cancelled jobs
		q.cleanupHistory(&q.cancelled)
	}

	return nil
}

// List returns jobs with optional status filter
func (q *JobQueue) List(status JobStatus) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*Job

	switch status {
	case JobStatusPending:
		result = make([]*Job, len(q.pending))
		for i, job := range q.pending {
			jobCopy := *job
			result[i] = &jobCopy
		}
	case JobStatusRunning:
		result = make([]*Job, 0, len(q.running))
		for _, job := range q.running {
			jobCopy := *job
			result = append(result, &jobCopy)
		}
	case JobStatusCompleted:
		result = make([]*Job, len(q.completed))
		for i, job := range q.completed {
			jobCopy := *job
			result[i] = &jobCopy
		}
	case JobStatusFailed:
		result = make([]*Job, len(q.failed))
		for i, job := range q.failed {
			jobCopy := *job
			result[i] = &jobCopy
		}
	case JobStatusCancelled:
		result = make([]*Job, len(q.cancelled))
		for i, job := range q.cancelled {
			jobCopy := *job
			result[i] = &jobCopy
		}
	default:
		// Return all jobs
		result = make([]*Job, 0, len(q.jobs))
		for _, job := range q.jobs {
			jobCopy := *job
			result = append(result, &jobCopy)
		}
	}

	return result, nil
}

// Cancel marks a job as cancelled
func (q *JobQueue) Cancel(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.jobs[jobID]
	if !exists {
		return ErrJobNotFound
	}

	// Can only cancel pending or running jobs
	if job.Status != JobStatusPending && job.Status != JobStatusRunning {
		return fmt.Errorf("job %s cannot be cancelled (status: %s)", jobID, job.Status)
	}

	// Update job status
	job.Status = JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	// Remove from pending or running
	q.removeFromPending(jobID)
	delete(q.running, jobID)

	// Add to cancelled
	q.cancelled = append(q.cancelled, job)
	q.stats.totalCancelled++

	// Cleanup old cancelled jobs
	q.cleanupHistory(&q.cancelled)

	return nil
}

// Clear removes all pending jobs
func (q *JobQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove pending jobs from jobs map
	for _, job := range q.pending {
		delete(q.jobs, job.ID)
	}

	// Clear pending queue
	q.pending = make([]*Job, 0)

	return nil
}

// Size returns the current queue size
func (q *JobQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.pending)
}

// Stats returns queue statistics
func (q *JobQueue) Stats() map[string]int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return map[string]int{
		"pending":         len(q.pending),
		"running":         len(q.running),
		"completed":       len(q.completed),
		"failed":          len(q.failed),
		"cancelled":       len(q.cancelled),
		"total_added":     q.stats.totalAdded,
		"total_completed": q.stats.totalCompleted,
		"total_failed":    q.stats.totalFailed,
		"total_cancelled": q.stats.totalCancelled,
	}
}

// Helper methods

// removeFromPending removes a job from the pending queue
func (q *JobQueue) removeFromPending(jobID string) {
	for i, job := range q.pending {
		if job.ID == jobID {
			q.pending = append(q.pending[:i], q.pending[i+1:]...)
			break
		}
	}
}

// cleanupHistory removes old jobs from history to prevent memory leaks
func (q *JobQueue) cleanupHistory(history *[]*Job) {
	if len(*history) > q.maxHistory {
		// Keep only the most recent jobs
		*history = (*history)[len(*history)-q.maxHistory:]
	}
}
