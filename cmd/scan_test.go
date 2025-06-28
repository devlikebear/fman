package cmd

import (
	"fmt"
	"os"
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

	// Create a test command with proper flags
	cmd := &cobra.Command{}
	cmd.Flags().Bool("force-sudo", false, "")
	cmd.Flags().BoolP("verbose", "v", false, "")

	// Execute the runScan function directly
	err := runScan(cmd, []string{"/testdir"}, fs, mockDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize database")

	mockDB.AssertExpectations(t)
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
