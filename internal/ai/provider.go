package ai

import "context"

type AIProvider interface {
	SuggestOrganization(ctx context.Context, filePaths []string) (string, error)
}
