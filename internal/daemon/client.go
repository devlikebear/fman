/*
Copyright © 2025 changheonshin
*/
package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// DaemonClient implements ClientInterface for communicating with daemon
type DaemonClient struct {
	config     *DaemonConfig
	conn       net.Conn
	connected  bool
	timeout    time.Duration
	retryCount int
}

// NewDaemonClient creates a new daemon client
func NewDaemonClient(config *DaemonConfig) *DaemonClient {
	if config == nil {
		config = GetDefaultConfig()
	}

	return &DaemonClient{
		config:     config,
		timeout:    30 * time.Second,
		retryCount: 3,
	}
}

// SetTimeout sets the connection timeout
func (c *DaemonClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// SetRetryCount sets the number of connection retries
func (c *DaemonClient) SetRetryCount(count int) {
	c.retryCount = count
}

// Connect connects to the daemon
func (c *DaemonClient) Connect() error {
	if c.connected && c.conn != nil {
		return nil
	}

	socketPath := c.getSocketPath()

	// Try to connect with retries
	var lastErr error
	for i := 0; i <= c.retryCount; i++ {
		conn, err := net.DialTimeout("unix", socketPath, c.timeout)
		if err == nil {
			c.conn = conn
			c.connected = true
			return nil
		}
		lastErr = err

		// If first attempt fails, try to start daemon
		if i == 0 && !c.IsDaemonRunning() {
			if startErr := c.StartDaemon(); startErr != nil {
				return fmt.Errorf("failed to start daemon: %w", startErr)
			}
			// Wait a bit for daemon to start
			time.Sleep(2 * time.Second)
			continue
		}

		if i < c.retryCount {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	return fmt.Errorf("failed to connect after %d retries: %w", c.retryCount, lastErr)
}

// Disconnect disconnects from the daemon
func (c *DaemonClient) Disconnect() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.connected = false
		return err
	}
	c.connected = false
	return nil
}

// IsConnected checks if connected to daemon
func (c *DaemonClient) IsConnected() bool {
	return c.connected && c.conn != nil
}

// SendRequest sends a request and waits for response
func (c *DaemonClient) SendRequest(req *Request) (*Response, error) {
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Create message with unique ID
	msg := &Message{
		Type:      MessageTypeRequest,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Request:   req,
	}

	// Send request
	if err := c.sendMessage(msg); err != nil {
		c.Disconnect() // Reset connection on error
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	respMsg, err := c.receiveMessage()
	if err != nil {
		c.Disconnect() // Reset connection on error
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Validate response
	if respMsg.Type != MessageTypeResponse {
		return nil, fmt.Errorf("unexpected message type: %s", respMsg.Type)
	}

	if respMsg.ID != msg.ID {
		return nil, fmt.Errorf("response ID mismatch: expected %s, got %s", msg.ID, respMsg.ID)
	}

	if respMsg.Response == nil {
		return nil, fmt.Errorf("missing response data")
	}

	return respMsg.Response, nil
}

// IsDaemonRunning checks if daemon is running
func (c *DaemonClient) IsDaemonRunning() bool {
	pidPath := c.getPIDPath()

	// Check if PID file exists
	pidData, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}

	// Parse PID
	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process is alive
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// StartDaemon starts the daemon in background
func (c *DaemonClient) StartDaemon() error {
	if c.IsDaemonRunning() {
		return ErrDaemonAlreadyRunning
	}

	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create daemon start command
	cmd := exec.Command(executable, "daemon", "start", "--background")

	// Set process attributes for background execution
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session
	}

	// Redirect stdout/stderr to prevent hanging
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start daemon process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Wait for daemon to be ready
	return c.waitForDaemon(10 * time.Second)
}

// StopDaemon stops the daemon
func (c *DaemonClient) StopDaemon() error {
	req := &Request{
		Type: RequestTypeShutdown,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return fmt.Errorf("failed to send shutdown request: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("shutdown request failed: %s", resp.Error)
	}

	return nil
}

// GetStatus gets daemon status
func (c *DaemonClient) GetStatus() (*DaemonStatus, error) {
	req := &Request{
		Type: RequestTypeStatus,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("status request failed: %s", resp.Error)
	}

	// Parse response data
	statusData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid status response format")
	}

	// Convert to DaemonStatus
	statusJSON, err := json.Marshal(statusData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status data: %w", err)
	}

	var status DaemonStatus
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

// EnqueueScan adds a scan job to the queue
func (c *DaemonClient) EnqueueScan(request *ScanRequest) (*Job, error) {
	req := &Request{
		Type: RequestTypeScan,
		Data: request,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue scan: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("scan request failed: %s", resp.Error)
	}

	// Parse response data
	jobData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid job response format")
	}

	// Convert to Job
	jobJSON, err := json.Marshal(jobData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job data: %w", err)
	}

	var job Job
	if err := json.Unmarshal(jobJSON, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// GetJob retrieves a job by ID
func (c *DaemonClient) GetJob(jobID string) (*Job, error) {
	req := &Request{
		Type: RequestTypeJobStatus,
		Data: jobID,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("job request failed: %s", resp.Error)
	}

	// Parse response data
	jobData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid job response format")
	}

	// Convert to Job
	jobJSON, err := json.Marshal(jobData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job data: %w", err)
	}

	var job Job
	if err := json.Unmarshal(jobJSON, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// CancelJob cancels a job
func (c *DaemonClient) CancelJob(jobID string) error {
	req := &Request{
		Type: RequestTypeJobCancel,
		Data: jobID,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("cancel request failed: %s", resp.Error)
	}

	return nil
}

// ListJobs returns all jobs with optional status filter
func (c *DaemonClient) ListJobs(status JobStatus) ([]*Job, error) {
	req := &Request{
		Type: RequestTypeJobList,
		Data: string(status),
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("list request failed: %s", resp.Error)
	}

	// Parse response data
	jobsData, ok := resp.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid jobs response format")
	}

	// Convert to []*Job
	jobsJSON, err := json.Marshal(jobsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jobs data: %w", err)
	}

	var jobs []*Job
	if err := json.Unmarshal(jobsJSON, &jobs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal jobs: %w", err)
	}

	return jobs, nil
}

// ClearQueue clears all pending jobs
func (c *DaemonClient) ClearQueue() error {
	req := &Request{
		Type: RequestTypeQueueClear,
	}

	resp, err := c.SendRequest(req)
	if err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("clear request failed: %s", resp.Error)
	}

	return nil
}

// sendMessage sends a message over the connection
func (c *DaemonClient) sendMessage(msg *Message) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Encode message as JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send message length first (4 bytes)
	length := uint32(len(data))
	lengthBytes := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
	}

	if _, err := c.conn.Write(lengthBytes); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	// Send message data
	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("failed to write message data: %w", err)
	}

	return nil
}

// receiveMessage receives a message from the connection
func (c *DaemonClient) receiveMessage() (*Message, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read message length (4 bytes)
	lengthBytes := make([]byte, 4)
	if _, err := c.conn.Read(lengthBytes); err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	// Parse length
	length := uint32(lengthBytes[0])<<24 | uint32(lengthBytes[1])<<16 | uint32(lengthBytes[2])<<8 | uint32(lengthBytes[3])
	if length > 1024*1024 { // 1MB limit
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	// Read message data
	data := make([]byte, length)
	if _, err := c.conn.Read(data); err != nil {
		return nil, fmt.Errorf("failed to read message data: %w", err)
	}

	// Decode JSON
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// waitForDaemon waits for daemon to be ready
func (c *DaemonClient) waitForDaemon(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if c.IsDaemonRunning() {
			// Try to connect to verify daemon is ready
			if err := c.Connect(); err == nil {
				c.Disconnect() // Close test connection
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("daemon did not start within %v", timeout)
}

// getSocketPath returns the socket file path
func (c *DaemonClient) getSocketPath() string {
	if filepath.IsAbs(c.config.SocketPath) {
		return c.config.SocketPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return c.config.SocketPath
	}

	return filepath.Join(homeDir, ".fman", c.config.SocketPath)
}

// getPIDPath returns the PID file path
func (c *DaemonClient) getPIDPath() string {
	if filepath.IsAbs(c.config.PIDPath) {
		return c.config.PIDPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return c.config.PIDPath
	}

	return filepath.Join(homeDir, ".fman", c.config.PIDPath)
}
