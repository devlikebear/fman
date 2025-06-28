package rules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRuleValidation(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		rule := Rule{
			Name:        "test-rule",
			Description: "Test rule",
			Enabled:     true,
			Conditions: []Condition{
				{
					Type:     ConditionNamePattern,
					Operator: OpContains,
					Value:    "test",
				},
			},
			Actions: []Action{
				{
					Type:        ActionMove,
					Destination: "/tmp/test/",
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.Equal(t, "test-rule", rule.Name)
		assert.True(t, rule.Enabled)
		assert.Len(t, rule.Conditions, 1)
		assert.Len(t, rule.Actions, 1)
	})
}

func TestConditionTypes(t *testing.T) {
	tests := []struct {
		name     string
		condType ConditionType
		expected string
	}{
		{"name pattern", ConditionNamePattern, "name_pattern"},
		{"extension", ConditionExtension, "extension"},
		{"size", ConditionSize, "size"},
		{"age", ConditionAge, "age"},
		{"modified", ConditionModified, "modified"},
		{"path", ConditionPath, "path"},
		{"file type", ConditionFileType, "file_type"},
		{"mime type", ConditionMimeType, "mime_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.condType))
		})
	}
}

func TestActionTypes(t *testing.T) {
	tests := []struct {
		name     string
		actType  ActionType
		expected string
	}{
		{"move", ActionMove, "move"},
		{"copy", ActionCopy, "copy"},
		{"delete", ActionDelete, "delete"},
		{"rename", ActionRename, "rename"},
		{"link", ActionLink, "link"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.actType))
		})
	}
}

func TestOperatorConstants(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		expected string
	}{
		{"equal", OpEqual, "=="},
		{"not equal", OpNotEqual, "!="},
		{"greater than", OpGreaterThan, ">"},
		{"less than", OpLessThan, "<"},
		{"greater than or equal", OpGreaterThanOrEqual, ">="},
		{"less than or equal", OpLessThanOrEqual, "<="},
		{"contains", OpContains, "contains"},
		{"matches", OpMatches, "matches"},
		{"starts with", OpStartsWith, "starts_with"},
		{"ends with", OpEndsWith, "ends_with"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.operator)
		})
	}
}

func TestRulesConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := RulesConfig{
			Version: DefaultVersion,
			Rules:   []Rule{},
		}

		assert.Equal(t, "1.0", config.Version)
		assert.Empty(t, config.Rules)
	})
}

func TestExecutionResult(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		rule := &Rule{Name: "test-rule"}
		result := ExecutionResult{
			Rule:    rule,
			Success: true,
			Actions: []ActionResult{
				{
					Success: true,
				},
			},
		}

		assert.True(t, result.Success)
		assert.Equal(t, "test-rule", result.Rule.Name)
		assert.Len(t, result.Actions, 1)
		assert.True(t, result.Actions[0].Success)
	})

	t.Run("failed execution", func(t *testing.T) {
		rule := &Rule{Name: "test-rule"}
		result := ExecutionResult{
			Rule:    rule,
			Success: false,
			Error:   assert.AnError,
		}

		assert.False(t, result.Success)
		assert.Error(t, result.Error)
	})
}

func TestExecutionSummary(t *testing.T) {
	t.Run("summary calculation", func(t *testing.T) {
		summary := ExecutionSummary{
			TotalFiles:        10,
			ProcessedFiles:    8,
			SuccessfulActions: 15,
			FailedActions:     2,
			SkippedActions:    1,
			Duration:          time.Second * 5,
		}

		assert.Equal(t, 10, summary.TotalFiles)
		assert.Equal(t, 8, summary.ProcessedFiles)
		assert.Equal(t, 15, summary.SuccessfulActions)
		assert.Equal(t, 2, summary.FailedActions)
		assert.Equal(t, 1, summary.SkippedActions)
		assert.Equal(t, time.Second*5, summary.Duration)
	})
}
