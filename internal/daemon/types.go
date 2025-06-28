/*
Copyright © 2025 changheonshin
*/
package daemon

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/devlikebear/fman/internal/scanner"
)

// Constants for daemon configuration
const (
	// DefaultSocketPath is the default Unix socket path
	DefaultSocketPath = "daemon.sock"
	// DefaultPIDPath is the default PID file path
	DefaultPIDPath = "daemon.pid"
	// DefaultMaxWorkers is the default number of worker goroutines
	DefaultMaxWorkers = 2
	// DefaultQueueSize is the default queue buffer size
	DefaultQueueSize = 100
)

// JobStatus represents the status of a scan job
type JobStatus string

const (
	// JobStatusPending indicates the job is waiting to be processed
	JobStatusPending JobStatus = "pending"
	// JobStatusRunning indicates the job is currently being processed
	JobStatusRunning JobStatus = "running"
	// JobStatusCompleted indicates the job has completed successfully
	JobStatusCompleted JobStatus = "completed"
	// JobStatusFailed indicates the job has failed
	JobStatusFailed JobStatus = "failed"
	// JobStatusCancelled indicates the job was cancelled
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents a scan job in the queue
type Job struct {
	ID          string               `json:"id"`
	Path        string               `json:"path"`
	Options     *scanner.ScanOptions `json:"options"`
	Status      JobStatus            `json:"status"`
	CreatedAt   time.Time            `json:"created_at"`
	StartedAt   *time.Time           `json:"started_at,omitempty"`
	CompletedAt *time.Time           `json:"completed_at,omitempty"`
	Stats       *scanner.ScanStats   `json:"stats,omitempty"`
	Error       string               `json:"error,omitempty"`
	Progress    *JobProgress         `json:"progress,omitempty"`
}

// JobProgress represents the progress of a running job
type JobProgress struct {
	FilesProcessed int    `json:"files_processed"`
	TotalFiles     int    `json:"total_files,omitempty"`
	CurrentPath    string `json:"current_path,omitempty"`
}

// ScanRequest represents a request to scan a directory
type ScanRequest struct {
	Path    string               `json:"path"`
	Options *scanner.ScanOptions `json:"options"`
}

// DaemonStatus represents the current status of the daemon
type DaemonStatus struct {
	Running       bool      `json:"running"`
	PID           int       `json:"pid"`
	StartedAt     time.Time `json:"started_at"`
	ActiveJobs    int       `json:"active_jobs"`
	QueuedJobs    int       `json:"queued_jobs"`
	CompletedJobs int       `json:"completed_jobs"`
	FailedJobs    int       `json:"failed_jobs"`
	Workers       int       `json:"workers"`
}

// MessageType represents the type of message being sent
type MessageType string

const (
	// MessageTypeRequest indicates a request message
	MessageTypeRequest MessageType = "request"
	// MessageTypeResponse indicates a response message
	MessageTypeResponse MessageType = "response"
	// MessageTypeNotification indicates a notification message
	MessageTypeNotification MessageType = "notification"
)

// RequestType represents the type of request
type RequestType string

const (
	// RequestTypeScan requests a scan operation
	RequestTypeScan RequestType = "scan"
	// RequestTypeStatus requests daemon status
	RequestTypeStatus RequestType = "status"
	// RequestTypeJobStatus requests job status
	RequestTypeJobStatus RequestType = "job_status"
	// RequestTypeJobList requests list of jobs
	RequestTypeJobList RequestType = "job_list"
	// RequestTypeJobCancel requests job cancellation
	RequestTypeJobCancel RequestType = "job_cancel"
	// RequestTypeQueueClear requests queue clearing
	RequestTypeQueueClear RequestType = "queue_clear"
	// RequestTypeShutdown requests daemon shutdown
	RequestTypeShutdown RequestType = "shutdown"
)

// Message represents a message in the communication protocol
type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	Request   *Request    `json:"request,omitempty"`
	Response  *Response   `json:"response,omitempty"`
}

// Request represents a request message
type Request struct {
	Type RequestType `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// Response represents a response message
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// DaemonConfig represents daemon configuration
type DaemonConfig struct {
	SocketPath string `json:"socket_path" yaml:"socket_path" mapstructure:"socket_path"`
	PIDPath    string `json:"pid_path" yaml:"pid_path" mapstructure:"pid_path"`
	MaxWorkers int    `json:"max_workers" yaml:"max_workers" mapstructure:"max_workers"`
	QueueSize  int    `json:"queue_size" yaml:"queue_size" mapstructure:"queue_size"`
	LogLevel   string `json:"log_level" yaml:"log_level" mapstructure:"log_level"`
}

// Common errors
var (
	// ErrDaemonNotRunning indicates the daemon is not running
	ErrDaemonNotRunning = errors.New("daemon is not running")
	// ErrDaemonAlreadyRunning indicates the daemon is already running
	ErrDaemonAlreadyRunning = errors.New("daemon is already running")
	// ErrJobNotFound indicates the requested job was not found
	ErrJobNotFound = errors.New("job not found")
	// ErrInvalidRequest indicates the request is invalid
	ErrInvalidRequest = errors.New("invalid request")
	// ErrSocketExists indicates the socket file already exists
	ErrSocketExists = errors.New("socket file already exists")
	// ErrConnectionFailed indicates connection to daemon failed
	ErrConnectionFailed = errors.New("failed to connect to daemon")
)

// DaemonInterface defines the interface for daemon operations
type DaemonInterface interface {
	// Start starts the daemon
	Start(ctx context.Context) error
	// Stop stops the daemon gracefully
	Stop() error
	// Status returns the current daemon status
	Status() (*DaemonStatus, error)
	// IsRunning checks if the daemon is running
	IsRunning() bool
	// EnqueueScan adds a scan job to the queue
	EnqueueScan(request *ScanRequest) (*Job, error)
	// GetJob retrieves a job by ID
	GetJob(jobID string) (*Job, error)
	// CancelJob cancels a job
	CancelJob(jobID string) error
	// ListJobs returns all jobs with optional status filter
	ListJobs(status JobStatus) ([]*Job, error)
	// ClearQueue clears all pending jobs
	ClearQueue() error
}

// QueueInterface defines the interface for job queue operations
type QueueInterface interface {
	// Add adds a job to the queue
	Add(job *Job) error
	// Next gets the next job from the queue (blocking)
	Next(ctx context.Context) (*Job, error)
	// Get retrieves a job by ID
	Get(jobID string) (*Job, error)
	// Update updates a job's status and data
	Update(job *Job) error
	// List returns jobs with optional status filter
	List(status JobStatus) ([]*Job, error)
	// Cancel marks a job as cancelled
	Cancel(jobID string) error
	// Clear removes all pending jobs
	Clear() error
	// Size returns the current queue size
	Size() int
	// Stats returns queue statistics
	Stats() map[string]int
}

// ClientInterface defines the interface for daemon client operations
type ClientInterface interface {
	// Connect connects to the daemon
	Connect() error
	// Disconnect disconnects from the daemon
	Disconnect() error
	// SendRequest sends a request and waits for response
	SendRequest(req *Request) (*Response, error)
	// IsConnected checks if connected to daemon
	IsConnected() bool
}

// GetDefaultConfig returns the default daemon configuration
func GetDefaultConfig() *DaemonConfig {
	homeDir, _ := os.UserHomeDir()
	fmanDir := filepath.Join(homeDir, ".fman")

	return &DaemonConfig{
		SocketPath: filepath.Join(fmanDir, DefaultSocketPath),
		PIDPath:    filepath.Join(fmanDir, DefaultPIDPath),
		MaxWorkers: DefaultMaxWorkers,
		QueueSize:  DefaultQueueSize,
		LogLevel:   "info",
	}
}

// NewJob creates a new job with the given parameters
func NewJob(path string, options *scanner.ScanOptions) *Job {
	return &Job{
		ID:        generateJobID(),
		Path:      path,
		Options:   options,
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
	}
}

// generateJobID generates a unique job ID using UUID-like format
func generateJobID() string {
	// Generate 16 random bytes
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), os.Getpid())
	}

	// Format as UUID-like string
	return fmt.Sprintf("job_%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// IsTerminal returns true if the job status is terminal (completed, failed, or cancelled)
func (j *Job) IsTerminal() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed || j.Status == JobStatusCancelled
}

// Duration returns the job duration if completed, or duration so far if running
func (j *Job) Duration() time.Duration {
	if j.StartedAt == nil {
		return 0
	}

	endTime := time.Now()
	if j.CompletedAt != nil {
		endTime = *j.CompletedAt
	}

	return endTime.Sub(*j.StartedAt)
}
