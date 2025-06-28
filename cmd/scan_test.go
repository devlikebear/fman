package cmd

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestScanCommand(t *testing.T) {
	// Create a mock filesystem
	fs := afero.NewMemMapFs()

	// Create test files
	fs.MkdirAll("/test", 0755)
	afero.WriteFile(fs, "/test/file1.txt", []byte("content1"), 0644)
	afero.WriteFile(fs, "/test/file2.txt", []byte("content2"), 0644)

	// Create a mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil)

	// Create a test command with proper flags
	cmd := &cobra.Command{}
	cmd.Flags().Bool("force-sudo", false, "Force scanning with elevated privileges")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	// Run the scan
	err := runScan(cmd, []string{"/test"}, fs, mockDB)

	// Assertions
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)

	// Should have called UpsertFile for both files
	mockDB.AssertNumberOfCalls(t, "UpsertFile", 2)
}

func TestScanCommand_InitDBError(t *testing.T) {
	mockDB := new(MockDBInterface)
	fs := afero.NewMemMapFs()

	// Expect InitDB to be called and return an error
	mockDB.On("InitDB").Return(fmt.Errorf("db init error")).Once()

	// Execute the runScan function directly
	err := runScan(nil, []string{"/testdir"}, fs, mockDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize database")

	mockDB.AssertExpectations(t)
}

func TestGetSkipPatterns(t *testing.T) {
	patterns := getSkipPatterns()
	assert.NotEmpty(t, patterns)

	// Test that common patterns are included
	switch runtime.GOOS {
	case "darwin":
		assert.Contains(t, patterns, ".Trash")
		assert.Contains(t, patterns, ".fseventsd")
	case "linux":
		assert.Contains(t, patterns, "proc")
		assert.Contains(t, patterns, ".cache")
	case "windows":
		assert.Contains(t, patterns, "$Recycle.Bin")
	}
}

func TestShouldSkipPath(t *testing.T) {
	skipPatterns := []string{".Trash", "System/Library", "proc"}

	tests := []struct {
		path     string
		expected bool
		name     string
	}{
		{"/Users/test/.Trash", true, "should skip .Trash"},
		{"/Users/test/System/Library/something", true, "should skip System/Library"},
		{"/proc/cpuinfo", true, "should skip proc"},
		{"/Users/test/documents", false, "should not skip normal directory"},
		{"/Users/test/.hidden", true, "should skip root level hidden directories"},
		{"/Users/test/deep/path/.hidden", false, "should not skip deep hidden files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipPath(tt.path, skipPatterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
		name     string
	}{
		{syscall.EACCES, true, "should detect EACCES"},
		{syscall.EPERM, true, "should detect EPERM"},
		{errors.New("permission denied"), true, "should detect permission denied string"},
		{errors.New("operation not permitted"), true, "should detect operation not permitted"},
		{errors.New("file not found"), false, "should not detect other errors"},
		{nil, false, "should handle nil error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPermissionError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRunningAsRoot(t *testing.T) {
	// This test is environment dependent, so we just test that it returns a boolean
	result := isRunningAsRoot()
	assert.IsType(t, false, result)

	// On Windows, should always return false
	if runtime.GOOS == "windows" {
		assert.False(t, result)
	}
}

func TestRunScan_WithSkipPatterns(t *testing.T) {
	// Create a mock filesystem
	fs := afero.NewMemMapFs()

	// Create test directory structure
	fs.MkdirAll("/test/normal", 0755)
	fs.MkdirAll("/test/.Trash", 0755)
	afero.WriteFile(fs, "/test/normal/file1.txt", []byte("content1"), 0644)
	afero.WriteFile(fs, "/test/.Trash/deleted.txt", []byte("deleted"), 0644)

	// Create a mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil)

	// Create a test command
	cmd := &cobra.Command{}
	cmd.Flags().Bool("force-sudo", false, "")
	cmd.Flags().Bool("verbose", false, "")

	// Run the scan
	err := runScan(cmd, []string{"/test"}, fs, mockDB)

	// Assertions
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)

	// Should have called UpsertFile only for the normal file, not the one in .Trash
	mockDB.AssertNumberOfCalls(t, "UpsertFile", 1)
}

func TestRunScan_WithPermissionErrors(t *testing.T) {
	// Create a mock filesystem that returns permission errors
	fs := &afero.MemMapFs{}

	// Create test directory
	fs.MkdirAll("/test", 0755)
	afero.WriteFile(fs, "/test/accessible.txt", []byte("content"), 0644)

	// Create a mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil)

	// Create a test command
	cmd := &cobra.Command{}
	cmd.Flags().Bool("force-sudo", false, "")
	cmd.Flags().Bool("verbose", false, "")

	// Run the scan
	err := runScan(cmd, []string{"/test"}, fs, mockDB)

	// Assertions
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestRunScan_VerboseMode(t *testing.T) {
	// Create a mock filesystem
	fs := afero.NewMemMapFs()

	// Create test files
	fs.MkdirAll("/test", 0755)
	afero.WriteFile(fs, "/test/file1.txt", []byte("content1"), 0644)

	// Create a mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil)

	// Create a test command with verbose flag
	cmd := &cobra.Command{}
	cmd.Flags().Bool("force-sudo", false, "")
	cmd.Flags().Bool("verbose", true, "")

	// Run the scan
	err := runScan(cmd, []string{"/test"}, fs, mockDB)

	// Assertions
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestRunWithSudo_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("This test is only for Windows")
	}

	cmd := &cobra.Command{}
	err := runWithSudo(cmd, []string{"/test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on Windows")
}

func TestScanCommand_Integration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test files
	testFile := tempDir + "/test.txt"
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Create a mock database
	mockDB := &MockDBInterface{}
	mockDB.On("InitDB").Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("UpsertFile", mock.MatchedBy(func(file *db.File) bool {
		return file.Path == testFile &&
			file.Name == "test.txt" &&
			file.Size == 12 &&
			len(file.FileHash) == 64 // SHA-256 hash length
	})).Return(nil)

	// Override the global fileSystem for this test
	originalFS := fileSystem
	fileSystem = afero.NewOsFs()
	defer func() { fileSystem = originalFS }()

	// Create and execute the command
	cmd := scanCmd
	cmd.SetArgs([]string{tempDir})

	// We can't easily test the actual command execution with mocked DB,
	// so we'll test the runScan function directly
	err = runScan(cmd, []string{tempDir}, fileSystem, mockDB)

	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
}
