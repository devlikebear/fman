/*
Copyright © 2025 changheonshin
*/
package daemon

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/devlikebear/fman/internal/scanner"
	"github.com/spf13/afero"
)

// ScanWorker manages background scan execution
type ScanWorker struct {
	id              int
	queue           QueueInterface
	scanner         scanner.ScannerInterface
	running         bool
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	stats           WorkerStats
	maxRetries      int
	retryDelay      time.Duration
	progressChan    chan *JobProgress
	resourceMonitor *ResourceMonitor
}

// WorkerStats tracks worker performance metrics
type WorkerStats struct {
	JobsProcessed int64
	JobsSucceeded int64
	JobsFailed    int64
	TotalRuntime  time.Duration
	LastJobAt     *time.Time
	MemoryUsage   uint64
}

// NewScanWorker creates a new scan worker
func NewScanWorker(id int, queue QueueInterface) *ScanWorker {
	fs := afero.NewOsFs()
	database := db.NewDatabase(nil)
	scannerImpl := scanner.NewFileScanner(fs, database)

	// 리소스 모니터 설정 (데몬용 보수적 설정)
	resourceLimits := ResourceLimits{
		MaxMemoryMB:   200,                    // 200MB 메모리 제한
		MaxCPUPercent: 15.0,                   // 15% CPU 사용률 제한
		CheckInterval: 3 * time.Second,        // 3초마다 체크
		ThrottleDelay: 200 * time.Millisecond, // 제한 시 200ms 대기
	}

	return &ScanWorker{
		id:              id,
		queue:           queue,
		scanner:         scannerImpl,
		maxRetries:      3,
		retryDelay:      time.Second * 5,
		progressChan:    make(chan *JobProgress, 100),
		resourceMonitor: NewResourceMonitor(resourceLimits),
	}
}

// Start starts the worker in a background goroutine
func (w *ScanWorker) Start(ctx context.Context, wg *sync.WaitGroup) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.running = true

	// 리소스 모니터링 시작
	w.resourceMonitor.Start(w.ctx)

	wg.Add(1)
	go w.run(wg)
}

// Stop stops the worker gracefully
func (w *ScanWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	if w.cancel != nil {
		w.cancel()
	}

	// 리소스 모니터링 중지
	w.resourceMonitor.Stop()
	w.running = false
}

// IsRunning returns whether the worker is currently running
func (w *ScanWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetStats returns current worker statistics
func (w *ScanWorker) GetStats() WorkerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Update memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	w.stats.MemoryUsage = m.Alloc

	return w.stats
}

// GetProgressChannel returns the progress channel for monitoring
func (w *ScanWorker) GetProgressChannel() <-chan *JobProgress {
	return w.progressChan
}

// run is the main worker loop
func (w *ScanWorker) run(wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(w.progressChan)

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// 리소스 사용량 확인 후 대기 (필요시)
		if err := w.resourceMonitor.WaitIfThrottling(w.ctx); err != nil {
			return
		}

		// Get next job from queue with timeout
		jobCtx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
		job, err := w.queue.Next(jobCtx)
		cancel()

		if err != nil {
			if w.ctx.Err() != nil {
				return
			}
			// No job available, continue polling
			continue
		}

		// Process the job
		w.processJobs(job)
	}
}

// processJobs handles a single job with retry logic
func (w *ScanWorker) processJobs(job *Job) {
	w.mu.Lock()
	w.stats.JobsProcessed++
	w.mu.Unlock()

	startTime := time.Now()

	// Update job status to running
	w.updateJobStatus(job, JobStatusRunning, startTime, nil, nil)

	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-w.ctx.Done():
				return
			case <-time.After(w.retryDelay):
			}
		}

		// Execute scan with progress monitoring
		err := w.executeScan(job)
		if err == nil {
			// Success
			w.handleScanResult(job, nil, startTime)
			w.mu.Lock()
			w.stats.JobsSucceeded++
			w.mu.Unlock()
			return
		}

		lastErr = err

		// Check if error is retryable
		if !w.isRetryableError(err) {
			break
		}
	}

	// All retries failed
	w.handleScanResult(job, lastErr, startTime)
	w.mu.Lock()
	w.stats.JobsFailed++
	w.mu.Unlock()
}

// executeScan performs the actual scan operation
func (w *ScanWorker) executeScan(job *Job) error {
	// Create a context that can be cancelled if the job is cancelled
	ctx, cancel := context.WithCancel(w.ctx)
	defer cancel()

	// Monitor for job cancellation
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check if job was cancelled
				currentJob, err := w.queue.Get(job.ID)
				if err != nil || currentJob.Status == JobStatusCancelled {
					cancel()
					return
				}
			}
		}
	}()

	// Execute the scan
	stats, err := w.scanner.ScanDirectory(ctx, job.Path, job.Options)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Update job with scan results
	job.Stats = stats
	return nil
}

// updateProgress sends progress updates
func (w *ScanWorker) updateProgress(job *Job, progress *JobProgress) {
	// Update job progress in queue if supported
	job.Progress = progress
	w.queue.Update(job)

	// Send progress to channel (non-blocking)
	select {
	case w.progressChan <- progress:
	default:
		// Channel is full, skip this update
	}
}

// updateJobStatus updates the job status in the queue
func (w *ScanWorker) updateJobStatus(job *Job, status JobStatus, startTime time.Time, completedTime *time.Time, err error) {
	job.Status = status

	if status == JobStatusRunning {
		job.StartedAt = &startTime
	}

	if completedTime != nil {
		job.CompletedAt = completedTime
	}

	if err != nil {
		job.Error = err.Error()
	}

	w.queue.Update(job)
}

// handleScanResult processes the result of a scan operation
func (w *ScanWorker) handleScanResult(job *Job, err error, startTime time.Time) {
	completedAt := time.Now()
	duration := completedAt.Sub(startTime)

	w.mu.Lock()
	w.stats.TotalRuntime += duration
	w.stats.LastJobAt = &completedAt
	w.mu.Unlock()

	if err != nil {
		w.updateJobStatus(job, JobStatusFailed, startTime, &completedAt, err)
	} else {
		w.updateJobStatus(job, JobStatusCompleted, startTime, &completedAt, nil)
	}

	// Send final progress update
	progress := &JobProgress{
		FilesProcessed: 0,
		TotalFiles:     0,
		CurrentPath:    job.Path,
	}

	if job.Stats != nil {
		progress.FilesProcessed = job.Stats.FilesIndexed
		progress.TotalFiles = job.Stats.FilesIndexed
	}

	w.updateProgress(job, progress)
}

// isRetryableError determines if an error should trigger a retry
func (w *ScanWorker) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types that are retryable
	errStr := err.Error()
	retryableErrors := []string{
		"permission denied",
		"no such file or directory",
		"device or resource busy",
		"connection refused",
		"timeout",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsInner(s, substr))))
}

// containsInner checks for substring in the middle of string
func containsInner(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
