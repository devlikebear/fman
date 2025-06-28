package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestDuplicateCommand(t *testing.T) {
	// Test that duplicate command exists and has correct structure
	assert.NotNil(t, duplicateCmd)
	assert.Equal(t, "duplicate [directory]", duplicateCmd.Use)
	assert.Contains(t, duplicateCmd.Short, "duplicate")
	assert.Contains(t, duplicateCmd.Long, "hash")
	assert.NotNil(t, duplicateCmd.RunE)
}

func TestRunDuplicateWithMockDB(t *testing.T) {
	t.Run("no duplicates found", func(t *testing.T) {
		mockDB := new(MockDBInterface)

		mockDB.On("FindFilesWithHashes", "", int64(1024)).Return([]db.File{}, nil)

		// Test the core logic with mock database
		duplicates, err := findDuplicateFiles(mockDB, "", 1024)

		assert.NoError(t, err)
		assert.Empty(t, duplicates)
		mockDB.AssertExpectations(t)
	})

	t.Run("duplicates found and grouped", func(t *testing.T) {
		mockDB := new(MockDBInterface)

		// Mock files with same hash (duplicates)
		files := []db.File{
			{ID: 1, Path: "/path1/file.txt", Name: "file.txt", Size: 1024, FileHash: "hash1", ModifiedAt: time.Now()},
			{ID: 2, Path: "/path2/file.txt", Name: "file.txt", Size: 1024, FileHash: "hash1", ModifiedAt: time.Now()},
			{ID: 3, Path: "/path3/other.txt", Name: "other.txt", Size: 2048, FileHash: "hash2", ModifiedAt: time.Now()},
		}

		mockDB.On("FindFilesWithHashes", "", int64(1024)).Return(files, nil)

		duplicates, err := findDuplicateFiles(mockDB, "", 1024)

		assert.NoError(t, err)
		assert.Len(t, duplicates, 1)          // Only one group of duplicates (hash1)
		assert.Len(t, duplicates[0].Files, 2) // Two files with same hash
		mockDB.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockDB := new(MockDBInterface)

		mockDB.On("FindFilesWithHashes", "", int64(1024)).Return([]db.File{}, errors.New("database error"))

		duplicates, err := findDuplicateFiles(mockDB, "", 1024)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, duplicates)
		mockDB.AssertExpectations(t)
	})
}

func TestDisplayDuplicates(t *testing.T) {
	t.Run("display duplicate groups", func(t *testing.T) {
		// Create test duplicate groups
		duplicates := []DuplicateGroup{
			{
				Hash: "hash1234567890abcdef",
				Size: 1024,
				Files: []db.File{
					{Path: "/dir1/file.txt", ModifiedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
					{Path: "/dir2/file.txt", ModifiedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)},
				},
			},
		}

		totalDuplicates, totalSize := displayDuplicates(duplicates)

		assert.Equal(t, 2, totalDuplicates)
		assert.Equal(t, int64(1024), totalSize) // One file worth of wasted space
	})
}

func TestDuplicateCommandFlags(t *testing.T) {
	// Test that duplicate command has the expected flags
	assert.NotNil(t, duplicateCmd.Flags().Lookup("remove"))
	assert.NotNil(t, duplicateCmd.Flags().Lookup("interactive"))
	assert.NotNil(t, duplicateCmd.Flags().Lookup("min-size"))
}

func TestDuplicateGroup(t *testing.T) {
	t.Run("duplicate group structure", func(t *testing.T) {
		group := DuplicateGroup{
			Hash: "test-hash",
			Files: []db.File{
				{Path: "/file1.txt", Size: 1024},
				{Path: "/file2.txt", Size: 1024},
			},
			Size: 1024,
		}

		assert.Equal(t, "test-hash", group.Hash)
		assert.Len(t, group.Files, 2)
		assert.Equal(t, int64(1024), group.Size)
	})
}

func TestDuplicateCommandHelp(t *testing.T) {
	t.Run("help text contains key information", func(t *testing.T) {
		helpText := duplicateCmd.Long

		assert.Contains(t, strings.ToLower(helpText), "duplicate")
		assert.Contains(t, strings.ToLower(helpText), "hash")
		assert.Contains(t, strings.ToLower(helpText), "directory")
	})
}

func TestRunDuplicate(t *testing.T) {
	// Create mock with expected behavior
	mockDB := &MockDBInterface{}

	// Test data
	duplicateFiles := []db.File{
		{ID: 1, Path: "/test/file1.txt", Name: "file1.txt", Size: 1024, FileHash: "hash123"},
		{ID: 2, Path: "/test/copy/file1.txt", Name: "file1.txt", Size: 1024, FileHash: "hash123"},
	}

	t.Run("successful duplicate finding", func(t *testing.T) {
		// Setup expectations
		mockDB.On("FindFilesWithHashes", "/test", int64(0)).Return(duplicateFiles, nil).Once()

		// Test the findDuplicateFiles function directly
		groups, err := findDuplicateFiles(mockDB, "/test", 0)

		assert.NoError(t, err)
		assert.Len(t, groups, 1)
		assert.Len(t, groups[0].Files, 2)
		assert.Equal(t, "hash123", groups[0].Hash)

		mockDB.AssertExpectations(t)
	})

	t.Run("no duplicates found", func(t *testing.T) {
		mockDB := &MockDBInterface{}
		mockDB.On("FindFilesWithHashes", "", int64(0)).Return([]db.File{}, nil).Once()

		groups, err := findDuplicateFiles(mockDB, "", 0)

		assert.NoError(t, err)
		assert.Len(t, groups, 0)

		mockDB.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockDB := &MockDBInterface{}
		mockDB.On("FindFilesWithHashes", "", int64(0)).Return([]db.File(nil), errors.New("database error")).Once()

		groups, err := findDuplicateFiles(mockDB, "", 0)

		assert.Error(t, err)
		assert.Nil(t, groups)
		assert.Contains(t, err.Error(), "database error")

		mockDB.AssertExpectations(t)
	})
}
