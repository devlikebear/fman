package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFindCommand(t *testing.T) {
	// Test that find command exists and has correct structure
	assert.NotNil(t, findCmd)
	assert.Equal(t, "find [name-pattern]", findCmd.Use)
	assert.Contains(t, findCmd.Short, "advanced search")
	assert.Contains(t, findCmd.Long, "criteria")
	assert.NotNil(t, findCmd.RunE)
}

func TestRunFind(t *testing.T) {
	t.Run("successful search with name pattern", func(t *testing.T) {
		mockDB := new(MockDBInterface)

		// Mock files
		files := []db.File{
			{ID: 1, Path: "/test/file1.txt", Name: "file1.txt", Size: 1024, ModifiedAt: time.Now()},
			{ID: 2, Path: "/test/file2.txt", Name: "file2.txt", Size: 2048, ModifiedAt: time.Now()},
		}

		mockDB.On("InitDB").Return(nil)
		mockDB.On("Close").Return(nil)
		mockDB.On("FindFilesByAdvancedCriteria", mock.MatchedBy(func(criteria db.SearchCriteria) bool {
			return criteria.NamePattern == "test"
		})).Return(files, nil)

		err := runFind(findCmd, []string{"test"}, mockDB)

		assert.NoError(t, err)
		mockDB.AssertExpectations(t)
	})

	t.Run("no files found", func(t *testing.T) {
		mockDB := new(MockDBInterface)

		mockDB.On("InitDB").Return(nil)
		mockDB.On("Close").Return(nil)
		mockDB.On("FindFilesByAdvancedCriteria", mock.AnythingOfType("db.SearchCriteria")).Return([]db.File{}, nil)

		err := runFind(findCmd, []string{"nonexistent"}, mockDB)

		assert.NoError(t, err)
		mockDB.AssertExpectations(t)
	})

	t.Run("database init error", func(t *testing.T) {
		mockDB := new(MockDBInterface)

		mockDB.On("InitDB").Return(errors.New("init error"))

		err := runFind(findCmd, []string{"test"}, mockDB)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize database")
		mockDB.AssertExpectations(t)
	})

	t.Run("database search error", func(t *testing.T) {
		mockDB := new(MockDBInterface)

		mockDB.On("InitDB").Return(nil)
		mockDB.On("Close").Return(nil)
		mockDB.On("FindFilesByAdvancedCriteria", mock.AnythingOfType("db.SearchCriteria")).Return([]db.File{}, errors.New("search error"))

		err := runFind(findCmd, []string{"test"}, mockDB)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find files")
		mockDB.AssertExpectations(t)
	})
}

func TestRunFind_Integration(t *testing.T) {
	t.Run("parseSize function tests", func(t *testing.T) {
		// Test parseSize function
		minSize, maxSize, err := parseSize("1M-10M")
		assert.NoError(t, err)
		assert.NotNil(t, minSize)
		assert.NotNil(t, maxSize)
		assert.Equal(t, int64(1024*1024), *minSize)
		assert.Equal(t, int64(10*1024*1024), *maxSize)
	})

	t.Run("parseSize with single value", func(t *testing.T) {
		minSize, maxSize, err := parseSize("+1M")
		assert.NoError(t, err)
		assert.NotNil(t, minSize)
		assert.Nil(t, maxSize)
		assert.Equal(t, int64(1024*1024), *minSize)
	})

	t.Run("parseModified with relative date", func(t *testing.T) {
		after, before, err := parseModified("-30d")
		assert.NoError(t, err)
		assert.NotNil(t, after)
		assert.Nil(t, before)
	})

	t.Run("parseFileType with extension", func(t *testing.T) {
		types, err := parseFileType(".txt")
		assert.NoError(t, err)
		assert.Len(t, types, 1)
		assert.Contains(t, types, ".txt")
	})

	t.Run("parseFileType with category", func(t *testing.T) {
		types, err := parseFileType("image")
		assert.NoError(t, err)
		assert.True(t, len(types) > 0)
		assert.Contains(t, types, ".jpg")
		assert.Contains(t, types, ".png")
	})
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMin *int64
		wantMax *int64
		wantErr bool
	}{
		{
			name:    "size with + prefix",
			input:   "+100M",
			wantMin: int64Ptr(100 * 1024 * 1024),
			wantMax: nil,
			wantErr: false,
		},
		{
			name:    "size with - prefix",
			input:   "-1G",
			wantMin: int64Ptr(0),
			wantMax: int64Ptr(1024 * 1024 * 1024),
			wantErr: false,
		},
		{
			name:    "size range",
			input:   "1M-10M",
			wantMin: int64Ptr(1024 * 1024),
			wantMax: int64Ptr(10 * 1024 * 1024),
			wantErr: false,
		},
		{
			name:    "exact size",
			input:   "500K",
			wantMin: int64Ptr(500 * 1024),
			wantMax: int64Ptr(500 * 1024),
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantMin: nil,
			wantMax: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin, gotMax, err := parseSize(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantMin, gotMin)
			assert.Equal(t, tt.wantMax, gotMax)
		})
	}
}

func TestParseSingleSize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		wantErr  bool
	}{
		{"bytes", "100", 100, false},
		{"kilobytes", "100K", 100 * 1024, false},
		{"megabytes", "100M", 100 * 1024 * 1024, false},
		{"gigabytes", "1G", 1024 * 1024 * 1024, false},
		{"decimal", "1.5M", int64(1.5 * 1024 * 1024), false},
		{"invalid format", "100X", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSingleSize(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseModified(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"relative newer", "-30d", false},
		{"relative older", "+7d", false},
		{"absolute date", "2024-01-01", false},
		{"invalid relative", "-30x", true},
		{"invalid date", "invalid-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			after, before, err := parseModified(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Verify that at least one of after/before is set
			assert.True(t, after != nil || before != nil)

			// For relative dates, verify they're reasonable
			if tt.input == "-30d" {
				assert.NotNil(t, after)
				assert.True(t, after.Before(now))
			}
			if tt.input == "+7d" {
				assert.NotNil(t, before)
				assert.True(t, before.Before(now))
			}
		})
	}
}

func TestParseRelativeDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"days", "30d", 30 * 24 * time.Hour, false},
		{"weeks", "2w", 2 * 7 * 24 * time.Hour, false},
		{"months", "1m", 30 * 24 * time.Hour, false},
		{"years", "1y", 365 * 24 * time.Hour, false},
		{"invalid unit", "30x", 0, true},
		{"invalid format", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRelativeDuration(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFileType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{"image type", "image", []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg"}, false},
		{"video type", "video", []string{".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v"}, false},
		{"specific extension with dot", ".pdf", []string{".pdf"}, false},
		{"specific extension without dot", "pdf", []string{".pdf"}, false},
		{"document type", "document", []string{".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".pages"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFileType(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 2097152, "2.0 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"zero", 0, "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}
