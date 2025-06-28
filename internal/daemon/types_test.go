/*
Copyright © 2025 changheonshin
*/
package daemon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobSerialization(t *testing.T) {
	// Create a test job
	options := &scanner.ScanOptions{
		Verbose:   true,
		ForceSudo: false,
	}

	job := NewJob("/test/path", options)
	job.Status = JobStatusRunning
	startTime := time.Now()
	job.StartedAt = &startTime

	// Test JSON serialization
	data, err := json.Marshal(job)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test JSON deserialization
	var deserializedJob Job
	err = json.Unmarshal(data, &deserializedJob)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, job.ID, deserializedJob.ID)
	assert.Equal(t, job.Path, deserializedJob.Path)
	assert.Equal(t, job.Status, deserializedJob.Status)
	assert.Equal(t, job.Options.Verbose, deserializedJob.Options.Verbose)
	assert.Equal(t, job.Options.ForceSudo, deserializedJob.Options.ForceSudo)
	assert.NotNil(t, deserializedJob.StartedAt)
}

func TestJobProgress(t *testing.T) {
	progress := &JobProgress{
		FilesProcessed: 50,
		TotalFiles:     100,
		CurrentPath:    "/current/file.txt",
	}

	// Test JSON serialization
	data, err := json.Marshal(progress)
	require.NoError(t, err)

	// Test JSON deserialization
	var deserializedProgress JobProgress
	err = json.Unmarshal(data, &deserializedProgress)
	require.NoError(t, err)

	assert.Equal(t, progress.FilesProcessed, deserializedProgress.FilesProcessed)
	assert.Equal(t, progress.TotalFiles, deserializedProgress.TotalFiles)
	assert.Equal(t, progress.CurrentPath, deserializedProgress.CurrentPath)
}

func TestScanRequest(t *testing.T) {
	options := &scanner.ScanOptions{
		Verbose:   true,
		ForceSudo: false,
	}

	request := &ScanRequest{
		Path:    "/test/scan/path",
		Options: options,
	}

	// Test JSON serialization
	data, err := json.Marshal(request)
	require.NoError(t, err)

	// Test JSON deserialization
	var deserializedRequest ScanRequest
	err = json.Unmarshal(data, &deserializedRequest)
	require.NoError(t, err)

	assert.Equal(t, request.Path, deserializedRequest.Path)
	assert.Equal(t, request.Options.Verbose, deserializedRequest.Options.Verbose)
	assert.Equal(t, request.Options.ForceSudo, deserializedRequest.Options.ForceSudo)
}

func TestDaemonStatus(t *testing.T) {
	status := &DaemonStatus{
		Running:       true,
		PID:           12345,
		StartedAt:     time.Now(),
		ActiveJobs:    2,
		QueuedJobs:    5,
		CompletedJobs: 10,
		FailedJobs:    1,
		Workers:       3,
	}

	// Test JSON serialization
	data, err := json.Marshal(status)
	require.NoError(t, err)

	// Test JSON deserialization
	var deserializedStatus DaemonStatus
	err = json.Unmarshal(data, &deserializedStatus)
	require.NoError(t, err)

	assert.Equal(t, status.Running, deserializedStatus.Running)
	assert.Equal(t, status.PID, deserializedStatus.PID)
	assert.Equal(t, status.ActiveJobs, deserializedStatus.ActiveJobs)
	assert.Equal(t, status.QueuedJobs, deserializedStatus.QueuedJobs)
	assert.Equal(t, status.CompletedJobs, deserializedStatus.CompletedJobs)
	assert.Equal(t, status.FailedJobs, deserializedStatus.FailedJobs)
	assert.Equal(t, status.Workers, deserializedStatus.Workers)
}

func TestMessage(t *testing.T) {
	request := &Request{
		Type: RequestTypeScan,
		Data: map[string]interface{}{
			"path": "/test/path",
		},
	}

	message := &Message{
		Type:      MessageTypeRequest,
		ID:        "test-message-123",
		Timestamp: time.Now(),
		Request:   request,
	}

	// Test JSON serialization
	data, err := json.Marshal(message)
	require.NoError(t, err)

	// Test JSON deserialization
	var deserializedMessage Message
	err = json.Unmarshal(data, &deserializedMessage)
	require.NoError(t, err)

	assert.Equal(t, message.Type, deserializedMessage.Type)
	assert.Equal(t, message.ID, deserializedMessage.ID)
	assert.NotNil(t, deserializedMessage.Request)
	assert.Equal(t, message.Request.Type, deserializedMessage.Request.Type)
}

func TestResponse(t *testing.T) {
	response := &Response{
		Success: true,
		Data: map[string]interface{}{
			"job_id": "test-job-123",
			"status": "pending",
		},
		Error: "",
	}

	// Test JSON serialization
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Test JSON deserialization
	var deserializedResponse Response
	err = json.Unmarshal(data, &deserializedResponse)
	require.NoError(t, err)

	assert.Equal(t, response.Success, deserializedResponse.Success)
	assert.NotNil(t, deserializedResponse.Data)
	assert.Equal(t, response.Error, deserializedResponse.Error)
}

func TestDaemonConfig(t *testing.T) {
	config := &DaemonConfig{
		SocketPath: "/test/daemon.sock",
		PIDPath:    "/test/daemon.pid",
		MaxWorkers: 5,
		QueueSize:  200,
		LogLevel:   "debug",
	}

	// Test JSON serialization
	data, err := json.Marshal(config)
	require.NoError(t, err)

	// Test JSON deserialization
	var deserializedConfig DaemonConfig
	err = json.Unmarshal(data, &deserializedConfig)
	require.NoError(t, err)

	assert.Equal(t, config.SocketPath, deserializedConfig.SocketPath)
	assert.Equal(t, config.PIDPath, deserializedConfig.PIDPath)
	assert.Equal(t, config.MaxWorkers, deserializedConfig.MaxWorkers)
	assert.Equal(t, config.QueueSize, deserializedConfig.QueueSize)
	assert.Equal(t, config.LogLevel, deserializedConfig.LogLevel)
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	assert.NotNil(t, config)
	assert.NotEmpty(t, config.SocketPath)
	assert.NotEmpty(t, config.PIDPath)
	assert.Equal(t, DefaultMaxWorkers, config.MaxWorkers)
	assert.Equal(t, DefaultQueueSize, config.QueueSize)
	assert.Equal(t, "info", config.LogLevel)

	// Check that paths contain .fman directory
	assert.Contains(t, config.SocketPath, ".fman")
	assert.Contains(t, config.PIDPath, ".fman")
}

func TestNewJob(t *testing.T) {
	options := &scanner.ScanOptions{
		Verbose:   true,
		ForceSudo: false,
	}

	job := NewJob("/test/path", options)

	assert.NotEmpty(t, job.ID)
	assert.Equal(t, "/test/path", job.Path)
	assert.Equal(t, options, job.Options)
	assert.Equal(t, JobStatusPending, job.Status)
	assert.False(t, job.CreatedAt.IsZero())
	assert.Nil(t, job.StartedAt)
	assert.Nil(t, job.CompletedAt)
}

func TestJobIsTerminal(t *testing.T) {
	job := NewJob("/test", &scanner.ScanOptions{})

	// Test non-terminal states
	job.Status = JobStatusPending
	assert.False(t, job.IsTerminal())

	job.Status = JobStatusRunning
	assert.False(t, job.IsTerminal())

	// Test terminal states
	job.Status = JobStatusCompleted
	assert.True(t, job.IsTerminal())

	job.Status = JobStatusFailed
	assert.True(t, job.IsTerminal())

	job.Status = JobStatusCancelled
	assert.True(t, job.IsTerminal())
}

func TestJobDuration(t *testing.T) {
	job := NewJob("/test", &scanner.ScanOptions{})

	// Test with no start time
	assert.Equal(t, time.Duration(0), job.Duration())

	// Test with start time but no completion
	startTime := time.Now().Add(-5 * time.Second)
	job.StartedAt = &startTime
	duration := job.Duration()
	assert.Greater(t, duration, 4*time.Second)
	assert.Less(t, duration, 6*time.Second)

	// Test with completion time
	completedTime := startTime.Add(3 * time.Second)
	job.CompletedAt = &completedTime
	duration = job.Duration()
	assert.Equal(t, 3*time.Second, duration)
}

func TestJobStatusConstants(t *testing.T) {
	// Test that status constants are properly defined
	assert.Equal(t, JobStatus("pending"), JobStatusPending)
	assert.Equal(t, JobStatus("running"), JobStatusRunning)
	assert.Equal(t, JobStatus("completed"), JobStatusCompleted)
	assert.Equal(t, JobStatus("failed"), JobStatusFailed)
	assert.Equal(t, JobStatus("cancelled"), JobStatusCancelled)
}

func TestRequestTypeConstants(t *testing.T) {
	// Test that request type constants are properly defined
	assert.Equal(t, RequestType("scan"), RequestTypeScan)
	assert.Equal(t, RequestType("status"), RequestTypeStatus)
	assert.Equal(t, RequestType("job_status"), RequestTypeJobStatus)
	assert.Equal(t, RequestType("job_list"), RequestTypeJobList)
	assert.Equal(t, RequestType("job_cancel"), RequestTypeJobCancel)
	assert.Equal(t, RequestType("queue_clear"), RequestTypeQueueClear)
	assert.Equal(t, RequestType("shutdown"), RequestTypeShutdown)
}

func TestErrorConstants(t *testing.T) {
	// Test that error constants are properly defined
	assert.NotNil(t, ErrDaemonNotRunning)
	assert.NotNil(t, ErrDaemonAlreadyRunning)
	assert.NotNil(t, ErrJobNotFound)
	assert.NotNil(t, ErrInvalidRequest)
	assert.NotNil(t, ErrSocketExists)
	assert.NotNil(t, ErrConnectionFailed)

	// Test error messages
	assert.Contains(t, ErrDaemonNotRunning.Error(), "not running")
	assert.Contains(t, ErrDaemonAlreadyRunning.Error(), "already running")
	assert.Contains(t, ErrJobNotFound.Error(), "not found")
}
