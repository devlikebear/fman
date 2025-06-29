package cmd

import (
	"testing"

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
