package rules

import (
	"time"

	"github.com/devlikebear/fman/internal/db"
)

// Rule represents a file organization rule
type Rule struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Enabled     bool        `yaml:"enabled"`
	Conditions  []Condition `yaml:"conditions"`
	Actions     []Action    `yaml:"actions"`
	CreatedAt   time.Time   `yaml:"created_at"`
	UpdatedAt   time.Time   `yaml:"updated_at"`
}

// Condition represents a condition that files must meet
type Condition struct {
	Type     ConditionType `yaml:"type"`
	Operator string        `yaml:"operator,omitempty"` // >, <, >=, <=, ==, !=, contains, matches
	Value    string        `yaml:"value"`
	Field    string        `yaml:"field,omitempty"` // for custom field conditions
}

// Action represents an action to perform on matching files
type Action struct {
	Type        ActionType `yaml:"type"`
	Destination string     `yaml:"destination,omitempty"`
	Template    string     `yaml:"template,omitempty"` // for dynamic destinations
	Backup      bool       `yaml:"backup,omitempty"`
	Confirm     bool       `yaml:"confirm,omitempty"`
}

// ConditionType represents the type of condition
type ConditionType string

const (
	ConditionNamePattern ConditionType = "name_pattern"
	ConditionExtension   ConditionType = "extension"
	ConditionSize        ConditionType = "size"
	ConditionAge         ConditionType = "age"
	ConditionModified    ConditionType = "modified"
	ConditionPath        ConditionType = "path"
	ConditionFileType    ConditionType = "file_type"
	ConditionMimeType    ConditionType = "mime_type"
)

// ActionType represents the type of action
type ActionType string

const (
	ActionMove   ActionType = "move"
	ActionCopy   ActionType = "copy"
	ActionDelete ActionType = "delete"
	ActionRename ActionType = "rename"
	ActionLink   ActionType = "link"
)

// RulesConfig represents the configuration file structure
type RulesConfig struct {
	Version string `yaml:"version"`
	Rules   []Rule `yaml:"rules"`
}

// EvaluationContext provides context for rule evaluation
type EvaluationContext struct {
	File      db.File
	BaseDir   string
	DryRun    bool
	Verbose   bool
	Timestamp time.Time
}

// ExecutionResult represents the result of executing a rule
type ExecutionResult struct {
	Rule       *Rule
	File       db.File
	Actions    []ActionResult
	Success    bool
	Error      error
	Skipped    bool
	SkipReason string
}

// ActionResult represents the result of executing an action
type ActionResult struct {
	Action      Action
	Source      string
	Destination string
	Success     bool
	Error       error
	Skipped     bool
	SkipReason  string
}

// ExecutionSummary provides a summary of rule execution
type ExecutionSummary struct {
	TotalFiles        int
	ProcessedFiles    int
	SuccessfulActions int
	FailedActions     int
	SkippedActions    int
	Results           []ExecutionResult
	Errors            []error
	Duration          time.Duration
}

// Operator constants for conditions
const (
	OpEqual              = "=="
	OpNotEqual           = "!="
	OpGreaterThan        = ">"
	OpLessThan           = "<"
	OpGreaterThanOrEqual = ">="
	OpLessThanOrEqual    = "<="
	OpContains           = "contains"
	OpMatches            = "matches"
	OpStartsWith         = "starts_with"
	OpEndsWith           = "ends_with"
)

// Default rule file configuration
const (
	DefaultRulesFile = "rules.yml"
	DefaultVersion   = "1.0"
)
