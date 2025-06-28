package ai

import (
	"context"

	"github.com/google/generative-ai-go/genai"
	"github.com/stretchr/testify/mock"
)

// MockGenerativeModel is a mock implementation of GenerativeModelInterface for testing.
type MockGenerativeModel struct {
	mock.Mock
}

// GenerateContent mocks the GenerateContent method of GenerativeModelInterface.
func (m *MockGenerativeModel) GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
	args := m.Called(ctx, parts)
	return args.Get(0).(*genai.GenerateContentResponse), args.Error(1)
}