package ai

import (
	"context"

	"github.com/devlikebear/fman/internal/db"
	"github.com/google/generative-ai-go/genai"
)

// AIProvider defines the interface for AI-powered operations.
type AIProvider interface {
	// SuggestOrganization suggests file organization based on file paths.
	SuggestOrganization(ctx context.Context, filePaths []string) (string, error)

	// ParseSearchQuery converts natural language query into structured search criteria.
	ParseSearchQuery(ctx context.Context, query string) (*db.SearchCriteria, error)
}

// GenerativeModelInterface defines the interface for genai.GenerativeModel's methods used by GeminiProvider.
type GenerativeModelInterface interface {
	GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)
}
