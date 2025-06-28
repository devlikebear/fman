/*
Copyright © 2025 changheonshin
*/
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/devlikebear/fman/internal/scanner"
	"github.com/spf13/afero"
)

// DaemonServer implements the DaemonInterface using Unix Domain Socket
type DaemonServer struct {
	config     *DaemonConfig
	listener   net.Listener
	queue      QueueInterface
	running    bool
	startedAt  time.Time
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	workers    []*Worker
	workerWg   sync.WaitGroup
	shutdownCh chan struct{}
}

// Worker represents a background worker that processes jobs
type Worker struct {
	id     int
	server *DaemonServer
	ctx    context.Context
	cancel context.CancelFunc
}

// NewDaemonServer creates a new daemon server instance
func NewDaemonServer(config *DaemonConfig, queue QueueInterface) *DaemonServer {
	if config == nil {
		config = GetDefaultConfig()
	}

	return &DaemonServer{
		config:     config,
		queue:      queue,
		shutdownCh: make(chan struct{}),
	}
}

// Start starts the daemon server
func (s *DaemonServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrDaemonAlreadyRunning
	}

	// Check if daemon is already running
	if s.isDaemonRunning() {
		return ErrDaemonAlreadyRunning
	}

	// Create context for cancellation
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Create socket directory if it doesn't exist
	socketDir := filepath.Dir(s.getSocketPath())
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket file if it exists
	if err := s.removeSocketFile(); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", s.getSocketPath())
	if err != nil {
		return fmt.Errorf("failed to create socket listener: %w", err)
	}
	s.listener = listener

	// Set socket file permissions (owner read/write only)
	if err := os.Chmod(s.getSocketPath(), 0600); err != nil {
		s.listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Write PID file
	if err := s.writePIDFile(); err != nil {
		s.listener.Close()
		s.removeSocketFile()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	s.running = true
	s.startedAt = time.Now()

	// Start worker goroutines
	s.startWorkers()

	// Start signal handler
	go s.handleSignals()

	// Start accepting connections
	go s.acceptConnections()

	return nil
}

// Stop stops the daemon gracefully
func (s *DaemonServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrDaemonNotRunning
	}

	// Signal shutdown
	close(s.shutdownCh)

	// Cancel context to stop workers
	if s.cancel != nil {
		s.cancel()
	}

	// Wait for workers to finish
	s.workerWg.Wait()

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Clean up files
	s.removeSocketFile()
	s.removePIDFile()

	s.running = false
	return nil
}

// Status returns the current daemon status
func (s *DaemonServer) Status() (*DaemonStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return nil, ErrDaemonNotRunning
	}

	stats := s.queue.Stats()

	return &DaemonStatus{
		Running:       s.running,
		PID:           os.Getpid(),
		StartedAt:     s.startedAt,
		ActiveJobs:    stats["running"],
		QueuedJobs:    stats["pending"],
		CompletedJobs: stats["completed"],
		FailedJobs:    stats["failed"],
		Workers:       len(s.workers),
	}, nil
}

// IsRunning checks if the daemon is running
func (s *DaemonServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// EnqueueScan adds a scan job to the queue
func (s *DaemonServer) EnqueueScan(request *ScanRequest) (*Job, error) {
	if !s.IsRunning() {
		return nil, ErrDaemonNotRunning
	}

	job := NewJob(request.Path, request.Options)
	if err := s.queue.Add(job); err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	return job, nil
}

// GetJob retrieves a job by ID
func (s *DaemonServer) GetJob(jobID string) (*Job, error) {
	if !s.IsRunning() {
		return nil, ErrDaemonNotRunning
	}

	return s.queue.Get(jobID)
}

// CancelJob cancels a job
func (s *DaemonServer) CancelJob(jobID string) error {
	if !s.IsRunning() {
		return ErrDaemonNotRunning
	}

	return s.queue.Cancel(jobID)
}

// ListJobs returns all jobs with optional status filter
func (s *DaemonServer) ListJobs(status JobStatus) ([]*Job, error) {
	if !s.IsRunning() {
		return nil, ErrDaemonNotRunning
	}

	return s.queue.List(status)
}

// ClearQueue clears all pending jobs
func (s *DaemonServer) ClearQueue() error {
	if !s.IsRunning() {
		return ErrDaemonNotRunning
	}

	return s.queue.Clear()
}

// Private helper methods

func (s *DaemonServer) getSocketPath() string {
	if filepath.IsAbs(s.config.SocketPath) {
		return s.config.SocketPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".fman", s.config.SocketPath)
	}

	return filepath.Join(homeDir, ".fman", s.config.SocketPath)
}

func (s *DaemonServer) getPIDPath() string {
	if filepath.IsAbs(s.config.PIDPath) {
		return s.config.PIDPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".fman", s.config.PIDPath)
	}

	return filepath.Join(homeDir, ".fman", s.config.PIDPath)
}

func (s *DaemonServer) isDaemonRunning() bool {
	pidPath := s.getPIDPath()

	// Check if PID file exists
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}

	// Parse PID
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false
	}

	// Check if process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (s *DaemonServer) writePIDFile() error {
	pidPath := s.getPIDPath()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(pidPath), 0755); err != nil {
		return err
	}

	// Write PID to file
	pid := os.Getpid()
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
}

func (s *DaemonServer) removePIDFile() error {
	pidPath := s.getPIDPath()
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(pidPath)
}

func (s *DaemonServer) removeSocketFile() error {
	socketPath := s.getSocketPath()
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(socketPath)
}

func (s *DaemonServer) startWorkers() {
	s.workers = make([]*Worker, s.config.MaxWorkers)

	for i := 0; i < s.config.MaxWorkers; i++ {
		worker := &Worker{
			id:     i,
			server: s,
		}
		worker.ctx, worker.cancel = context.WithCancel(s.ctx)
		s.workers[i] = worker

		s.workerWg.Add(1)
		go worker.run(&s.workerWg)
	}
}

func (s *DaemonServer) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		s.Stop()
	case <-s.shutdownCh:
		return
	}
}

func (s *DaemonServer) acceptConnections() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

func (s *DaemonServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return
			}
			s.sendErrorResponse(encoder, "invalid_message", fmt.Sprintf("failed to decode message: %v", err))
			return
		}

		response := s.handleMessage(&msg)
		if err := encoder.Encode(response); err != nil {
			return
		}
	}
}

func (s *DaemonServer) handleMessage(msg *Message) *Message {
	response := &Message{
		Type:      MessageTypeResponse,
		ID:        msg.ID,
		Timestamp: time.Now(),
		Response:  &Response{},
	}

	if msg.Request == nil {
		response.Response.Success = false
		response.Response.Error = "missing request"
		return response
	}

	switch msg.Request.Type {
	case RequestTypeScan:
		s.handleScanRequest(msg.Request, response.Response)
	case RequestTypeStatus:
		s.handleStatusRequest(response.Response)
	case RequestTypeJobStatus:
		s.handleJobStatusRequest(msg.Request, response.Response)
	case RequestTypeJobList:
		s.handleJobListRequest(msg.Request, response.Response)
	case RequestTypeJobCancel:
		s.handleJobCancelRequest(msg.Request, response.Response)
	case RequestTypeQueueClear:
		s.handleQueueClearRequest(response.Response)
	case RequestTypeShutdown:
		s.handleShutdownRequest(response.Response)
	default:
		response.Response.Success = false
		response.Response.Error = "unknown request type"
	}

	return response
}

func (s *DaemonServer) handleScanRequest(req *Request, resp *Response) {
	var scanReq ScanRequest
	if err := s.parseRequestData(req.Data, &scanReq); err != nil {
		resp.Success = false
		resp.Error = fmt.Sprintf("invalid scan request: %v", err)
		return
	}

	job, err := s.EnqueueScan(&scanReq)
	if err != nil {
		resp.Success = false
		resp.Error = fmt.Sprintf("failed to enqueue scan: %v", err)
		return
	}

	resp.Success = true
	resp.Data = job
}

func (s *DaemonServer) handleStatusRequest(resp *Response) {
	status, err := s.Status()
	if err != nil {
		resp.Success = false
		resp.Error = fmt.Sprintf("failed to get status: %v", err)
		return
	}

	resp.Success = true
	resp.Data = status
}

func (s *DaemonServer) handleJobStatusRequest(req *Request, resp *Response) {
	jobID, ok := req.Data.(string)
	if !ok {
		resp.Success = false
		resp.Error = "job ID must be a string"
		return
	}

	job, err := s.GetJob(jobID)
	if err != nil {
		resp.Success = false
		resp.Error = fmt.Sprintf("failed to get job: %v", err)
		return
	}

	resp.Success = true
	resp.Data = job
}

func (s *DaemonServer) handleJobListRequest(req *Request, resp *Response) {
	var status JobStatus
	if req.Data != nil {
		if statusStr, ok := req.Data.(string); ok {
			status = JobStatus(statusStr)
		}
	}

	jobs, err := s.ListJobs(status)
	if err != nil {
		resp.Success = false
		resp.Error = fmt.Sprintf("failed to list jobs: %v", err)
		return
	}

	resp.Success = true
	resp.Data = jobs
}

func (s *DaemonServer) handleJobCancelRequest(req *Request, resp *Response) {
	jobID, ok := req.Data.(string)
	if !ok {
		resp.Success = false
		resp.Error = "job ID must be a string"
		return
	}

	err := s.CancelJob(jobID)
	if err != nil {
		resp.Success = false
		resp.Error = fmt.Sprintf("failed to cancel job: %v", err)
		return
	}

	resp.Success = true
}

func (s *DaemonServer) handleQueueClearRequest(resp *Response) {
	err := s.ClearQueue()
	if err != nil {
		resp.Success = false
		resp.Error = fmt.Sprintf("failed to clear queue: %v", err)
		return
	}

	resp.Success = true
}

func (s *DaemonServer) handleShutdownRequest(resp *Response) {
	resp.Success = true

	// Shutdown asynchronously to allow response to be sent
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.Stop()
	}()
}

func (s *DaemonServer) parseRequestData(data interface{}, target interface{}) error {
	if data == nil {
		return fmt.Errorf("missing request data")
	}

	// Convert to JSON and back to parse into target struct
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, target)
}

func (s *DaemonServer) sendErrorResponse(encoder *json.Encoder, code, message string) {
	resp := &Message{
		Type:      MessageTypeResponse,
		Timestamp: time.Now(),
		Response: &Response{
			Success: false,
			Error:   message,
		},
	}
	encoder.Encode(resp)
}

// Worker implementation

func (w *Worker) run(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// Get next job from queue (blocking with timeout)
		jobCtx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
		job, err := w.server.queue.Next(jobCtx)
		cancel()

		if err != nil {
			if w.ctx.Err() != nil {
				return
			}
			continue
		}

		w.processJob(job)
	}
}

func (w *Worker) processJob(job *Job) {
	// Update job status to running
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	w.server.queue.Update(job)

	// Create file scanner
	fs := afero.NewOsFs()
	database := db.NewDatabase(nil)
	scanner := scanner.NewFileScanner(fs, database)

	// Execute scan
	stats, err := scanner.ScanDirectory(w.ctx, job.Path, job.Options)

	// Update job with results
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	job.Stats = stats

	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err.Error()
	} else {
		job.Status = JobStatusCompleted
	}

	w.server.queue.Update(job)
}
