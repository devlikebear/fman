/*
Copyright © 2025 changheonshin
*/
package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockQueue implements QueueInterface for testing
type MockQueue struct {
	mock.Mock
}

func (m *MockQueue) Add(job *Job) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockQueue) Next(ctx context.Context) (*Job, error) {
	args := m.Called(ctx)
	return args.Get(0).(*Job), args.Error(1)
}

func (m *MockQueue) Get(jobID string) (*Job, error) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Job), args.Error(1)
}

func (m *MockQueue) Update(job *Job) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockQueue) List(status JobStatus) ([]*Job, error) {
	args := m.Called(status)
	return args.Get(0).([]*Job), args.Error(1)
}

func (m *MockQueue) Cancel(jobID string) error {
	args := m.Called(jobID)
	return args.Error(0)
}

func (m *MockQueue) Clear() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockQueue) Size() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockQueue) Stats() map[string]int {
	args := m.Called()
	return args.Get(0).(map[string]int)
}

func TestNewDaemonServer(t *testing.T) {
	tests := []struct {
		name   string
		config *DaemonConfig
	}{
		{
			name:   "with config",
			config: &DaemonConfig{SocketPath: "test.sock"},
		},
		{
			name:   "with nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueue{}
			server := NewDaemonServer(tt.config, mockQueue)

			assert.NotNil(t, server)
			assert.NotNil(t, server.config)
			assert.Equal(t, mockQueue, server.queue)
			assert.NotNil(t, server.shutdownCh)
		})
	}
}

func TestDaemonServer_IsRunning(t *testing.T) {
	mockQueue := &MockQueue{}
	server := NewDaemonServer(nil, mockQueue)

	// Initially not running
	assert.False(t, server.IsRunning())

	// Set running state
	server.running = true
	assert.True(t, server.IsRunning())
}

func TestDaemonServer_EnqueueScan(t *testing.T) {
	tests := []struct {
		name        string
		running     bool
		queueError  error
		expectError bool
	}{
		{
			name:        "success",
			running:     true,
			queueError:  nil,
			expectError: false,
		},
		{
			name:        "daemon not running",
			running:     false,
			queueError:  nil,
			expectError: true,
		},
		{
			name:        "queue error",
			running:     true,
			queueError:  fmt.Errorf("queue full"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueue{}
			server := NewDaemonServer(nil, mockQueue)
			server.running = tt.running

			if tt.running {
				mockQueue.On("Add", mock.AnythingOfType("*daemon.Job")).Return(tt.queueError)
			}

			request := &ScanRequest{
				Path:    "/test/path",
				Options: &scanner.ScanOptions{Verbose: true},
			}

			job, err := server.EnqueueScan(request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, job)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, job)
				assert.Equal(t, request.Path, job.Path)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func TestDaemonServer_GetJob(t *testing.T) {
	tests := []struct {
		name        string
		running     bool
		jobID       string
		queueJob    *Job
		queueError  error
		expectError bool
	}{
		{
			name:        "success",
			running:     true,
			jobID:       "test-job-id",
			queueJob:    &Job{ID: "test-job-id"},
			queueError:  nil,
			expectError: false,
		},
		{
			name:        "daemon not running",
			running:     false,
			jobID:       "test-job-id",
			expectError: true,
		},
		{
			name:        "job not found",
			running:     true,
			jobID:       "nonexistent",
			queueJob:    nil,
			queueError:  ErrJobNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueue{}
			server := NewDaemonServer(nil, mockQueue)
			server.running = tt.running

			if tt.running {
				mockQueue.On("Get", tt.jobID).Return(tt.queueJob, tt.queueError)
			}

			job, err := server.GetJob(tt.jobID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.queueJob, job)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func TestDaemonServer_Status(t *testing.T) {
	tests := []struct {
		name        string
		running     bool
		stats       map[string]int
		expectError bool
	}{
		{
			name:    "success",
			running: true,
			stats: map[string]int{
				"pending":   5,
				"running":   2,
				"completed": 10,
				"failed":    1,
			},
			expectError: false,
		},
		{
			name:        "daemon not running",
			running:     false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueue{}
			server := NewDaemonServer(nil, mockQueue)
			server.running = tt.running
			server.startedAt = time.Now()

			if tt.running {
				mockQueue.On("Stats").Return(tt.stats)
			}

			status, err := server.Status()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, status)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, status)
				assert.Equal(t, tt.running, status.Running)
				assert.Equal(t, os.Getpid(), status.PID)
				assert.Equal(t, tt.stats["running"], status.ActiveJobs)
				assert.Equal(t, tt.stats["pending"], status.QueuedJobs)
				assert.Equal(t, tt.stats["completed"], status.CompletedJobs)
				assert.Equal(t, tt.stats["failed"], status.FailedJobs)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func TestDaemonServer_CancelJob(t *testing.T) {
	tests := []struct {
		name        string
		running     bool
		jobID       string
		queueError  error
		expectError bool
	}{
		{
			name:        "success",
			running:     true,
			jobID:       "test-job-id",
			queueError:  nil,
			expectError: false,
		},
		{
			name:        "daemon not running",
			running:     false,
			jobID:       "test-job-id",
			expectError: true,
		},
		{
			name:        "queue error",
			running:     true,
			jobID:       "test-job-id",
			queueError:  ErrJobNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueue{}
			server := NewDaemonServer(nil, mockQueue)
			server.running = tt.running

			if tt.running {
				mockQueue.On("Cancel", tt.jobID).Return(tt.queueError)
			}

			err := server.CancelJob(tt.jobID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func TestDaemonServer_ListJobs(t *testing.T) {
	tests := []struct {
		name        string
		running     bool
		status      JobStatus
		jobs        []*Job
		queueError  error
		expectError bool
	}{
		{
			name:    "success with status filter",
			running: true,
			status:  JobStatusPending,
			jobs: []*Job{
				{ID: "job1", Status: JobStatusPending},
				{ID: "job2", Status: JobStatusPending},
			},
			queueError:  nil,
			expectError: false,
		},
		{
			name:        "daemon not running",
			running:     false,
			status:      JobStatusPending,
			expectError: true,
		},
		{
			name:        "queue error",
			running:     true,
			status:      JobStatusPending,
			jobs:        nil,
			queueError:  fmt.Errorf("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueue{}
			server := NewDaemonServer(nil, mockQueue)
			server.running = tt.running

			if tt.running {
				mockQueue.On("List", tt.status).Return(tt.jobs, tt.queueError)
			}

			jobs, err := server.ListJobs(tt.status)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.jobs, jobs)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func TestDaemonServer_ClearQueue(t *testing.T) {
	tests := []struct {
		name        string
		running     bool
		queueError  error
		expectError bool
	}{
		{
			name:        "success",
			running:     true,
			queueError:  nil,
			expectError: false,
		},
		{
			name:        "daemon not running",
			running:     false,
			expectError: true,
		},
		{
			name:        "queue error",
			running:     true,
			queueError:  fmt.Errorf("operation failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &MockQueue{}
			server := NewDaemonServer(nil, mockQueue)
			server.running = tt.running

			if tt.running {
				mockQueue.On("Clear").Return(tt.queueError)
			}

			err := server.ClearQueue()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockQueue.AssertExpectations(t)
		})
	}
}

func TestDaemonServer_PathHelpers(t *testing.T) {
	tempDir := t.TempDir()
	config := &DaemonConfig{
		SocketPath: "test.sock",
		PIDPath:    "test.pid",
	}

	t.Run("relative paths", func(t *testing.T) {
		server := NewDaemonServer(config, &MockQueue{})

		socketPath := server.getSocketPath()
		pidPath := server.getPIDPath()

		assert.Contains(t, socketPath, "test.sock")
		assert.Contains(t, pidPath, "test.pid")
	})

	t.Run("absolute paths", func(t *testing.T) {
		config := &DaemonConfig{
			SocketPath: filepath.Join(tempDir, "test.sock"),
			PIDPath:    filepath.Join(tempDir, "test.pid"),
		}
		server := NewDaemonServer(config, &MockQueue{})

		socketPath := server.getSocketPath()
		pidPath := server.getPIDPath()

		assert.Equal(t, filepath.Join(tempDir, "test.sock"), socketPath)
		assert.Equal(t, filepath.Join(tempDir, "test.pid"), pidPath)
	})
}

func TestDaemonServer_PIDFile(t *testing.T) {
	tempDir := t.TempDir()
	config := &DaemonConfig{
		PIDPath: filepath.Join(tempDir, "test.pid"),
	}
	server := NewDaemonServer(config, &MockQueue{})

	t.Run("write and check PID file", func(t *testing.T) {
		err := server.writePIDFile()
		require.NoError(t, err)

		// Check if file exists and contains correct PID
		data, err := os.ReadFile(server.getPIDPath())
		require.NoError(t, err)

		pid, err := strconv.Atoi(string(data))
		require.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)

		// Check if daemon is detected as running
		assert.True(t, server.isDaemonRunning())
	})

	t.Run("remove PID file", func(t *testing.T) {
		err := server.removePIDFile()
		require.NoError(t, err)

		// Check if file is removed
		_, err = os.Stat(server.getPIDPath())
		assert.True(t, os.IsNotExist(err))

		// Check if daemon is detected as not running
		assert.False(t, server.isDaemonRunning())
	})

	t.Run("remove non-existent PID file", func(t *testing.T) {
		err := server.removePIDFile()
		assert.NoError(t, err) // Should not error
	})
}

func TestDaemonServer_SocketFile(t *testing.T) {
	tempDir := t.TempDir()
	config := &DaemonConfig{
		SocketPath: filepath.Join(tempDir, "test.sock"),
	}
	server := NewDaemonServer(config, &MockQueue{})

	t.Run("remove non-existent socket file", func(t *testing.T) {
		err := server.removeSocketFile()
		assert.NoError(t, err) // Should not error
	})

	t.Run("remove existing socket file", func(t *testing.T) {
		// Create a dummy socket file
		socketPath := server.getSocketPath()
		file, err := os.Create(socketPath)
		require.NoError(t, err)
		file.Close()

		err = server.removeSocketFile()
		require.NoError(t, err)

		// Check if file is removed
		_, err = os.Stat(socketPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestDaemonServer_MessageHandling(t *testing.T) {
	mockQueue := &MockQueue{}
	server := NewDaemonServer(nil, mockQueue)
	server.running = true

	t.Run("handle scan request", func(t *testing.T) {
		mockQueue.On("Add", mock.AnythingOfType("*daemon.Job")).Return(nil)

		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "test-id",
			Request: &Request{
				Type: RequestTypeScan,
				Data: map[string]interface{}{
					"path": "/test/path",
					"options": map[string]interface{}{
						"verbose": true,
					},
				},
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "test-id", response.ID)
		assert.True(t, response.Response.Success)
		assert.NotNil(t, response.Response.Data)
	})

	t.Run("handle status request", func(t *testing.T) {
		mockQueue.On("Stats").Return(map[string]int{
			"pending":   1,
			"running":   0,
			"completed": 2,
			"failed":    0,
		})

		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "status-id",
			Request: &Request{
				Type: RequestTypeStatus,
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "status-id", response.ID)
		assert.True(t, response.Response.Success)
		assert.NotNil(t, response.Response.Data)
	})

	t.Run("handle invalid request", func(t *testing.T) {
		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "invalid-id",
			// Missing Request field
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "invalid-id", response.ID)
		assert.False(t, response.Response.Success)
		assert.Contains(t, response.Response.Error, "missing request")
	})

	t.Run("handle unknown request type", func(t *testing.T) {
		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "unknown-id",
			Request: &Request{
				Type: RequestType("unknown"),
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "unknown-id", response.ID)
		assert.False(t, response.Response.Success)
		assert.Contains(t, response.Response.Error, "unknown request type")
	})

	mockQueue.AssertExpectations(t)
}

func TestDaemonServer_parseRequestData(t *testing.T) {
	server := NewDaemonServer(nil, &MockQueue{})

	t.Run("success", func(t *testing.T) {
		data := map[string]interface{}{
			"path": "/test/path",
			"options": map[string]interface{}{
				"verbose": true,
			},
		}

		var scanReq ScanRequest
		err := server.parseRequestData(data, &scanReq)

		assert.NoError(t, err)
		assert.Equal(t, "/test/path", scanReq.Path)
		assert.True(t, scanReq.Options.Verbose)
	})

	t.Run("nil data", func(t *testing.T) {
		var scanReq ScanRequest
		err := server.parseRequestData(nil, &scanReq)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing request data")
	})
}

func TestWorker_processJob(t *testing.T) {
	mockQueue := &MockQueue{}
	server := NewDaemonServer(nil, mockQueue)

	worker := &Worker{
		id:     0,
		server: server,
		ctx:    context.Background(),
	}

	job := &Job{
		ID:      "test-job",
		Path:    t.TempDir(), // Use temp dir for testing
		Options: &scanner.ScanOptions{Verbose: false},
		Status:  JobStatusPending,
	}

	// Mock queue Update calls
	mockQueue.On("Update", mock.AnythingOfType("*daemon.Job")).Return(nil)

	worker.processJob(job)

	// Verify job was updated
	mockQueue.AssertExpectations(t)

	// Job should be completed or failed (depending on scan result)
	assert.True(t, job.Status == JobStatusCompleted || job.Status == JobStatusFailed)
	assert.NotNil(t, job.StartedAt)
	assert.NotNil(t, job.CompletedAt)
}

func TestDaemonServer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	config := &DaemonConfig{
		SocketPath: filepath.Join(tempDir, "test.sock"),
		PIDPath:    filepath.Join(tempDir, "test.pid"),
		MaxWorkers: 1,
		QueueSize:  10,
	}

	mockQueue := &MockQueue{}
	server := NewDaemonServer(config, mockQueue)

	// Mock queue for server start and status checks
	mockQueue.On("Stats").Return(map[string]int{
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
	}).Maybe()

	// Mock Next method for workers (should timeout and return context error)
	mockQueue.On("Next", mock.Anything).Return((*Job)(nil), context.DeadlineExceeded).Maybe()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("start and stop server", func(t *testing.T) {
		err := server.Start(ctx)
		require.NoError(t, err)
		assert.True(t, server.IsRunning())

		// Check if socket file exists
		_, err = os.Stat(server.getSocketPath())
		assert.NoError(t, err)

		// Check if PID file exists
		_, err = os.Stat(server.getPIDPath())
		assert.NoError(t, err)

		err = server.Stop()
		require.NoError(t, err)
		assert.False(t, server.IsRunning())

		// Check if files are cleaned up
		_, err = os.Stat(server.getSocketPath())
		assert.True(t, os.IsNotExist(err))

		_, err = os.Stat(server.getPIDPath())
		assert.True(t, os.IsNotExist(err))
	})

	mockQueue.AssertExpectations(t)
}

func TestDaemonServer_HandleRequestTypes(t *testing.T) {
	mockQueue := &MockQueue{}
	server := NewDaemonServer(nil, mockQueue)
	server.running = true

	t.Run("handle job status request", func(t *testing.T) {
		job := &Job{ID: "test-job", Status: JobStatusPending}
		mockQueue.On("Get", "test-job").Return(job, nil)

		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "job-status-id",
			Request: &Request{
				Type: RequestTypeJobStatus,
				Data: "test-job",
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "job-status-id", response.ID)
		assert.True(t, response.Response.Success)
		assert.Equal(t, job, response.Response.Data)
	})

	t.Run("handle job status request with invalid data", func(t *testing.T) {
		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "job-status-invalid-id",
			Request: &Request{
				Type: RequestTypeJobStatus,
				Data: 123, // Invalid type
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "job-status-invalid-id", response.ID)
		assert.False(t, response.Response.Success)
		assert.Contains(t, response.Response.Error, "job ID must be a string")
	})

	t.Run("handle job list request", func(t *testing.T) {
		jobs := []*Job{
			{ID: "job1", Status: JobStatusPending},
			{ID: "job2", Status: JobStatusRunning},
		}
		mockQueue.On("List", JobStatus("")).Return(jobs, nil)

		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "job-list-id",
			Request: &Request{
				Type: RequestTypeJobList,
				Data: nil,
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "job-list-id", response.ID)
		assert.True(t, response.Response.Success)
		assert.Equal(t, jobs, response.Response.Data)
	})

	t.Run("handle job cancel request", func(t *testing.T) {
		mockQueue.On("Cancel", "test-job").Return(nil)

		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "job-cancel-id",
			Request: &Request{
				Type: RequestTypeJobCancel,
				Data: "test-job",
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "job-cancel-id", response.ID)
		assert.True(t, response.Response.Success)
	})

	t.Run("handle job cancel request with invalid data", func(t *testing.T) {
		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "job-cancel-invalid-id",
			Request: &Request{
				Type: RequestTypeJobCancel,
				Data: 123, // Invalid type
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "job-cancel-invalid-id", response.ID)
		assert.False(t, response.Response.Success)
		assert.Contains(t, response.Response.Error, "job ID must be a string")
	})

	t.Run("handle queue clear request", func(t *testing.T) {
		mockQueue.On("Clear").Return(nil)

		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "queue-clear-id",
			Request: &Request{
				Type: RequestTypeQueueClear,
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "queue-clear-id", response.ID)
		assert.True(t, response.Response.Success)
	})

	t.Run("handle shutdown request", func(t *testing.T) {
		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "shutdown-id",
			Request: &Request{
				Type: RequestTypeShutdown,
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "shutdown-id", response.ID)
		assert.True(t, response.Response.Success)
	})

	mockQueue.AssertExpectations(t)
}

func TestDaemonServer_ErrorHandling(t *testing.T) {
	mockQueue := &MockQueue{}
	server := NewDaemonServer(nil, mockQueue)
	server.running = true

	t.Run("handle scan request with invalid data", func(t *testing.T) {
		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "scan-invalid-id",
			Request: &Request{
				Type: RequestTypeScan,
				Data: "invalid", // Should be a map
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "scan-invalid-id", response.ID)
		assert.False(t, response.Response.Success)
		assert.Contains(t, response.Response.Error, "invalid scan request")
	})

	t.Run("handle status request when not running", func(t *testing.T) {
		server.running = false

		msg := &Message{
			Type: MessageTypeRequest,
			ID:   "status-not-running-id",
			Request: &Request{
				Type: RequestTypeStatus,
			},
		}

		response := server.handleMessage(msg)
		assert.Equal(t, MessageTypeResponse, response.Type)
		assert.Equal(t, "status-not-running-id", response.ID)
		assert.False(t, response.Response.Success)
		assert.Contains(t, response.Response.Error, "failed to get status")
	})
}

func TestDaemonServer_StartErrors(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("daemon already running", func(t *testing.T) {
		config := &DaemonConfig{
			SocketPath: filepath.Join(tempDir, "test1.sock"),
			PIDPath:    filepath.Join(tempDir, "test1.pid"),
		}

		mockQueue := &MockQueue{}
		server := NewDaemonServer(config, mockQueue)
		server.running = true

		ctx := context.Background()
		err := server.Start(ctx)
		assert.Equal(t, ErrDaemonAlreadyRunning, err)
	})

	t.Run("invalid socket directory", func(t *testing.T) {
		config := &DaemonConfig{
			SocketPath: "/invalid/nonexistent/path/test.sock",
			PIDPath:    filepath.Join(tempDir, "test2.pid"),
		}

		mockQueue := &MockQueue{}
		server := NewDaemonServer(config, mockQueue)

		ctx := context.Background()
		err := server.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create socket directory")
	})
}

func TestDaemonServer_StopNotRunning(t *testing.T) {
	mockQueue := &MockQueue{}
	server := NewDaemonServer(nil, mockQueue)

	err := server.Stop()
	assert.Equal(t, ErrDaemonNotRunning, err)
}
