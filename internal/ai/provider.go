package ai

import (
	"context"
	"github.com/google/generative-ai-go/genai"
)

type AIProvider interface {
	SuggestOrganization(ctx context.Context, filePaths []string) (string, error)
	String() string
}

// GenerativeModelInterface defines the interface for genai.GenerativeModel's methods used by GeminiProvider.
type GenerativeModelInterface interface {
	GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)
}
