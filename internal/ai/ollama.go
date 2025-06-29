package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/viper"
)

// OllamaProvider implements the AIProvider interface for Ollama.
type OllamaProvider struct{}

// NewOllamaProvider creates a new OllamaProvider.
func NewOllamaProvider() AIProvider {
	return &OllamaProvider{}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

// SuggestOrganization sends a request to the Ollama API to get organization suggestions.
func (p *OllamaProvider) SuggestOrganization(ctx context.Context, filePaths []string) (string, error) {
	baseURL := viper.GetString("ollama.base_url")
	if baseURL == "" {
		return "", fmt.Errorf("ollama base_url is not set in the configuration")
	}
	model := viper.GetString("ollama.model")
	if model == "" {
		return "", fmt.Errorf("ollama model is not set in the configuration")
	}

	prompt := buildPrompt(filePaths)

	return p.makeRequest(ctx, baseURL, model, prompt)
}

// ParseSearchQuery converts natural language query into structured search criteria using Ollama.
func (p *OllamaProvider) ParseSearchQuery(ctx context.Context, query string) (*db.SearchCriteria, error) {
	baseURL := viper.GetString("ollama.base_url")
	if baseURL == "" {
		return nil, fmt.Errorf("ollama base_url is not set in the configuration")
	}
	model := viper.GetString("ollama.model")
	if model == "" {
		return nil, fmt.Errorf("ollama model is not set in the configuration")
	}

	prompt := buildSearchPrompt(query)

	response, err := p.makeRequest(ctx, baseURL, model, prompt)
	if err != nil {
		return nil, err
	}

	return parseSearchCriteriaFromJSON(response)
}

// makeRequest makes an HTTP request to Ollama API
func (p *OllamaProvider) makeRequest(ctx context.Context, baseURL, model, prompt string) (string, error) {
	apiURL := fmt.Sprintf("%s/api/generate", baseURL)

	reqBody := ollamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK response from ollama: %s", resp.Status)
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return ollamaResp.Response, nil
}

func buildPrompt(filePaths []string) string {
	prompt := "You are a file organization expert. Based on the following file list, suggest a series of shell commands (like mv or mkdir) to organize them into a more structured directory. Only output the shell commands, without any explanation.\n\nFile list:\n"
	for _, path := range filePaths {
		prompt += fmt.Sprintf("- %s\n", path)
	}
	return prompt
}

func (p *OllamaProvider) String() string {
	return "ollama"
}
