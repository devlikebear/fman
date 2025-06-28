package ai

import (
	"context"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGeminiSuggestOrganization(t *testing.T) {
	// Setup viper config using the global viper instance
	viper.Set("gemini.api_key", "test-api-key")
	viper.Set("gemini.model", "test-model")
	defer func() {
		// Clean up after test
		viper.Set("gemini.api_key", "")
		viper.Set("gemini.model", "")
	}()

	// Create mock GenerativeModel
	mockModel := new(MockGenerativeModel)
	mockModel.On("GenerateContent", mock.Anything, mock.Anything).Return(
		&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []genai.Part{genai.Text("mv file.txt documents/file.txt")},
					},
				},
			},
		},
		nil,
	)

	provider := NewGeminiProvider().(*GeminiProvider) // Cast to concrete type to access SetGenerativeModel
	provider.SetGenerativeModel(mockModel)            // Inject mock model

	filePaths := []string{"file.txt"}

	suggestions, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.NoError(t, err)
	assert.Equal(t, "mv file.txt documents/file.txt", suggestions)

	mockModel.AssertExpectations(t)
}

func TestGeminiSuggestOrganization_APIKeyMissing(t *testing.T) {
	// Setup viper config with missing API key
	viper.Set("gemini.api_key", "")
	viper.Set("gemini.model", "test-model")
	defer func() {
		// Clean up after test
		viper.Set("gemini.api_key", "")
		viper.Set("gemini.model", "")
	}()

	provider := NewGeminiProvider()
	filePaths := []string{"file.txt"}

	suggestions, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini api_key is not set")
	assert.Empty(t, suggestions)
}

func TestGeminiSuggestOrganization_ModelMissing(t *testing.T) {
	// Setup viper config with missing model
	viper.Set("gemini.api_key", "test-api-key")
	viper.Set("gemini.model", "")
	defer func() {
		// Clean up after test
		viper.Set("gemini.api_key", "")
		viper.Set("gemini.model", "")
	}()

	provider := NewGeminiProvider()
	filePaths := []string{"file.txt"}

	suggestions, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini model is not set")
	assert.Empty(t, suggestions)
}
