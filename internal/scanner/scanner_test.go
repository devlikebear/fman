/*
Copyright © 2025 changheonshin
*/
package scanner

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func (m *MockDBInterface) FindFilesWithHashes(searchDir string, minSize int64) ([]db.File, error) {
	args := m.Called(searchDir, minSize)
	return args.Get(0).([]db.File), args.Error(1)
}

func (m *MockDBInterface) FindFilesByAdvancedCriteria(criteria db.SearchCriteria) ([]db.File, error) {
	args := m.Called(criteria)
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

	// Create options with resource limits for testing
	options := &ScanOptions{
		Verbose:       false,
		ForceSudo:     false,
		ThrottleDelay: time.Microsecond * 100, // 매우 짧은 지연으로 테스트 속도 유지
		MaxFileSize:   1024 * 1024,            // 1MB 제한
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

func TestFileScanner_ScanDirectoryWithVerbose(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create test files
	afero.WriteFile(fs, "/test/file1.txt", []byte("content1"), 0644)

	// Create mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil)

	// Create scanner
	scanner := NewFileScanner(fs, mockDB)

	// Create options with verbose
	options := &ScanOptions{
		Verbose:   true,
		ForceSudo: false,
	}

	// Perform scan
	ctx := context.Background()
	stats, err := scanner.ScanDirectory(ctx, "/test", options)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 1, stats.FilesIndexed)

	// Verify mock calls
	mockDB.AssertExpectations(t)
}

func TestFileScanner_ScanDirectoryWithDBInitError(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create mock database that fails to initialize
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(fmt.Errorf("database init failed"))

	// Create scanner
	scanner := NewFileScanner(fs, mockDB)

	// Create options with resource limits for testing
	options := &ScanOptions{
		Verbose:       false,
		ForceSudo:     false,
		ThrottleDelay: time.Microsecond * 100, // 테스트용 짧은 지연
		MaxFileSize:   1024 * 1024,            // 1MB 제한
	}

	// Perform scan
	ctx := context.Background()
	stats, err := scanner.ScanDirectory(ctx, "/test", options)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "failed to initialize database")

	// Verify mock calls
	mockDB.AssertExpectations(t)
}

func TestFileScanner_ScanDirectoryWithUpsertError(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/test/file1.txt", []byte("content1"), 0644)

	// Create mock database that fails to upsert
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(fmt.Errorf("upsert failed"))

	// Create scanner
	scanner := NewFileScanner(fs, mockDB)

	// Create options with resource limits for testing
	options := &ScanOptions{
		Verbose:       false,
		ForceSudo:     false,
		ThrottleDelay: time.Microsecond * 100, // 테스트용 짧은 지연
		MaxFileSize:   1024 * 1024,            // 1MB 제한
	}

	// Perform scan
	ctx := context.Background()
	stats, err := scanner.ScanDirectory(ctx, "/test", options)

	// Should not return error but stats should show 0 indexed files
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.FilesIndexed) // Should be 0 due to upsert failure

	// Verify mock calls
	mockDB.AssertExpectations(t)
}

func TestFileScanner_ScanDirectoryWithSkippedDirectories(t *testing.T) {
	// Create in-memory filesystem with directories that should be skipped
	fs := afero.NewMemMapFs()

	// Create directories that should be skipped based on current OS
	fs.MkdirAll("/test/.Trash", 0755)
	fs.MkdirAll("/test/normal", 0755)
	afero.WriteFile(fs, "/test/normal/file1.txt", []byte("content1"), 0644)
	afero.WriteFile(fs, "/test/.Trash/trashed.txt", []byte("trash"), 0644)

	// Create mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil)

	// Create scanner
	scanner := NewFileScanner(fs, mockDB)

	// Create options
	options := &ScanOptions{
		Verbose:   true,
		ForceSudo: false,
	}

	// Perform scan
	ctx := context.Background()
	stats, err := scanner.ScanDirectory(ctx, "/test", options)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	// Should have skipped some directories
	assert.Greater(t, stats.DirectoriesSkipped, 0)

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

func TestFileScanner_ScanDirectoryNonExistentPath(t *testing.T) {
	// Create in-memory filesystem
	fs := afero.NewMemMapFs()

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

	// Perform scan on non-existent directory
	ctx := context.Background()
	stats, err := scanner.ScanDirectory(ctx, "/nonexistent", options)

	// Should return error
	assert.Error(t, err)
	assert.Nil(t, stats)

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

func TestFileScanner_calculateFileHashEmptyFile(t *testing.T) {
	// Create in-memory filesystem with empty file
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/empty.txt", []byte(""), 0644)

	// Create scanner
	mockDB := &MockDBInterface{}
	scanner := NewFileScanner(fs, mockDB)

	// Calculate hash of empty file
	hash, err := scanner.calculateFileHash("/empty.txt")

	// Assertions
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA-256 produces 64-character hex string
	// Empty file should have a specific SHA-256 hash
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
}

func TestScanOptions(t *testing.T) {
	options := &ScanOptions{
		Verbose:       true,
		ForceSudo:     false,
		ThrottleDelay: time.Millisecond * 10,
		MaxFileSize:   50 * 1024 * 1024,
	}

	assert.True(t, options.Verbose)
	assert.False(t, options.ForceSudo)
	assert.Equal(t, time.Millisecond*10, options.ThrottleDelay)
	assert.Equal(t, int64(50*1024*1024), options.MaxFileSize)
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

func TestScanStatsInitialization(t *testing.T) {
	// Test that ScanStats can be initialized with zero values
	stats := &ScanStats{}

	assert.Equal(t, 0, stats.FilesIndexed)
	assert.Equal(t, 0, stats.DirectoriesSkipped)
	assert.Equal(t, 0, stats.PermissionErrors)
	assert.Len(t, stats.SkippedPaths, 0)
	assert.Nil(t, stats.SkippedPaths) // Should be nil slice initially
}

func TestScannerInterfaceImplementation(t *testing.T) {
	// Test that FileScanner implements ScannerInterface
	fs := afero.NewMemMapFs()
	mockDB := &MockDBInterface{}

	var scanner ScannerInterface = NewFileScanner(fs, mockDB)
	assert.NotNil(t, scanner)

	// Test that the interface method exists by checking type
	_, ok := scanner.(*FileScanner)
	assert.True(t, ok, "Scanner should be of type *FileScanner")
}
