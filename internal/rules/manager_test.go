package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Run("creates manager with correct path", func(t *testing.T) {
		configDir := "/tmp/test-config"
		manager := NewManager(configDir)

		expectedPath := filepath.Join(configDir, DefaultRulesFile)
		assert.Equal(t, expectedPath, manager.GetConfigPath())
	})
}

func TestManagerLoadRules(t *testing.T) {
	t.Run("creates default config when file doesn't exist", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		rules := manager.GetRules()
		assert.Empty(t, rules)
	})

	t.Run("loads existing config", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create manager and add a rule
		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		rule := Rule{
			Name:        "test-rule",
			Description: "Test rule",
			Enabled:     true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err = manager.AddRule(rule)
		require.NoError(t, err)

		// Create new manager and load
		manager2 := NewManager(tempDir)
		err = manager2.LoadRules()
		require.NoError(t, err)

		rules := manager2.GetRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, "test-rule", rules[0].Name)
	})
}

func TestManagerAddRule(t *testing.T) {
	t.Run("adds valid rule", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		rule := Rule{
			Name:        "test-rule",
			Description: "Test rule",
			Enabled:     true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err = manager.AddRule(rule)
		require.NoError(t, err)

		rules := manager.GetRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, "test-rule", rules[0].Name)
		assert.False(t, rules[0].CreatedAt.IsZero())
		assert.False(t, rules[0].UpdatedAt.IsZero())
	})

	t.Run("prevents duplicate rule names", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		rule := Rule{
			Name:        "test-rule",
			Description: "Test rule",
			Enabled:     true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err = manager.AddRule(rule)
		require.NoError(t, err)

		// Try to add same rule again
		err = manager.AddRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestManagerUpdateRule(t *testing.T) {
	t.Run("updates existing rule", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		// Add initial rule
		rule := Rule{
			Name:        "test-rule",
			Description: "Original description",
			Enabled:     true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err = manager.AddRule(rule)
		require.NoError(t, err)

		// Update rule
		updatedRule := rule
		updatedRule.Description = "Updated description"
		updatedRule.Enabled = false

		err = manager.UpdateRule("test-rule", updatedRule)
		require.NoError(t, err)

		// Verify update
		retrievedRule, err := manager.GetRule("test-rule")
		require.NoError(t, err)
		assert.Equal(t, "Updated description", retrievedRule.Description)
		assert.False(t, retrievedRule.Enabled)
		assert.False(t, retrievedRule.UpdatedAt.IsZero())
	})

	t.Run("fails for non-existent rule", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		rule := Rule{Name: "non-existent"}
		err = manager.UpdateRule("non-existent", rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestManagerRemoveRule(t *testing.T) {
	t.Run("removes existing rule", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		// Add rule
		rule := Rule{
			Name:        "test-rule",
			Description: "Test rule",
			Enabled:     true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err = manager.AddRule(rule)
		require.NoError(t, err)

		// Remove rule
		err = manager.RemoveRule("test-rule")
		require.NoError(t, err)

		// Verify removal
		rules := manager.GetRules()
		assert.Empty(t, rules)
	})

	t.Run("fails for non-existent rule", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		err = manager.RemoveRule("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestManagerEnableDisableRule(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fman-rules-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	err = manager.LoadRules()
	require.NoError(t, err)

	// Add rule
	rule := Rule{
		Name:        "test-rule",
		Description: "Test rule",
		Enabled:     false,
		Conditions: []Condition{
			{Type: ConditionNamePattern, Value: "test"},
		},
		Actions: []Action{
			{Type: ActionMove, Destination: "/tmp/"},
		},
	}

	err = manager.AddRule(rule)
	require.NoError(t, err)

	t.Run("enable rule", func(t *testing.T) {
		err = manager.EnableRule("test-rule")
		require.NoError(t, err)

		retrievedRule, err := manager.GetRule("test-rule")
		require.NoError(t, err)
		assert.True(t, retrievedRule.Enabled)
	})

	t.Run("disable rule", func(t *testing.T) {
		err = manager.DisableRule("test-rule")
		require.NoError(t, err)

		retrievedRule, err := manager.GetRule("test-rule")
		require.NoError(t, err)
		assert.False(t, retrievedRule.Enabled)
	})
}

func TestManagerGetEnabledRules(t *testing.T) {
	t.Run("returns only enabled rules", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		// Add enabled rule
		enabledRule := Rule{
			Name:        "enabled-rule",
			Description: "Enabled rule",
			Enabled:     true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		// Add disabled rule
		disabledRule := Rule{
			Name:        "disabled-rule",
			Description: "Disabled rule",
			Enabled:     false,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err = manager.AddRule(enabledRule)
		require.NoError(t, err)
		err = manager.AddRule(disabledRule)
		require.NoError(t, err)

		enabledRules := manager.GetEnabledRules()
		assert.Len(t, enabledRules, 1)
		assert.Equal(t, "enabled-rule", enabledRules[0].Name)
	})
}

func TestManagerValidateRule(t *testing.T) {
	manager := NewManager("/tmp")

	t.Run("valid rule passes validation", func(t *testing.T) {
		rule := Rule{
			Name:        "valid-rule",
			Description: "Valid rule",
			Enabled:     true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err := manager.ValidateRule(rule)
		assert.NoError(t, err)
	})

	t.Run("rule without name fails", func(t *testing.T) {
		rule := Rule{
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err := manager.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("rule without conditions fails", func(t *testing.T) {
		rule := Rule{
			Name: "test-rule",
			Actions: []Action{
				{Type: ActionMove, Destination: "/tmp/"},
			},
		}

		err := manager.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one condition")
	})

	t.Run("rule without actions fails", func(t *testing.T) {
		rule := Rule{
			Name: "test-rule",
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
		}

		err := manager.ValidateRule(rule)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one action")
	})
}

func TestManagerCreateExampleRules(t *testing.T) {
	t.Run("creates example rules", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		err = manager.CreateExampleRules()
		require.NoError(t, err)

		rules := manager.GetRules()
		assert.NotEmpty(t, rules)

		// Check that example rules are disabled by default
		for _, rule := range rules {
			assert.False(t, rule.Enabled, "Example rule '%s' should be disabled by default", rule.Name)
		}
	})

	t.Run("doesn't duplicate existing rules", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fman-rules-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager := NewManager(tempDir)
		err = manager.LoadRules()
		require.NoError(t, err)

		// Create examples twice
		err = manager.CreateExampleRules()
		require.NoError(t, err)

		initialCount := len(manager.GetRules())

		err = manager.CreateExampleRules()
		require.NoError(t, err)

		finalCount := len(manager.GetRules())
		assert.Equal(t, initialCount, finalCount, "Should not create duplicate example rules")
	})
}
