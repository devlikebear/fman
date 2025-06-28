package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/devlikebear/fman/internal/ai"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAIProvider is a mock implementation of ai.AIProvider for testing
type MockAIProvider struct {
	mock.Mock
}

func (m *MockAIProvider) SuggestOrganization(ctx context.Context, filePaths []string) (string, error) {
	args := m.Called(ctx, filePaths)
	return args.String(0), args.Error(1)
}

func (m *MockAIProvider) String() string {
	args := m.Called()
	return args.String(0)
}

func TestOrganizeCommand_NoAIFlag(t *testing.T) {
	// Reset viper for each test
	viper.Reset()

	// Execute the command without --ai flag
	err := organizeCmd.RunE(organizeCmd, []string{"/testdir"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the --ai flag is required")
}

func TestOrganizeCommand_UnknownAIProvider(t *testing.T) {
	// Reset viper for each test
	viper.Reset()
	viper.Set("ai_provider", "unknown")

	// Set --ai flag
	organizeCmd.Flags().Set("ai", "true")
	defer organizeCmd.Flags().Set("ai", "false")

	// Execute the command
	err := organizeCmd.RunE(organizeCmd, []string{"/testdir"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AI provider")
}

func TestOrganizeCommand_GeminiProvider(t *testing.T) {
	// Reset viper for each test
	viper.Reset()
	viper.Set("ai_provider", "gemini")
	viper.Set("gemini.api_key", "test-key")
	viper.Set("gemini.model", "test-model")

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fman-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test file in the temporary directory
	testFile := fmt.Sprintf("%s/file1.txt", tempDir)
	err = os.WriteFile(testFile, []byte("content"), 0644)
	assert.NoError(t, err)

	// Create a mock AI provider
	mockAIProvider := new(MockAIProvider)
	mockAIProvider.On("String").Return("gemini")
	mockAIProvider.On("SuggestOrganization", mock.Anything, mock.Anything).Return("echo 'test command'", nil).Once()

	// Override the NewGeminiProvider to return our mock
	oldNewGeminiProvider := NewGeminiProvider
	NewGeminiProvider = func() ai.AIProvider { return mockAIProvider }
	defer func() { NewGeminiProvider = oldNewGeminiProvider }()

	// Use the real filesystem for this test since we need to execute actual commands
	fs := afero.NewOsFs()

	// Redirect stdin to provide 'y' input
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
		w.Close()
	}()

	// Provide 'y' as input to confirm execution
	go func() {
		defer w.Close()
		io.WriteString(w, "y\n")
	}()

	// Execute the command
	err = runOrganize(organizeCmd, []string{tempDir}, fs, mockAIProvider)
	assert.NoError(t, err)

	// Assert that mock expectations were met
	mockAIProvider.AssertExpectations(t)
}

func TestOrganizeCommand_OllamaProvider(t *testing.T) {
	// Reset viper for each test
	viper.Reset()
	viper.Set("ai_provider", "ollama")
	viper.Set("ollama.base_url", "http://localhost:11434")
	viper.Set("ollama.model", "test-model")

	// Create a mock AI provider
	mockAIProvider := new(MockAIProvider)
	mockAIProvider.On("String").Return("ollama")
	mockAIProvider.On("SuggestOrganization", mock.Anything, mock.Anything).Return("mv file2.txt images/file2.txt", nil).Once()

	// Override the NewOllamaProvider to return our mock
	oldNewOllamaProvider := NewOllamaProvider
	NewOllamaProvider = func() ai.AIProvider { return mockAIProvider }
	defer func() { NewOllamaProvider = oldNewOllamaProvider }()

	// Initialize afero filesystem
	fs := afero.NewMemMapFs()

	// Create a dummy file in the mock filesystem
	_ = afero.WriteFile(fs, "/testdir/file2.txt", []byte("content"), 0644)

	// Redirect stdin to provide 'n' input
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
		w.Close()
	}()

	// Provide 'n' as input to cancel execution
	go func() {
		defer w.Close()
		io.WriteString(w, "n\n")
	}()

	// Execute the command
	err := runOrganize(organizeCmd, []string{"/testdir"}, fs, mockAIProvider)
	assert.NoError(t, err)

	// Assert that mock expectations were met
	mockAIProvider.AssertExpectations(t)
}

func TestOrganizeCommand_NoFilesInDir(t *testing.T) {
	// Reset viper for each test
	viper.Reset()
	viper.Set("ai_provider", "gemini")
	viper.Set("gemini.api_key", "test-key")
	viper.Set("gemini.model", "test-model")

	// Create a mock AI provider (should not be called)
	mockAIProvider := new(MockAIProvider)
	mockAIProvider.On("String").Return("gemini")
	// No expectations set for SuggestOrganization, as it shouldn't be called

	// Override the NewGeminiProvider to return our mock
	oldNewGeminiProvider := NewGeminiProvider
	NewGeminiProvider = func() ai.AIProvider { return mockAIProvider }
	defer func() { NewGeminiProvider = oldNewGeminiProvider }()

	// Initialize afero filesystem
	fs := afero.NewMemMapFs()

	// Create an empty directory in the mock filesystem
	_ = fs.MkdirAll("/empty_dir", 0755)

	// Execute the command
	err := runOrganize(organizeCmd, []string{"/empty_dir"}, fs, mockAIProvider)
	assert.NoError(t, err)

	// Assert that SuggestOrganization was NOT called
	mockAIProvider.AssertNotCalled(t, "SuggestOrganization", mock.Anything, mock.Anything)
}
