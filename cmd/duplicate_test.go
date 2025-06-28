package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBInterface for testing duplicate command
type MockDBInterfaceForDuplicate struct {
	mock.Mock
}

func (m *MockDBInterfaceForDuplicate) InitDB() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDBInterfaceForDuplicate) UpsertFile(file *db.File) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockDBInterfaceForDuplicate) FindFilesByName(namePattern string) ([]db.File, error) {
	args := m.Called(namePattern)
	return args.Get(0).([]db.File), args.Error(1)
}

func (m *MockDBInterfaceForDuplicate) FindFilesWithHashes(searchDir string, minSize int64) ([]db.File, error) {
	args := m.Called(searchDir, minSize)
	return args.Get(0).([]db.File), args.Error(1)
}

func (m *MockDBInterfaceForDuplicate) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestDuplicateCommand(t *testing.T) {
	// Test that duplicate command exists and has correct structure
	assert.NotNil(t, duplicateCmd)
	assert.Equal(t, "duplicate [directory]", duplicateCmd.Use)
	assert.Contains(t, duplicateCmd.Short, "duplicate")
	assert.Contains(t, duplicateCmd.Long, "hash")
	assert.NotNil(t, duplicateCmd.RunE)
}

func TestFindDuplicateFiles(t *testing.T) {
	t.Run("no duplicates found", func(t *testing.T) {
		mockDB := new(MockDBInterfaceForDuplicate)

		// Mock files with unique hashes
		files := []db.File{
			{ID: 1, Path: "/file1.txt", Name: "file1.txt", Size: 1024, FileHash: "hash1"},
			{ID: 2, Path: "/file2.txt", Name: "file2.txt", Size: 2048, FileHash: "hash2"},
		}

		mockDB.On("FindFilesWithHashes", "", int64(1024)).Return(files, nil)

		duplicates, err := findDuplicateFiles(mockDB, "", 1024)

		assert.NoError(t, err)
		assert.Empty(t, duplicates)
		mockDB.AssertExpectations(t)
	})

	t.Run("duplicates found", func(t *testing.T) {
		mockDB := new(MockDBInterfaceForDuplicate)

		// Mock files with duplicate hashes
		files := []db.File{
			{ID: 1, Path: "/dir1/file.txt", Name: "file.txt", Size: 1024, FileHash: "hash1", ModifiedAt: time.Now()},
			{ID: 2, Path: "/dir2/file.txt", Name: "file.txt", Size: 1024, FileHash: "hash1", ModifiedAt: time.Now()},
			{ID: 3, Path: "/dir3/unique.txt", Name: "unique.txt", Size: 2048, FileHash: "hash2", ModifiedAt: time.Now()},
		}

		mockDB.On("FindFilesWithHashes", "", int64(1024)).Return(files, nil)

		duplicates, err := findDuplicateFiles(mockDB, "", 1024)

		assert.NoError(t, err)
		assert.Len(t, duplicates, 1)
		assert.Equal(t, "hash1", duplicates[0].Hash)
		assert.Len(t, duplicates[0].Files, 2)
		assert.Equal(t, int64(1024), duplicates[0].Size)
		mockDB.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockDB := new(MockDBInterfaceForDuplicate)

		mockDB.On("FindFilesWithHashes", "", int64(1024)).Return([]db.File{}, errors.New("db error"))

		duplicates, err := findDuplicateFiles(mockDB, "", 1024)

		assert.Error(t, err)
		assert.Nil(t, duplicates)
		assert.Contains(t, err.Error(), "db error")
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

func TestRunDuplicateWithMockDB(t *testing.T) {
	t.Run("no duplicates found", func(t *testing.T) {
		mockDB := new(MockDBInterfaceForDuplicate)

		mockDB.On("FindFilesWithHashes", "", int64(1024)).Return([]db.File{}, nil)

		// Test the core logic with mock database
		duplicates, err := findDuplicateFiles(mockDB, "", 1024)

		assert.NoError(t, err)
		assert.Empty(t, duplicates)
		mockDB.AssertExpectations(t)
	})

	t.Run("database init error", func(t *testing.T) {
		mockDB := new(MockDBInterfaceForDuplicate)

		mockDB.On("InitDB").Return(errors.New("init error"))

		err := mockDB.InitDB()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "init error")
		mockDB.AssertExpectations(t)
	})
}

func TestDuplicateCommandFlags(t *testing.T) {
	t.Run("flags are properly defined", func(t *testing.T) {
		removeFlag := duplicateCmd.Flags().Lookup("remove")
		assert.NotNil(t, removeFlag)
		assert.Equal(t, "false", removeFlag.DefValue)

		interactiveFlag := duplicateCmd.Flags().Lookup("interactive")
		assert.NotNil(t, interactiveFlag)
		assert.Equal(t, "false", interactiveFlag.DefValue)

		minSizeFlag := duplicateCmd.Flags().Lookup("min-size")
		assert.NotNil(t, minSizeFlag)
		assert.Equal(t, "1024", minSizeFlag.DefValue)
	})
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
