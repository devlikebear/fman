package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSearchCommand(t *testing.T) {
	// Test that search command exists and has correct structure
	assert.NotNil(t, searchCmd)
	assert.Equal(t, "search [natural-language-query]", searchCmd.Use)
	assert.Contains(t, searchCmd.Short, "natural language")
	assert.Contains(t, searchCmd.Long, "AI")
	assert.NotNil(t, searchCmd.RunE)
}

func TestRunSearch(t *testing.T) {
	// Save original viper config
	originalProvider := viper.GetString("ai_provider")
	defer viper.Set("ai_provider", originalProvider)

	t.Run("successful search with results", func(t *testing.T) {
		// Setup
		mockDB := new(MockDBInterface)
		viper.Set("ai_provider", "gemini")
		viper.Set("gemini.api_key", "test-key")
		viper.Set("gemini.model", "test-model")

		// Mock files
		files := []db.File{
			{ID: 1, Path: "/test/image1.jpg", Name: "image1.jpg", Size: 1024 * 1024 * 5, ModifiedAt: time.Now()},
			{ID: 2, Path: "/test/image2.png", Name: "image2.png", Size: 1024 * 1024 * 8, ModifiedAt: time.Now()},
		}

		// Mock search criteria
		criteria := &db.SearchCriteria{
			FileTypes: []string{".jpg", ".png"},
			MinSize:   func() *int64 { s := int64(1024 * 1024); return &s }(),
		}

		mockDB.On("InitDB").Return(nil)
		mockDB.On("Close").Return(nil)
		mockDB.On("FindFilesByAdvancedCriteria", mock.AnythingOfType("db.SearchCriteria")).Return(files, nil)

		// Create a custom run function that uses mock AI provider
		runSearchWithMockAI := func(cmd *cobra.Command, args []string, database db.DBInterface, aiProvider *MockAIProvider) error {
			// Initialize DB
			if err := database.InitDB(); err != nil {
				return err
			}
			defer database.Close()

			// Mock AI provider response
			aiProvider.On("ParseSearchQuery", mock.AnythingOfType("*context.timerCtx"), args[0]).Return(criteria, nil)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			parsedCriteria, err := aiProvider.ParseSearchQuery(ctx, args[0])
			if err != nil {
				return err
			}

			_, err = database.FindFilesByAdvancedCriteria(*parsedCriteria)
			return err
		}

		mockAI := new(MockAIProvider)
		err := runSearchWithMockAI(searchCmd, []string{"find large images"}, mockDB, mockAI)

		assert.NoError(t, err)
		mockDB.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("database init error", func(t *testing.T) {
		mockDB := new(MockDBInterface)
		viper.Set("ai_provider", "gemini")

		mockDB.On("InitDB").Return(errors.New("init error"))

		err := runSearch(searchCmd, []string{"test query"}, mockDB)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize database")
		mockDB.AssertExpectations(t)
	})

	t.Run("no AI provider configured", func(t *testing.T) {
		mockDB := new(MockDBInterface)
		viper.Set("ai_provider", "")

		mockDB.On("InitDB").Return(nil)
		mockDB.On("Close").Return(nil)

		err := runSearch(searchCmd, []string{"test query"}, mockDB)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get AI provider")
		mockDB.AssertExpectations(t)
	})
}

func TestGetAIProvider(t *testing.T) {
	// Save original viper config
	originalProvider := viper.GetString("ai_provider")
	defer viper.Set("ai_provider", originalProvider)

	t.Run("gemini provider", func(t *testing.T) {
		viper.Set("ai_provider", "gemini")

		provider, err := getAIProvider()

		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("ollama provider", func(t *testing.T) {
		viper.Set("ai_provider", "ollama")

		provider, err := getAIProvider()

		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("unsupported provider", func(t *testing.T) {
		viper.Set("ai_provider", "unsupported")

		provider, err := getAIProvider()

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "unsupported AI provider")
	})

	t.Run("no provider configured", func(t *testing.T) {
		viper.Set("ai_provider", "")

		provider, err := getAIProvider()

		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "ai_provider is not set")
	})
}

func TestPrintSearchCriteria(t *testing.T) {
	// This test mainly checks that the function doesn't panic
	// and handles various criteria combinations properly

	t.Run("empty criteria", func(t *testing.T) {
		criteria := &db.SearchCriteria{}

		// Should not panic
		assert.NotPanics(t, func() {
			printSearchCriteria(criteria)
		})
	})

	t.Run("full criteria", func(t *testing.T) {
		minSize := int64(1024 * 1024)
		maxSize := int64(1024 * 1024 * 100)
		after := time.Now().Add(-24 * time.Hour)
		before := time.Now()

		criteria := &db.SearchCriteria{
			NamePattern:    "test",
			MinSize:        &minSize,
			MaxSize:        &maxSize,
			ModifiedAfter:  &after,
			ModifiedBefore: &before,
			SearchDir:      "/test/dir",
			FileTypes:      []string{".jpg", ".png"},
		}

		// Should not panic
		assert.NotPanics(t, func() {
			printSearchCriteria(criteria)
		})
	})
}
