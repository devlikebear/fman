package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEvaluator(t *testing.T) {
	t.Run("create evaluator", func(t *testing.T) {
		evaluator := NewEvaluator(false)
		assert.NotNil(t, evaluator)
	})
}

func TestNewExecutor(t *testing.T) {
	t.Run("create executor with settings", func(t *testing.T) {
		executor := NewExecutor(true, false, true)
		assert.NotNil(t, executor)

		dryRun, verbose, confirm := executor.GetExecutorSettings()
		assert.True(t, dryRun)
		assert.False(t, verbose)
		assert.True(t, confirm)
	})
}

func TestValidateActionType(t *testing.T) {
	t.Run("valid action types", func(t *testing.T) {
		assert.True(t, ValidateActionType(ActionMove))
		assert.True(t, ValidateActionType(ActionCopy))
		assert.True(t, ValidateActionType(ActionDelete))
	})

	t.Run("invalid action type", func(t *testing.T) {
		assert.False(t, ValidateActionType(ActionType("invalid")))
	})
}
