package daemon

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJobQueue(t *testing.T) {
	t.Run("with valid parameters", func(t *testing.T) {
		queue := NewJobQueue(100, 500)

		assert.NotNil(t, queue)
		assert.Equal(t, 100, queue.maxQueueSize)
		assert.Equal(t, 500, queue.maxHistory)
		assert.NotNil(t, queue.jobs)
		assert.NotNil(t, queue.pending)
		assert.NotNil(t, queue.running)
		assert.NotNil(t, queue.completed)
		assert.NotNil(t, queue.failed)
		assert.NotNil(t, queue.cancelled)
		assert.NotNil(t, queue.newJobCh)
	})

	t.Run("with zero parameters", func(t *testing.T) {
		queue := NewJobQueue(0, 0)

		assert.Equal(t, DefaultQueueSize, queue.maxQueueSize)
		assert.Equal(t, 1000, queue.maxHistory)
	})

	t.Run("with negative parameters", func(t *testing.T) {
		queue := NewJobQueue(-10, -5)

		assert.Equal(t, DefaultQueueSize, queue.maxQueueSize)
		assert.Equal(t, 1000, queue.maxHistory)
	})
}

func TestJobQueue_Add(t *testing.T) {
	queue := NewJobQueue(3, 10)

	t.Run("add valid job", func(t *testing.T) {
		job := NewJob("/test/path", &scanner.ScanOptions{Verbose: true})

		err := queue.Add(job)
		assert.NoError(t, err)
		assert.Equal(t, JobStatusPending, job.Status)
		assert.Equal(t, 1, queue.Size())

		stats := queue.Stats()
		assert.Equal(t, 1, stats["pending"])
		assert.Equal(t, 1, stats["total_added"])
	})

	t.Run("add nil job", func(t *testing.T) {
		err := queue.Add(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job cannot be nil")
	})

	t.Run("add duplicate job ID", func(t *testing.T) {
		job1 := NewJob("/test/path1", nil)
		job2 := &Job{
			ID:   job1.ID, // Same ID
			Path: "/test/path2",
		}

		err := queue.Add(job1)
		require.NoError(t, err)

		err = queue.Add(job2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("add duplicate path - pending", func(t *testing.T) {
		queue := NewJobQueue(10, 10)

		job1 := NewJob("/duplicate/path", nil)
		job2 := &Job{
			ID:   "different-id-1",
			Path: "/duplicate/path", // Same path, different ID
		}

		err := queue.Add(job1)
		require.NoError(t, err)

		err = queue.Add(job2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already pending")
	})

	t.Run("add duplicate path - running", func(t *testing.T) {
		queue := NewJobQueue(10, 10)

		job1 := NewJob("/running/path", nil)
		err := queue.Add(job1)
		require.NoError(t, err)

		// Move job to running state
		ctx := context.Background()
		runningJob, err := queue.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, job1.ID, runningJob.ID)

		// Try to add same path
		job2 := &Job{
			ID:   "different-id-2",
			Path: "/running/path", // Same path, different ID
		}
		err = queue.Add(job2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")
	})

	t.Run("queue size limit", func(t *testing.T) {
		smallQueue := NewJobQueue(2, 10)

		// Add jobs up to limit
		job1 := &Job{ID: "job1", Path: "/path1"}
		job2 := &Job{ID: "job2", Path: "/path2"}
		job3 := &Job{ID: "job3", Path: "/path3"}

		err := smallQueue.Add(job1)
		assert.NoError(t, err)

		err = smallQueue.Add(job2)
		assert.NoError(t, err)

		// This should fail due to size limit
		err = smallQueue.Add(job3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "queue is full")
	})
}

func TestJobQueue_Next(t *testing.T) {
	queue := NewJobQueue(10, 10)

	t.Run("get next job from empty queue with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := queue.Next(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("get next job from queue", func(t *testing.T) {
		job1 := NewJob("/path1", nil)
		job2 := NewJob("/path2", nil)

		err := queue.Add(job1)
		require.NoError(t, err)
		err = queue.Add(job2)
		require.NoError(t, err)

		ctx := context.Background()

		// Get first job (FIFO)
		nextJob, err := queue.Next(ctx)
		assert.NoError(t, err)
		assert.Equal(t, job1.ID, nextJob.ID)
		assert.Equal(t, JobStatusRunning, nextJob.Status)
		assert.NotNil(t, nextJob.StartedAt)

		// Get second job
		nextJob, err = queue.Next(ctx)
		assert.NoError(t, err)
		assert.Equal(t, job2.ID, nextJob.ID)
		assert.Equal(t, JobStatusRunning, nextJob.Status)

		// Queue should be empty now
		assert.Equal(t, 0, queue.Size())

		stats := queue.Stats()
		assert.Equal(t, 0, stats["pending"])
		assert.Equal(t, 2, stats["running"])
	})

	t.Run("concurrent Next calls", func(t *testing.T) {
		queue := NewJobQueue(10, 10)

		// Add multiple jobs with unique IDs
		for i := 0; i < 5; i++ {
			job := &Job{
				ID:   fmt.Sprintf("concurrent-next-%d", i),
				Path: fmt.Sprintf("/path%d", i),
			}
			err := queue.Add(job)
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		results := make([]*Job, 5)

		// Start multiple goroutines to get jobs
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				ctx := context.Background()
				job, err := queue.Next(ctx)
				assert.NoError(t, err)
				results[index] = job
			}(i)
		}

		wg.Wait()

		// All jobs should be retrieved
		jobIDs := make(map[string]bool)
		for _, job := range results {
			assert.NotNil(t, job)
			assert.Equal(t, JobStatusRunning, job.Status)
			jobIDs[job.ID] = true
		}

		// Should have 5 unique jobs
		assert.Equal(t, 5, len(jobIDs))
		assert.Equal(t, 0, queue.Size())
	})
}

func TestJobQueue_Get(t *testing.T) {
	queue := NewJobQueue(10, 10)

	t.Run("get existing job", func(t *testing.T) {
		originalJob := NewJob("/test/path", &scanner.ScanOptions{Verbose: true})
		err := queue.Add(originalJob)
		require.NoError(t, err)

		retrievedJob, err := queue.Get(originalJob.ID)
		assert.NoError(t, err)
		assert.Equal(t, originalJob.ID, retrievedJob.ID)
		assert.Equal(t, originalJob.Path, retrievedJob.Path)
		assert.Equal(t, originalJob.Status, retrievedJob.Status)

		// Should be a copy, not the same instance
		assert.NotSame(t, originalJob, retrievedJob)
	})

	t.Run("get non-existent job", func(t *testing.T) {
		_, err := queue.Get("non-existent-id")
		assert.Error(t, err)
		assert.Equal(t, ErrJobNotFound, err)
	})
}

func TestJobQueue_Update(t *testing.T) {
	queue := NewJobQueue(10, 10)

	t.Run("update nil job", func(t *testing.T) {
		err := queue.Update(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job cannot be nil")
	})

	t.Run("update non-existent job", func(t *testing.T) {
		job := &Job{ID: "non-existent"}
		err := queue.Update(job)
		assert.Error(t, err)
		assert.Equal(t, ErrJobNotFound, err)
	})

	t.Run("update job to completed", func(t *testing.T) {
		job := NewJob("/test/path", nil)
		err := queue.Add(job)
		require.NoError(t, err)

		// Move to running
		ctx := context.Background()
		runningJob, err := queue.Next(ctx)
		require.NoError(t, err)

		// Update to completed
		runningJob.Status = JobStatusCompleted
		runningJob.Stats = &scanner.ScanStats{FilesIndexed: 100}

		err = queue.Update(runningJob)
		assert.NoError(t, err)

		// Check status
		updatedJob, err := queue.Get(job.ID)
		require.NoError(t, err)
		assert.Equal(t, JobStatusCompleted, updatedJob.Status)
		assert.NotNil(t, updatedJob.CompletedAt)
		assert.NotNil(t, updatedJob.Stats)

		stats := queue.Stats()
		assert.Equal(t, 0, stats["running"])
		assert.Equal(t, 1, stats["completed"])
		assert.Equal(t, 1, stats["total_completed"])
	})

	t.Run("update job to failed", func(t *testing.T) {
		job := NewJob("/test/path2", nil)
		err := queue.Add(job)
		require.NoError(t, err)

		// Move to running
		ctx := context.Background()
		runningJob, err := queue.Next(ctx)
		require.NoError(t, err)

		// Update to failed
		runningJob.Status = JobStatusFailed
		runningJob.Error = "scan failed"

		err = queue.Update(runningJob)
		assert.NoError(t, err)

		// Check status
		updatedJob, err := queue.Get(job.ID)
		require.NoError(t, err)
		assert.Equal(t, JobStatusFailed, updatedJob.Status)
		assert.Equal(t, "scan failed", updatedJob.Error)
		assert.NotNil(t, updatedJob.CompletedAt)

		stats := queue.Stats()
		assert.Equal(t, 1, stats["failed"])
		assert.Equal(t, 1, stats["total_failed"])
	})

	t.Run("update job to cancelled", func(t *testing.T) {
		job := NewJob("/test/path3", nil)
		err := queue.Add(job)
		require.NoError(t, err)

		// Update to cancelled (from pending)
		job.Status = JobStatusCancelled

		err = queue.Update(job)
		assert.NoError(t, err)

		// Check status
		updatedJob, err := queue.Get(job.ID)
		require.NoError(t, err)
		assert.Equal(t, JobStatusCancelled, updatedJob.Status)
		assert.NotNil(t, updatedJob.CompletedAt)

		stats := queue.Stats()
		assert.Equal(t, 1, stats["cancelled"])
		assert.Equal(t, 1, stats["total_cancelled"])
	})
}

func TestJobQueue_List(t *testing.T) {
	queue := NewJobQueue(10, 10)

	ctx := context.Background()

	// Add and process jobs to get them in different states

	// 1. Add a job that will stay pending
	pendingJob := &Job{ID: "pending-job", Path: "/pending"}
	err := queue.Add(pendingJob)
	require.NoError(t, err)

	// 2. Add a job that will become running
	runningJob := &Job{ID: "running-job", Path: "/running"}
	err = queue.Add(runningJob)
	require.NoError(t, err)

	// 3. Add a job that will be completed
	completedJob := &Job{ID: "completed-job", Path: "/completed"}
	err = queue.Add(completedJob)
	require.NoError(t, err)

	// Move first job (pending-job) to running
	firstJob, err := queue.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, pendingJob.ID, firstJob.ID)

	// Move second job (running-job) to running
	secondJob, err := queue.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, runningJob.ID, secondJob.ID)

	// Complete the first job
	firstJob.Status = JobStatusCompleted
	err = queue.Update(firstJob)
	require.NoError(t, err)

	// Now we have:
	// - completed: pending-job (first job)
	// - running: running-job (second job)
	// - pending: completed-job (third job, still in queue)

	t.Run("list pending jobs", func(t *testing.T) {
		jobs, err := queue.List(JobStatusPending)
		assert.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, completedJob.ID, jobs[0].ID) // third job is still pending
	})

	t.Run("list running jobs", func(t *testing.T) {
		jobs, err := queue.List(JobStatusRunning)
		assert.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, runningJob.ID, jobs[0].ID) // second job is running
	})

	t.Run("list completed jobs", func(t *testing.T) {
		jobs, err := queue.List(JobStatusCompleted)
		assert.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, pendingJob.ID, jobs[0].ID) // first job was completed
	})

	t.Run("list all jobs", func(t *testing.T) {
		jobs, err := queue.List("")
		assert.NoError(t, err)
		assert.Len(t, jobs, 3)
	})

	t.Run("list non-existent status", func(t *testing.T) {
		jobs, err := queue.List(JobStatusFailed)
		assert.NoError(t, err)
		assert.Len(t, jobs, 0)
	})
}

func TestJobQueue_Cancel(t *testing.T) {
	queue := NewJobQueue(10, 10)

	t.Run("cancel non-existent job", func(t *testing.T) {
		err := queue.Cancel("non-existent")
		assert.Error(t, err)
		assert.Equal(t, ErrJobNotFound, err)
	})

	t.Run("cancel pending job", func(t *testing.T) {
		job := NewJob("/pending", nil)
		err := queue.Add(job)
		require.NoError(t, err)

		err = queue.Cancel(job.ID)
		assert.NoError(t, err)

		// Check status
		cancelledJob, err := queue.Get(job.ID)
		require.NoError(t, err)
		assert.Equal(t, JobStatusCancelled, cancelledJob.Status)
		assert.NotNil(t, cancelledJob.CompletedAt)

		// Should be removed from pending
		assert.Equal(t, 0, queue.Size())

		stats := queue.Stats()
		assert.Equal(t, 1, stats["cancelled"])
	})

	t.Run("cancel running job", func(t *testing.T) {
		// Use fresh queue to avoid interference
		freshQueue := NewJobQueue(10, 10)

		job := &Job{ID: "cancel-running-job", Path: "/running"}
		err := freshQueue.Add(job)
		require.NoError(t, err)

		// Move to running
		ctx := context.Background()
		_, err = freshQueue.Next(ctx)
		require.NoError(t, err)

		err = freshQueue.Cancel(job.ID)
		assert.NoError(t, err)

		// Check status
		cancelledJob, err := freshQueue.Get(job.ID)
		require.NoError(t, err)
		assert.Equal(t, JobStatusCancelled, cancelledJob.Status)

		stats := freshQueue.Stats()
		assert.Equal(t, 0, stats["running"])
		assert.Equal(t, 1, stats["cancelled"])
	})

	t.Run("cancel completed job", func(t *testing.T) {
		job := NewJob("/completed", nil)
		err := queue.Add(job)
		require.NoError(t, err)

		// Move to running and complete
		ctx := context.Background()
		runningJob, err := queue.Next(ctx)
		require.NoError(t, err)

		runningJob.Status = JobStatusCompleted
		err = queue.Update(runningJob)
		require.NoError(t, err)

		// Try to cancel completed job
		err = queue.Cancel(job.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be cancelled")
	})
}

func TestJobQueue_Clear(t *testing.T) {
	queue := NewJobQueue(10, 10)

	t.Run("clear empty queue", func(t *testing.T) {
		err := queue.Clear()
		assert.NoError(t, err)
		assert.Equal(t, 0, queue.Size())
	})

	t.Run("clear queue with pending jobs", func(t *testing.T) {
		// Add several pending jobs
		for i := 0; i < 3; i++ {
			job := NewJob(fmt.Sprintf("/path%d", i), nil)
			err := queue.Add(job)
			require.NoError(t, err)
		}

		assert.Equal(t, 3, queue.Size())

		err := queue.Clear()
		assert.NoError(t, err)
		assert.Equal(t, 0, queue.Size())

		stats := queue.Stats()
		assert.Equal(t, 0, stats["pending"])
	})

	t.Run("clear does not affect running jobs", func(t *testing.T) {
		// Add job and move to running
		job := NewJob("/running", nil)
		err := queue.Add(job)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = queue.Next(ctx)
		require.NoError(t, err)

		// Add pending job
		pendingJob := NewJob("/pending", nil)
		err = queue.Add(pendingJob)
		require.NoError(t, err)

		// Clear should only remove pending
		err = queue.Clear()
		assert.NoError(t, err)

		stats := queue.Stats()
		assert.Equal(t, 0, stats["pending"])
		assert.Equal(t, 1, stats["running"])
	})
}

func TestJobQueue_Size(t *testing.T) {
	queue := NewJobQueue(10, 10)

	assert.Equal(t, 0, queue.Size())

	// Add jobs
	for i := 0; i < 3; i++ {
		job := NewJob(fmt.Sprintf("/path%d", i), nil)
		err := queue.Add(job)
		require.NoError(t, err)
		assert.Equal(t, i+1, queue.Size())
	}

	// Move one to running
	ctx := context.Background()
	_, err := queue.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, queue.Size())
}

func TestJobQueue_Stats(t *testing.T) {
	queue := NewJobQueue(10, 10)

	// Initial stats
	stats := queue.Stats()
	assert.Equal(t, 0, stats["pending"])
	assert.Equal(t, 0, stats["running"])
	assert.Equal(t, 0, stats["completed"])
	assert.Equal(t, 0, stats["failed"])
	assert.Equal(t, 0, stats["cancelled"])
	assert.Equal(t, 0, stats["total_added"])

	// Add jobs and check stats
	job1 := &Job{ID: "stats-job1", Path: "/path1"}
	job2 := &Job{ID: "stats-job2", Path: "/path2"}

	err := queue.Add(job1)
	require.NoError(t, err)
	err = queue.Add(job2)
	require.NoError(t, err)

	stats = queue.Stats()
	assert.Equal(t, 2, stats["pending"])
	assert.Equal(t, 2, stats["total_added"])

	// Move to running
	ctx := context.Background()
	runningJob, err := queue.Next(ctx)
	require.NoError(t, err)

	stats = queue.Stats()
	assert.Equal(t, 1, stats["pending"])
	assert.Equal(t, 1, stats["running"])

	// Complete job
	runningJob.Status = JobStatusCompleted
	err = queue.Update(runningJob)
	require.NoError(t, err)

	stats = queue.Stats()
	assert.Equal(t, 0, stats["running"])
	assert.Equal(t, 1, stats["completed"])
	assert.Equal(t, 1, stats["total_completed"])
}

func TestJobQueue_HistoryCleanup(t *testing.T) {
	// Create queue with small history limit
	queue := NewJobQueue(10, 2)

	// Add and complete multiple jobs
	for i := 0; i < 5; i++ {
		job := &Job{
			ID:   fmt.Sprintf("history-job-%d", i),
			Path: fmt.Sprintf("/path%d", i),
		}
		err := queue.Add(job)
		require.NoError(t, err)

		ctx := context.Background()
		runningJob, err := queue.Next(ctx)
		require.NoError(t, err)

		runningJob.Status = JobStatusCompleted
		err = queue.Update(runningJob)
		require.NoError(t, err)
	}

	// Should only keep the last 2 completed jobs
	completedJobs, err := queue.List(JobStatusCompleted)
	assert.NoError(t, err)
	assert.Len(t, completedJobs, 2)

	stats := queue.Stats()
	assert.Equal(t, 2, stats["completed"])
	assert.Equal(t, 5, stats["total_completed"]) // Total counter should not be affected
}

func TestJobQueue_Concurrency(t *testing.T) {
	queue := NewJobQueue(100, 100)

	var wg sync.WaitGroup
	numWorkers := 5    // Reduced to avoid timeout
	jobsPerWorker := 5 // Reduced to avoid timeout

	// Add all jobs first
	totalJobs := numWorkers * jobsPerWorker
	for i := 0; i < totalJobs; i++ {
		job := &Job{
			ID:   fmt.Sprintf("concurrent-job-%d", i),
			Path: fmt.Sprintf("/concurrent/path%d", i),
		}
		err := queue.Add(job)
		require.NoError(t, err)
	}

	// Consumer goroutines
	processedJobs := make([]*Job, 0, totalJobs)
	var processedMu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < jobsPerWorker; j++ {
				job, err := queue.Next(ctx)
				if err != nil {
					if err == context.DeadlineExceeded {
						return // Timeout is acceptable
					}
					t.Errorf("Unexpected error: %v", err)
					return
				}

				processedMu.Lock()
				processedJobs = append(processedJobs, job)
				processedMu.Unlock()

				// Simulate work and complete job
				job.Status = JobStatusCompleted
				err = queue.Update(job)
				if err != nil {
					t.Errorf("Failed to update job: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Verify results
	assert.Equal(t, totalJobs, len(processedJobs))

	stats := queue.Stats()
	assert.Equal(t, 0, stats["pending"])
	assert.Equal(t, 0, stats["running"])
	assert.Equal(t, totalJobs, stats["total_added"])
	assert.Equal(t, totalJobs, stats["total_completed"])
}
