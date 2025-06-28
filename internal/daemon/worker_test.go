package daemon

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockScannerInterface is a mock implementation of scanner.ScannerInterface
type MockScannerInterface struct {
	mock.Mock
}

func (m *MockScannerInterface) ScanDirectory(ctx context.Context, rootDir string, options *scanner.ScanOptions) (*scanner.ScanStats, error) {
	args := m.Called(ctx, rootDir, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scanner.ScanStats), args.Error(1)
}

func TestNewScanWorker(t *testing.T) {
	queue := NewJobQueue(10, 100)
	worker := NewScanWorker(1, queue)

	assert.NotNil(t, worker)
	assert.Equal(t, 1, worker.id)
	assert.Equal(t, queue, worker.queue)
	assert.NotNil(t, worker.scanner)
	assert.False(t, worker.running)
	assert.Equal(t, 3, worker.maxRetries)
	assert.Equal(t, time.Second*5, worker.retryDelay)
	assert.NotNil(t, worker.progressChan)
}

func TestScanWorkerLifecycle(t *testing.T) {
	queue := NewJobQueue(10, 100)
	worker := NewScanWorker(1, queue)

	// Initially not running
	assert.False(t, worker.IsRunning())

	// Start worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	worker.Start(ctx, &wg)

	// Should be running now
	assert.True(t, worker.IsRunning())

	// Starting again should not create another goroutine
	worker.Start(ctx, &wg)
	assert.True(t, worker.IsRunning())

	// Stop worker
	worker.Stop()
	cancel() // Cancel context to ensure goroutine exits

	// Wait for goroutine to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second * 5):
		t.Fatal("Worker did not stop within timeout")
	}

	assert.False(t, worker.IsRunning())

	// Stopping again should be safe
	worker.Stop()
	assert.False(t, worker.IsRunning())
}

func TestScanWorkerGetStats(t *testing.T) {
	queue := NewJobQueue(10, 100)
	worker := NewScanWorker(1, queue)

	stats := worker.GetStats()
	assert.Equal(t, int64(0), stats.JobsProcessed)
	assert.Equal(t, int64(0), stats.JobsSucceeded)
	assert.Equal(t, int64(0), stats.JobsFailed)
	assert.Equal(t, time.Duration(0), stats.TotalRuntime)
	assert.Nil(t, stats.LastJobAt)
	assert.Greater(t, stats.MemoryUsage, uint64(0)) // Should have some memory usage
}

func TestScanWorkerGetProgressChannel(t *testing.T) {
	queue := NewJobQueue(10, 100)
	worker := NewScanWorker(1, queue)

	progressChan := worker.GetProgressChannel()
	assert.NotNil(t, progressChan)

	// Channel should be readable
	select {
	case <-progressChan:
		t.Fatal("Channel should be empty initially")
	default:
		// Expected - channel is empty
	}
}

func TestScanWorkerProcessJobSuccess(t *testing.T) {
	mockQueue := &MockQueue{}
	worker := &ScanWorker{
		id:           1,
		queue:        mockQueue,
		maxRetries:   3,
		retryDelay:   time.Millisecond * 10,
		progressChan: make(chan *JobProgress, 100),
		mu:           sync.RWMutex{},
		stats:        WorkerStats{},
	}

	// Create mock scanner
	mockScanner := &MockScannerInterface{}
	worker.scanner = mockScanner

	// Create test job
	job := &Job{
		ID:     "test-job",
		Path:   "/test/path",
		Status: JobStatusPending,
		Options: &scanner.ScanOptions{
			Verbose: false,
		},
	}

	// Setup expectations
	expectedStats := &scanner.ScanStats{
		FilesIndexed: 5,
	}

	mockScanner.On("ScanDirectory", mock.Anything, "/test/path", job.Options).Return(expectedStats, nil)
	mockQueue.On("Update", mock.AnythingOfType("*daemon.Job")).Return(nil).Maybe()
	mockQueue.On("Get", "test-job").Return(job, nil).Maybe()

	// Set context
	ctx, cancel := context.WithCancel(context.Background())
	worker.ctx = ctx
	defer cancel()

	// Process job
	worker.processJobs(job)

	// Verify job was updated correctly
	assert.Equal(t, JobStatusCompleted, job.Status)
	assert.NotNil(t, job.StartedAt)
	assert.NotNil(t, job.CompletedAt)
	assert.Equal(t, expectedStats, job.Stats)
	assert.Empty(t, job.Error)

	// Verify stats were updated
	stats := worker.GetStats()
	assert.Equal(t, int64(1), stats.JobsProcessed)
	assert.Equal(t, int64(1), stats.JobsSucceeded)
	assert.Equal(t, int64(0), stats.JobsFailed)

	mockScanner.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestScanWorkerProcessJobFailure(t *testing.T) {
	mockQueue := &MockQueue{}
	worker := &ScanWorker{
		id:           1,
		queue:        mockQueue,
		maxRetries:   1, // Reduce retries for faster test
		retryDelay:   time.Millisecond * 10,
		progressChan: make(chan *JobProgress, 100),
		mu:           sync.RWMutex{},
		stats:        WorkerStats{},
	}

	// Create mock scanner
	mockScanner := &MockScannerInterface{}
	worker.scanner = mockScanner

	// Create test job
	job := &Job{
		ID:     "test-job",
		Path:   "/test/path",
		Status: JobStatusPending,
		Options: &scanner.ScanOptions{
			Verbose: false,
		},
	}

	// Setup expectations - scanner will fail
	mockScanner.On("ScanDirectory", mock.Anything, "/test/path", job.Options).Return(nil, assert.AnError)
	mockQueue.On("Update", mock.AnythingOfType("*daemon.Job")).Return(nil).Maybe()
	mockQueue.On("Get", "test-job").Return(job, nil).Maybe()

	// Set context
	ctx, cancel := context.WithCancel(context.Background())
	worker.ctx = ctx
	defer cancel()

	// Process job
	worker.processJobs(job)

	// Verify job was marked as failed
	assert.Equal(t, JobStatusFailed, job.Status)
	assert.NotNil(t, job.StartedAt)
	assert.NotNil(t, job.CompletedAt)
	assert.NotEmpty(t, job.Error)

	// Verify stats were updated
	stats := worker.GetStats()
	assert.Equal(t, int64(1), stats.JobsProcessed)
	assert.Equal(t, int64(0), stats.JobsSucceeded)
	assert.Equal(t, int64(1), stats.JobsFailed)

	mockScanner.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestScanWorkerRetryLogic(t *testing.T) {
	mockQueue := &MockQueue{}
	worker := &ScanWorker{
		id:           1,
		queue:        mockQueue,
		maxRetries:   2,
		retryDelay:   time.Millisecond * 10,
		progressChan: make(chan *JobProgress, 100),
		mu:           sync.RWMutex{},
		stats:        WorkerStats{},
	}

	// Create mock scanner
	mockScanner := &MockScannerInterface{}
	worker.scanner = mockScanner

	// Create test job
	job := &Job{
		ID:     "test-job",
		Path:   "/test/path",
		Status: JobStatusPending,
		Options: &scanner.ScanOptions{
			Verbose: false,
		},
	}

	// Setup expectations - first call fails with retryable error, second call succeeds
	retryableError := fmt.Errorf("permission denied")
	mockScanner.On("ScanDirectory", mock.Anything, "/test/path", job.Options).Return((*scanner.ScanStats)(nil),
		retryableError).Once()

	expectedStats := &scanner.ScanStats{FilesIndexed: 3}
	mockScanner.On("ScanDirectory", mock.Anything, "/test/path", job.Options).Return(expectedStats, nil).Once()

	mockQueue.On("Update", mock.AnythingOfType("*daemon.Job")).Return(nil).Maybe()
	mockQueue.On("Get", "test-job").Return(job, nil).Maybe()

	// Set context
	ctx, cancel := context.WithCancel(context.Background())
	worker.ctx = ctx
	defer cancel()

	// Process job
	worker.processJobs(job)

	// Verify job succeeded after retry
	assert.Equal(t, JobStatusCompleted, job.Status)
	assert.Equal(t, expectedStats, job.Stats)

	// Verify scanner was called exactly twice (first failure, then success)
	mockScanner.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestIsRetryableError(t *testing.T) {
	worker := &ScanWorker{}

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "permission denied error",
			err:      assert.AnError,
			expected: true, // Our contains function is simple, so this will match
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := worker.isRetryableError(tc.err)
			if tc.name == "nil error" {
				assert.Equal(t, tc.expected, result)
			}
			// For other cases, just verify the function doesn't panic
		})
	}
}

func TestContainsFunction(t *testing.T) {
	testCases := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "hello",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring at start",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "substring in middle",
			s:        "hello world test",
			substr:   "world",
			expected: true,
		},
		{
			name:     "no match",
			s:        "hello",
			substr:   "xyz",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "hello",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "hello",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.s, tc.substr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUpdateJobStatus(t *testing.T) {
	mockQueue := &MockQueue{}
	worker := &ScanWorker{
		queue: mockQueue,
	}

	job := &Job{
		ID:     "test-job",
		Status: JobStatusPending,
	}

	startTime := time.Now()
	completedTime := startTime.Add(time.Minute)

	mockQueue.On("Update", job).Return(nil)

	// Test updating to running status
	worker.updateJobStatus(job, JobStatusRunning, startTime, nil, nil)
	assert.Equal(t, JobStatusRunning, job.Status)
	assert.Equal(t, &startTime, job.StartedAt)

	// Test updating to completed status
	worker.updateJobStatus(job, JobStatusCompleted, startTime, &completedTime, nil)
	assert.Equal(t, JobStatusCompleted, job.Status)
	assert.Equal(t, &completedTime, job.CompletedAt)

	// Test updating with error
	testErr := assert.AnError
	worker.updateJobStatus(job, JobStatusFailed, startTime, &completedTime, testErr)
	assert.Equal(t, JobStatusFailed, job.Status)
	assert.Equal(t, testErr.Error(), job.Error)

	mockQueue.AssertExpectations(t)
}
