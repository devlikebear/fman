package rules

import (
	"testing"

	"github.com/devlikebear/fman/internal/db"
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

func TestEvaluatorConditions(t *testing.T) {
	evaluator := NewEvaluator(false)
	
	t.Run("evaluateExtension", func(t *testing.T) {
		file := db.File{Path: "/test/file.txt"}
		
		// Test equal condition
		condition := Condition{
			Type:     ConditionExtension,
			Operator: OpEqual,
			Value:    ".txt",
		}
		result, err := evaluator.evaluateExtension(condition, file)
		assert.NoError(t, err)
		assert.True(t, result)
		
		// Test not equal condition
		condition.Operator = OpNotEqual
		condition.Value = ".pdf"
		result, err = evaluator.evaluateExtension(condition, file)
		assert.NoError(t, err)
		assert.True(t, result)
		
		// Test unsupported operator
		condition.Operator = OpGreaterThan
		result, err = evaluator.evaluateExtension(condition, file)
		assert.Error(t, err)
		assert.False(t, result)
	})
	
	t.Run("evaluateSize", func(t *testing.T) {
		file := db.File{Path: "/test/file.txt", Size: 1024}
		
		// Test greater than
		condition := Condition{
			Type:     ConditionSize,
			Operator: OpGreaterThan,
			Value:    "500",
		}
		result, err := evaluator.evaluateSize(condition, file)
		assert.NoError(t, err)
		assert.True(t, result)
		
		// Test less than
		condition.Operator = OpLessThan
		condition.Value = "2048"
		result, err = evaluator.evaluateSize(condition, file)
		assert.NoError(t, err)
		assert.True(t, result)
		
		// Test invalid size value
		condition.Value = "invalid"
		result, err = evaluator.evaluateSize(condition, file)
		assert.Error(t, err)
		assert.False(t, result)
	})
}
