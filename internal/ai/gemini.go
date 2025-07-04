package ai

import (
	"context"
	"fmt"

	"github.com/devlikebear/fman/internal/db"
	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

// GeminiProvider implements the AIProvider interface for Gemini.
type GeminiProvider struct {
	generativeModel GenerativeModelInterface // Field to hold the generative model, can be mocked for testing
}

// NewGeminiProvider creates a new GeminiProvider.
func NewGeminiProvider() AIProvider {
	return &GeminiProvider{}
}

var NewGenaiClient = genai.NewClient

// SetGenerativeModel allows injecting a mock GenerativeModel for testing.
func (p *GeminiProvider) SetGenerativeModel(model GenerativeModelInterface) {
	p.generativeModel = model
}

// SuggestOrganization sends a request to the Gemini API to get organization suggestions.
func (p *GeminiProvider) SuggestOrganization(ctx context.Context, filePaths []string) (string, error) {
	apiKey := viper.GetString("gemini.api_key")
	if apiKey == "" {
		return "", fmt.Errorf("gemini api_key is not set in the configuration")
	}
	modelName := viper.GetString("gemini.model")
	if modelName == "" {
		return "", fmt.Errorf("gemini model is not set in the configuration")
	}

	var model GenerativeModelInterface
	if p.generativeModel != nil { // Use injected model for testing
		model = p.generativeModel
	} else {
		client, err := NewGenaiClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			return "", fmt.Errorf("failed to create gemini client: %w", err)
		}
		defer client.Close()
		model = client.GenerativeModel(modelName)
	}

	prompt := buildPrompt(filePaths)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("received an empty response from Gemini")
	}

	// Assuming the first part of the first candidate is the text we want.
	part := resp.Candidates[0].Content.Parts[0]
	if txt, ok := part.(genai.Text); ok {
		return string(txt), nil
	}

	return "", fmt.Errorf("unexpected response format from Gemini")
}

// ParseSearchQuery converts natural language query into structured search criteria using Gemini.
func (p *GeminiProvider) ParseSearchQuery(ctx context.Context, query string) (*db.SearchCriteria, error) {
	apiKey := viper.GetString("gemini.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api_key is not set in the configuration")
	}
	modelName := viper.GetString("gemini.model")
	if modelName == "" {
		return nil, fmt.Errorf("gemini model is not set in the configuration")
	}

	var model GenerativeModelInterface
	if p.generativeModel != nil { // Use injected model for testing
		model = p.generativeModel
	} else {
		client, err := NewGenaiClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini client: %w", err)
		}
		defer client.Close()
		model = client.GenerativeModel(modelName)
	}

	prompt := buildSearchPrompt(query)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("received an empty response from Gemini")
	}

	// Assuming the first part of the first candidate is the text we want.
	part := resp.Candidates[0].Content.Parts[0]
	if txt, ok := part.(genai.Text); ok {
		return parseSearchCriteriaFromJSON(string(txt))
	}

	return nil, fmt.Errorf("unexpected response format from Gemini")
}

func (p *GeminiProvider) String() string {
	return "gemini"
}
