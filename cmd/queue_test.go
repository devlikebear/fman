package cmd

import (
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/daemon"
	"github.com/stretchr/testify/assert"
)

func TestQueueCommands(t *testing.T) {
	t.Run("queue command exists", func(t *testing.T) {
		assert.NotNil(t, queueCmd, "queue command should be defined")
		assert.Equal(t, "queue", queueCmd.Use)
		assert.NotEmpty(t, queueCmd.Short)
	})

	t.Run("queue subcommands exist", func(t *testing.T) {
		subcommands := []string{"list", "status", "cancel", "clear"}

		for _, subcmdName := range subcommands {
			found := false
			for _, subcmd := range queueCmd.Commands() {
				if subcmd.Use == subcmdName || subcmd.Use == subcmdName+" <job-id>" {
					found = true
					break
				}
			}
			assert.True(t, found, "subcommand %s should exist", subcmdName)
		}
	})

	t.Run("queue list command", func(t *testing.T) {
		assert.NotNil(t, queueListCmd)
		assert.Equal(t, "list", queueListCmd.Use)
		assert.NotNil(t, queueListCmd.RunE)
	})

	t.Run("queue status command", func(t *testing.T) {
		assert.NotNil(t, queueStatusCmd)
		assert.Equal(t, "status <job-id>", queueStatusCmd.Use)
		assert.NotNil(t, queueStatusCmd.RunE)
		// Test Args validation function
		err := queueStatusCmd.Args(queueStatusCmd, []string{"job-id"})
		assert.NoError(t, err, "should accept exactly one argument")
		err = queueStatusCmd.Args(queueStatusCmd, []string{})
		assert.Error(t, err, "should reject no arguments")
		err = queueStatusCmd.Args(queueStatusCmd, []string{"job1", "job2"})
		assert.Error(t, err, "should reject multiple arguments")
	})

	t.Run("queue cancel command", func(t *testing.T) {
		assert.NotNil(t, queueCancelCmd)
		assert.Equal(t, "cancel <job-id>", queueCancelCmd.Use)
		assert.NotNil(t, queueCancelCmd.RunE)
		// Test Args validation function
		err := queueCancelCmd.Args(queueCancelCmd, []string{"job-id"})
		assert.NoError(t, err, "should accept exactly one argument")
		err = queueCancelCmd.Args(queueCancelCmd, []string{})
		assert.Error(t, err, "should reject no arguments")
		err = queueCancelCmd.Args(queueCancelCmd, []string{"job1", "job2"})
		assert.Error(t, err, "should reject multiple arguments")
	})

	t.Run("queue clear command", func(t *testing.T) {
		assert.NotNil(t, queueClearCmd)
		assert.Equal(t, "clear", queueClearCmd.Use)
		assert.NotNil(t, queueClearCmd.RunE)
	})
}

func TestFormatJobStatus(t *testing.T) {
	tests := []struct {
		status   daemon.JobStatus
		expected string
	}{
		{daemon.JobStatusPending, "⏳ Pending"},
		{daemon.JobStatusRunning, "🔄 Running"},
		{daemon.JobStatusCompleted, "✅ Completed"},
		{daemon.JobStatusFailed, "❌ Failed"},
		{daemon.JobStatusCancelled, "⛔ Cancelled"},
		{"unknown", "unknown"},
	}

	for _, test := range tests {
		result := formatJobStatus(test.status)
		assert.Equal(t, test.expected, result)
	}
}

func TestFormatJobDuration(t *testing.T) {
	now := time.Now()

	t.Run("no start time", func(t *testing.T) {
		job := &daemon.Job{}
		result := formatJobDuration(job)
		assert.Equal(t, "-", result)
	})

	t.Run("running job", func(t *testing.T) {
		startTime := now.Add(-30 * time.Second)
		job := &daemon.Job{
			StartedAt: &startTime,
		}
		result := formatJobDuration(job)
		assert.Contains(t, result, "s") // Should contain seconds
	})

	t.Run("completed job - seconds", func(t *testing.T) {
		startTime := now.Add(-30 * time.Second)
		endTime := now
		job := &daemon.Job{
			StartedAt:   &startTime,
			CompletedAt: &endTime,
		}
		result := formatJobDuration(job)
		assert.Equal(t, "30s", result)
	})

	t.Run("completed job - minutes", func(t *testing.T) {
		startTime := now.Add(-90 * time.Second)
		endTime := now
		job := &daemon.Job{
			StartedAt:   &startTime,
			CompletedAt: &endTime,
		}
		result := formatJobDuration(job)
		assert.Equal(t, "1m30s", result)
	})

	t.Run("completed job - hours", func(t *testing.T) {
		startTime := now.Add(-3*time.Hour - 30*time.Minute)
		endTime := now
		job := &daemon.Job{
			StartedAt:   &startTime,
			CompletedAt: &endTime,
		}
		result := formatJobDuration(job)
		assert.Equal(t, "3h30m", result)
	})
}

func TestRunQueueList_FunctionExists(t *testing.T) {
	// Test that runQueueList function can be called
	// This will fail because no daemon is running, but it tests the function exists
	err := runQueueList(queueListCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to daemon")
}

func TestRunQueueStatus_FunctionExists(t *testing.T) {
	// Test that runQueueStatus function can be called
	// This will fail because no daemon is running, but it tests the function exists
	err := runQueueStatus(queueStatusCmd, []string{"test-job-id"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to daemon")
}

func TestRunQueueCancel_FunctionExists(t *testing.T) {
	// Test that runQueueCancel function can be called
	// This will fail because no daemon is running, but it tests the function exists
	err := runQueueCancel(queueCancelCmd, []string{"test-job-id"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to daemon")
}

func TestRunQueueClear_FunctionExists(t *testing.T) {
	// Test that runQueueClear function can be called
	// This will fail because no daemon is running, but it tests the function exists
	err := runQueueClear(queueClearCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to daemon")
}

func TestQueueCommandDescriptions(t *testing.T) {
	t.Run("queue has description", func(t *testing.T) {
		assert.NotEmpty(t, queueCmd.Short)
		assert.NotEmpty(t, queueCmd.Long)
	})

	t.Run("queue list has description", func(t *testing.T) {
		assert.NotEmpty(t, queueListCmd.Short)
		assert.NotEmpty(t, queueListCmd.Long)
	})

	t.Run("queue status has description", func(t *testing.T) {
		assert.NotEmpty(t, queueStatusCmd.Short)
		assert.NotEmpty(t, queueStatusCmd.Long)
	})

	t.Run("queue cancel has description", func(t *testing.T) {
		assert.NotEmpty(t, queueCancelCmd.Short)
		assert.NotEmpty(t, queueCancelCmd.Long)
	})

	t.Run("queue clear has description", func(t *testing.T) {
		assert.NotEmpty(t, queueClearCmd.Short)
		assert.NotEmpty(t, queueClearCmd.Long)
	})
}
