package rules

import (
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestExecutorExecution(t *testing.T) {
	executor := NewExecutor(true, true, false) // dry run enabled
	
	t.Run("execute rule with move action", func(t *testing.T) {
		rule := Rule{
			Name:    "test-rule",
			Enabled: true,
			Actions: []Action{
				{
					Type:        ActionMove,
					Destination: "/tmp/archive/",
				},
			},
		}
		
		file := db.File{
			Path:       "/test/file.txt",
			Name:       "file.txt",
			Size:       1024,
			ModifiedAt: time.Now(),
		}
		
		result := executor.ExecuteRule(rule, file, "/test")
		
		// In dry run mode, should succeed but be marked as dry run
		assert.NotNil(t, result.Rule)
		assert.Equal(t, file.Path, result.File.Path)
		assert.Len(t, result.Actions, 1)
	})
	
	t.Run("execute rule with delete action", func(t *testing.T) {
		rule := Rule{
			Name:    "delete-rule",
			Enabled: true,
			Actions: []Action{
				{
					Type: ActionDelete,
				},
			},
		}
		
		file := db.File{
			Path:       "/test/temp.tmp",
			Name:       "temp.tmp",
			Size:       512,
			ModifiedAt: time.Now(),
		}
		
		result := executor.ExecuteRule(rule, file, "/test")
		
		assert.NotNil(t, result.Rule)
		assert.Len(t, result.Actions, 1)
	})
}