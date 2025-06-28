package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager handles rule configuration management
type Manager struct {
	configPath string
	config     *RulesConfig
}

// NewManager creates a new rules manager
func NewManager(configDir string) *Manager {
	return &Manager{
		configPath: filepath.Join(configDir, DefaultRulesFile),
	}
}

// LoadRules loads rules from the configuration file
func (m *Manager) LoadRules() error {
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Create default configuration if file doesn't exist
		m.config = &RulesConfig{
			Version: DefaultVersion,
			Rules:   []Rule{},
		}
		return m.SaveRules()
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	config := &RulesConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse rules file: %w", err)
	}

	m.config = config
	return nil
}

// SaveRules saves rules to the configuration file
func (m *Manager) SaveRules() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write rules file: %w", err)
	}

	return nil
}

// GetRules returns all rules
func (m *Manager) GetRules() []Rule {
	if m.config == nil {
		return []Rule{}
	}
	return m.config.Rules
}

// GetEnabledRules returns only enabled rules
func (m *Manager) GetEnabledRules() []Rule {
	var enabled []Rule
	for _, rule := range m.GetRules() {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled
}

// GetRule returns a rule by name
func (m *Manager) GetRule(name string) (*Rule, error) {
	for i, rule := range m.config.Rules {
		if rule.Name == name {
			return &m.config.Rules[i], nil
		}
	}
	return nil, fmt.Errorf("rule '%s' not found", name)
}

// AddRule adds a new rule
func (m *Manager) AddRule(rule Rule) error {
	// Check if rule name already exists
	if _, err := m.GetRule(rule.Name); err == nil {
		return fmt.Errorf("rule '%s' already exists", rule.Name)
	}

	// Set timestamps
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	// Add rule
	m.config.Rules = append(m.config.Rules, rule)
	return m.SaveRules()
}

// UpdateRule updates an existing rule
func (m *Manager) UpdateRule(name string, updatedRule Rule) error {
	for i, rule := range m.config.Rules {
		if rule.Name == name {
			// Preserve creation time, update modification time
			updatedRule.CreatedAt = rule.CreatedAt
			updatedRule.UpdatedAt = time.Now()
			updatedRule.Name = name // Ensure name doesn't change

			m.config.Rules[i] = updatedRule
			return m.SaveRules()
		}
	}
	return fmt.Errorf("rule '%s' not found", name)
}

// RemoveRule removes a rule by name
func (m *Manager) RemoveRule(name string) error {
	for i, rule := range m.config.Rules {
		if rule.Name == name {
			// Remove rule from slice
			m.config.Rules = append(m.config.Rules[:i], m.config.Rules[i+1:]...)
			return m.SaveRules()
		}
	}
	return fmt.Errorf("rule '%s' not found", name)
}

// EnableRule enables a rule
func (m *Manager) EnableRule(name string) error {
	rule, err := m.GetRule(name)
	if err != nil {
		return err
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()
	return m.SaveRules()
}

// DisableRule disables a rule
func (m *Manager) DisableRule(name string) error {
	rule, err := m.GetRule(name)
	if err != nil {
		return err
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()
	return m.SaveRules()
}

// ValidateRule validates a rule's configuration
func (m *Manager) ValidateRule(rule Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}

	if len(rule.Actions) == 0 {
		return fmt.Errorf("rule must have at least one action")
	}

	// Validate conditions
	for i, condition := range rule.Conditions {
		if err := m.validateCondition(condition); err != nil {
			return fmt.Errorf("condition %d: %w", i+1, err)
		}
	}

	// Validate actions
	for i, action := range rule.Actions {
		if err := m.validateAction(action); err != nil {
			return fmt.Errorf("action %d: %w", i+1, err)
		}
	}

	return nil
}

// validateCondition validates a single condition
func (m *Manager) validateCondition(condition Condition) error {
	validTypes := map[ConditionType]bool{
		ConditionNamePattern: true,
		ConditionExtension:   true,
		ConditionSize:        true,
		ConditionAge:         true,
		ConditionModified:    true,
		ConditionPath:        true,
		ConditionFileType:    true,
		ConditionMimeType:    true,
	}

	if !validTypes[condition.Type] {
		return fmt.Errorf("invalid condition type: %s", condition.Type)
	}

	if condition.Value == "" {
		return fmt.Errorf("condition value cannot be empty")
	}

	// Validate operators for specific condition types
	switch condition.Type {
	case ConditionSize, ConditionAge:
		validOps := map[string]bool{
			OpGreaterThan: true, OpLessThan: true,
			OpGreaterThanOrEqual: true, OpLessThanOrEqual: true,
			OpEqual: true, OpNotEqual: true,
		}
		if condition.Operator != "" && !validOps[condition.Operator] {
			return fmt.Errorf("invalid operator '%s' for condition type %s", condition.Operator, condition.Type)
		}
	}

	return nil
}

// validateAction validates a single action
func (m *Manager) validateAction(action Action) error {
	validTypes := map[ActionType]bool{
		ActionMove:   true,
		ActionCopy:   true,
		ActionDelete: true,
		ActionRename: true,
		ActionLink:   true,
	}

	if !validTypes[action.Type] {
		return fmt.Errorf("invalid action type: %s", action.Type)
	}

	// Actions that require destination
	if action.Type == ActionMove || action.Type == ActionCopy || action.Type == ActionLink {
		if action.Destination == "" && action.Template == "" {
			return fmt.Errorf("action type '%s' requires destination or template", action.Type)
		}
	}

	return nil
}

// GetConfigPath returns the path to the rules configuration file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// CreateExampleRules creates example rules for demonstration
func (m *Manager) CreateExampleRules() error {
	examples := []Rule{
		{
			Name:        "archive-old-screenshots",
			Description: "Move screenshots older than 30 days to archive folder",
			Enabled:     false, // Disabled by default for safety
			Conditions: []Condition{
				{
					Type:     ConditionNamePattern,
					Operator: OpContains,
					Value:    "Screenshot",
				},
				{
					Type:     ConditionAge,
					Operator: OpGreaterThan,
					Value:    "30d",
				},
			},
			Actions: []Action{
				{
					Type:        ActionMove,
					Destination: "~/Pictures/Archive/Screenshots/",
					Backup:      true,
					Confirm:     true,
				},
			},
		},
		{
			Name:        "cleanup-large-temp-files",
			Description: "Delete temporary files larger than 100MB and older than 7 days",
			Enabled:     false,
			Conditions: []Condition{
				{
					Type:     ConditionPath,
					Operator: OpContains,
					Value:    "/tmp/",
				},
				{
					Type:     ConditionSize,
					Operator: OpGreaterThan,
					Value:    "100M",
				},
				{
					Type:     ConditionAge,
					Operator: OpGreaterThan,
					Value:    "7d",
				},
			},
			Actions: []Action{
				{
					Type:    ActionDelete,
					Confirm: true,
				},
			},
		},
	}

	for _, rule := range examples {
		// Only add if it doesn't exist
		if _, err := m.GetRule(rule.Name); err != nil {
			if err := m.AddRule(rule); err != nil {
				return fmt.Errorf("failed to add example rule '%s': %w", rule.Name, err)
			}
		}
	}

	return nil
}
