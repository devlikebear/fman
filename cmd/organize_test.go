package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestRunOrganize(t *testing.T) {
	t.Run("runOrganize with no arguments", func(t *testing.T) {
		mockProvider := &MockAIProvider{}
		// This should fail due to index out of range since no args provided
		assert.Panics(t, func() {
			runOrganize(organizeCmd, []string{}, fileSystem, mockProvider)
		})
	})

	t.Run("runOrganize with invalid directory", func(t *testing.T) {
		mockProvider := &MockAIProvider{}
		mockProvider.On("String").Return("mock-provider")
		err := runOrganize(organizeCmd, []string{"/non/existent/directory"}, fileSystem, mockProvider)
		// Should fail due to invalid directory
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read directory")
	})

	t.Run("runOrganize with current directory", func(t *testing.T) {
		mockProvider := &MockAIProvider{}
		mockProvider.On("String").Return("mock-provider")
		mockProvider.On("SuggestOrganization", mock.Anything, mock.Anything).Return("", fmt.Errorf("mock error"))
		
		err := runOrganize(organizeCmd, []string{"."}, fileSystem, mockProvider)
		// Should fail due to mock provider error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get suggestions")
	})
}

// MockAIProvider for testing
type MockAIProvider struct {
	mock.Mock
}

func (m *MockAIProvider) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAIProvider) SuggestOrganization(ctx context.Context, filePaths []string) (string, error) {
	args := m.Called(ctx, filePaths)
	return args.String(0), args.Error(1)
}
