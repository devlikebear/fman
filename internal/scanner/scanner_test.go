/*
Copyright © 2025 changheonshin
*/
package scanner

import (
	"context"
	"fmt"
	"testing"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBInterface is a mock implementation of db.DBInterface
type MockDBInterface struct {
	mock.Mock
}

func (m *MockDBInterface) InitDB() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDBInterface) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDBInterface) UpsertFile(file *db.File) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockDBInterface) FindFilesByName(namePattern string) ([]db.File, error) {
	args := m.Called(namePattern)
	return args.Get(0).([]db.File), args.Error(1)
}

func TestNewFileScanner(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockDB := &MockDBInterface{}

	scanner := NewFileScanner(fs, mockDB)

	assert.NotNil(t, scanner)
	assert.Equal(t, fs, scanner.fs)
	assert.Equal(t, mockDB, scanner.database)
}

func TestFileScanner_ScanDirectory(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create test files
	fs.MkdirAll("/test/subdir", 0755)
	afero.WriteFile(fs, "/test/file1.txt", []byte("content1"), 0644)
	afero.WriteFile(fs, "/test/subdir/file2.txt", []byte("content2"), 0644)

	// Create mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil)

	// Create scanner
	scanner := NewFileScanner(fs, mockDB)

	// Create options
	options := &ScanOptions{
		Verbose:   false,
		ForceSudo: false,
	}

	// Perform scan
	ctx := context.Background()
	stats, err := scanner.ScanDirectory(ctx, "/test", options)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.FilesIndexed) // Should index 2 files
	assert.Equal(t, 0, stats.PermissionErrors)

	// Verify mock calls
	mockDB.AssertExpectations(t)
}

func TestFileScanner_ScanDirectoryWithContextCancellation(t *testing.T) {
	// Create in-memory filesystem with many files to ensure context cancellation
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/test", 0755)

	// Create many files to increase chance of context cancellation
	for i := 0; i < 100; i++ {
		afero.WriteFile(fs, fmt.Sprintf("/test/file%d.txt", i), []byte("content"), 0644)
	}

	// Create mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)

	// Create scanner
	scanner := NewFileScanner(fs, mockDB)

	// Create options
	options := &ScanOptions{
		Verbose:   false,
		ForceSudo: false,
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Perform scan
	_, err := scanner.ScanDirectory(ctx, "/test", options)

	// Should return context cancelled error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Verify mock calls
	mockDB.AssertExpectations(t)
}

func TestFileScanner_calculateFileHash(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()
	content := []byte("test content for hashing")
	afero.WriteFile(fs, "/test.txt", content, 0644)

	// Create scanner
	mockDB := &MockDBInterface{}
	scanner := NewFileScanner(fs, mockDB)

	// Calculate hash
	hash, err := scanner.calculateFileHash("/test.txt")

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA-256 produces 64-character hex string

	// Test with non-existent file
	_, err = scanner.calculateFileHash("/nonexistent.txt")
	assert.Error(t, err)
}

func TestScanOptions(t *testing.T) {
	options := &ScanOptions{
		Verbose:   true,
		ForceSudo: false,
	}

	assert.True(t, options.Verbose)
	assert.False(t, options.ForceSudo)
}

func TestScanStats(t *testing.T) {
	stats := &ScanStats{
		FilesIndexed:       10,
		DirectoriesSkipped: 2,
		PermissionErrors:   1,
		SkippedPaths:       []string{"/test/skipped"},
	}

	assert.Equal(t, 10, stats.FilesIndexed)
	assert.Equal(t, 2, stats.DirectoriesSkipped)
	assert.Equal(t, 1, stats.PermissionErrors)
	assert.Len(t, stats.SkippedPaths, 1)
}
