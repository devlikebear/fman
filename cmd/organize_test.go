package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrganizeCommand(t *testing.T) {
	t.Run("organize command exists", func(t *testing.T) {
		assert.NotNil(t, organizeCmd)
		assert.Equal(t, "organize <directory>", organizeCmd.Use)
	})

	t.Run("runOrganize function exists", func(t *testing.T) {
		// Just test that the function exists without executing it
		assert.NotNil(t, runOrganize)
	})

	t.Run("AI provider constructors exist", func(t *testing.T) {
		assert.NotNil(t, NewGeminiProvider)
		assert.NotNil(t, NewOllamaProvider)
	})
}
