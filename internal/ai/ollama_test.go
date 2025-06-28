package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestOllamaSuggestOrganization(t *testing.T) {
	// Setup mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/generate", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Simulate a successful response
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"response": "mv file1.txt documents/file1.txt"}`)
	}))
	defer server.Close()

	// Configure viper to use the mock server URL and a dummy model
	viper.Set("ollama.base_url", server.URL)
	viper.Set("ollama.model", "test-model")
	defer func() {
		// Clean up after test
		viper.Set("ollama.base_url", "")
		viper.Set("ollama.model", "")
	}()

	provider := NewOllamaProvider()
	filePaths := []string{"file1.txt", "file2.jpg"}

	suggestions, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.NoError(t, err)
	assert.Equal(t, "mv file1.txt documents/file1.txt", suggestions)
}

func TestOllamaSuggestOrganization_APIError(t *testing.T) {
	// Setup mock Ollama server to return an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": "internal server error"}`)
	}))
	defer server.Close()

	// Configure viper to use the mock server URL
	viper.Set("ollama.base_url", server.URL)
	viper.Set("ollama.model", "test-model")
	defer func() {
		// Clean up after test
		viper.Set("ollama.base_url", "")
		viper.Set("ollama.model", "")
	}()

	provider := NewOllamaProvider()
	filePaths := []string{"file1.txt"}

	suggestions, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "received non-OK response from ollama")
	assert.Empty(t, suggestions)
}

func TestOllamaSuggestOrganization_ConfigError(t *testing.T) {
	// Test missing base_url
	viper.Set("ollama.base_url", "")
	viper.Set("ollama.model", "test-model")
	defer func() {
		// Clean up after test
		viper.Set("ollama.base_url", "")
		viper.Set("ollama.model", "")
	}()

	provider := NewOllamaProvider()
	filePaths := []string{"file1.txt"}

	suggestions, err := provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ollama base_url is not set")
	assert.Empty(t, suggestions)

	// Test missing model
	viper.Set("ollama.base_url", "http://localhost:11434")
	viper.Set("ollama.model", "")

	provider = NewOllamaProvider()
	suggestions, err = provider.SuggestOrganization(context.Background(), filePaths)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ollama model is not set")
	assert.Empty(t, suggestions)
}
