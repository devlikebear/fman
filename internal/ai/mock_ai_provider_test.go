package ai

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockAIProvider is a mock implementation of the AIProvider interface for testing.
type MockAIProvider struct {
	mock.Mock
}

// SuggestOrganization mocks the SuggestOrganization method of AIProvider.
func (m *MockAIProvider) SuggestOrganization(ctx context.Context, filePaths []string) (string, error) {
	args := m.Called(ctx, filePaths)
	return args.String(0), args.Error(1)
}
