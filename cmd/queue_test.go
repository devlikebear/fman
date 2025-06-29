package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQueueCommands(t *testing.T) {
	t.Run("queue command exists", func(t *testing.T) {
		assert.NotNil(t, queueCmd)
		assert.Equal(t, "queue", queueCmd.Use)
	})

	t.Run("queue functions exist", func(t *testing.T) {
		assert.NotNil(t, runQueueList)
		assert.NotNil(t, runQueueStatus)
		assert.NotNil(t, runQueueCancel)
		assert.NotNil(t, runQueueClear)
	})

	t.Run("format functions exist", func(t *testing.T) {
		assert.NotNil(t, formatJobStatus)
		assert.NotNil(t, formatJobDuration)
	})
}

func TestRunQueueCommands(t *testing.T) {
	// Set short timeout for quick test execution
	if testing.Short() {
		t.Skip("Skipping queue tests in short mode")
	}

	// 테스트 환경 변수 설정으로 빠른 실패 유도
	oldValue := os.Getenv("FMAN_TEST_MODE")
	os.Setenv("FMAN_TEST_MODE", "1")
	defer func() {
		if oldValue == "" {
			os.Unsetenv("FMAN_TEST_MODE")
		} else {
			os.Setenv("FMAN_TEST_MODE", oldValue)
		}
	}()

	t.Run("runQueueList", func(t *testing.T) {
		// Test the function quickly fails with connection error
		start := time.Now()
		err := runQueueList(queueCmd, []string{})
		duration := time.Since(start)
		
		// Should fail due to daemon connection, but quickly (within 500ms)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to daemon")
		assert.Less(t, duration, 500*time.Millisecond, "Test should complete within 500ms")
	})

	t.Run("runQueueStatus", func(t *testing.T) {
		start := time.Now()
		err := runQueueStatus(queueCmd, []string{"test-job-id"})
		duration := time.Since(start)
		
		// Should fail due to daemon connection, but quickly
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to daemon")
		assert.Less(t, duration, 500*time.Millisecond, "Test should complete within 500ms")
	})

	t.Run("runQueueCancel", func(t *testing.T) {
		start := time.Now()
		err := runQueueCancel(queueCmd, []string{"1"})
		duration := time.Since(start)
		
		// Should fail due to daemon connection, but quickly
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to daemon")
		assert.Less(t, duration, 500*time.Millisecond, "Test should complete within 500ms")
	})

	t.Run("runQueueClear", func(t *testing.T) {
		start := time.Now()
		err := runQueueClear(queueCmd, []string{})
		duration := time.Since(start)
		
		// Should fail due to daemon connection, but quickly
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to daemon")
		assert.Less(t, duration, 500*time.Millisecond, "Test should complete within 500ms")
	})
}
