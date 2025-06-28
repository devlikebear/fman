/*
Copyright © 2025 changheonshin
*/
package ai

import (
	"context"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGenerativeModel is a mock implementation of genai.GenerativeModel
type MockGenerativeModel struct {
	mock.Mock
}

func (m *MockGenerativeModel) GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
	args := m.Called(ctx, parts)
	return args.Get(0).(*genai.GenerateContentResponse), args.Error(1)
}

func TestGeminiProvider_String(t *testing.T) {
	provider := &GeminiProvider{}

	result := provider.String()
	expected := "gemini"
	assert.Equal(t, expected, result)
}

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

	_, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini api_key is not set")
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

	_, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gemini model is not set")
}

func TestGeminiSuggestOrganization_EmptyResponse(t *testing.T) {
	// Setup viper config
	viper.Set("gemini.api_key", "test-api-key")
	viper.Set("gemini.model", "test-model")
	defer func() {
		// Clean up after test
		viper.Set("gemini.api_key", "")
		viper.Set("gemini.model", "")
	}()

	// Create mock GenerativeModel that returns empty response
	mockModel := new(MockGenerativeModel)
	mockModel.On("GenerateContent", mock.Anything, mock.Anything).Return(
		&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{}, // Empty candidates
		},
		nil,
	)

	provider := NewGeminiProvider().(*GeminiProvider)
	provider.SetGenerativeModel(mockModel)

	filePaths := []string{"file.txt"}

	_, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "received an empty response from Gemini")

	mockModel.AssertExpectations(t)
}

func TestGeminiSuggestOrganization_GenerateContentError(t *testing.T) {
	// Setup viper config
	viper.Set("gemini.api_key", "test-api-key")
	viper.Set("gemini.model", "test-model")
	defer func() {
		// Clean up after test
		viper.Set("gemini.api_key", "")
		viper.Set("gemini.model", "")
	}()

	// Create mock GenerativeModel that returns an error
	mockModel := new(MockGenerativeModel)
	mockModel.On("GenerateContent", mock.Anything, mock.Anything).Return(
		(*genai.GenerateContentResponse)(nil),
		assert.AnError,
	)

	provider := NewGeminiProvider().(*GeminiProvider)
	provider.SetGenerativeModel(mockModel)

	filePaths := []string{"file.txt"}

	_, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate content")

	mockModel.AssertExpectations(t)
}

func TestNewGeminiProvider(t *testing.T) {
	provider := NewGeminiProvider()
	assert.NotNil(t, provider)
	assert.IsType(t, &GeminiProvider{}, provider)
}

func TestGeminiProvider_SetGenerativeModel(t *testing.T) {
	provider := NewGeminiProvider().(*GeminiProvider)
	mockModel := new(MockGenerativeModel)

	provider.SetGenerativeModel(mockModel)

	assert.Equal(t, mockModel, provider.generativeModel)
}
