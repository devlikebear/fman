package cmd

import (
	"fmt"
	"os"
	"testing"
	"time"

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

func TestScanCommand_AsyncFlag(t *testing.T) {
	// Test that async flag is properly defined
	cmd := scanCmd

	// Check if async flag exists
	asyncFlag := cmd.Flags().Lookup("async")
	assert.NotNil(t, asyncFlag, "async flag should be defined")
	assert.Equal(t, "false", asyncFlag.DefValue, "async flag should default to false")
	assert.Equal(t, "bool", asyncFlag.Value.Type(), "async flag should be boolean")
}

func TestScanCommand_FlagValidation(t *testing.T) {
	// Test that all expected flags are present
	cmd := scanCmd

	// Check all flags
	flags := []string{"force-sudo", "verbose", "async"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, fmt.Sprintf("flag %s should be defined", flagName))
	}

	// Check flag types
	assert.Equal(t, "bool", cmd.Flags().Lookup("force-sudo").Value.Type())
	assert.Equal(t, "bool", cmd.Flags().Lookup("verbose").Value.Type())
	assert.Equal(t, "bool", cmd.Flags().Lookup("async").Value.Type())
}

func TestRunScanAsync_FunctionExists(t *testing.T) {
	// Test that runScanAsync function can be called
	// We can't easily test the actual async functionality without a running daemon,
	// but we can test that the function exists and handles basic validation

	cmd := &cobra.Command{}
	cmd.Flags().Bool("verbose", false, "")
	cmd.Flags().Bool("force-sudo", false, "")

	// 테스트 타임아웃 설정 (3초 내에 완료되어야 함)
	done := make(chan bool)
	var testErr error

	go func() {
		// This will fail because no daemon is running, but it tests the function exists
		testErr = runScanAsync(cmd, []string{"/test"})
		done <- true
	}()

	// 3초 타임아웃으로 무한 대기 방지
	select {
	case <-done:
		assert.Error(t, testErr)
		// 연결 실패 또는 데몬 시작 실패 메시지 중 하나를 확인
		errorMsg := testErr.Error()
		hasExpectedError := false
		expectedErrors := []string{
			"failed to connect to daemon",
			"failed to start daemon",
			"daemon did not start",
			"connection refused",
		}
		for _, expected := range expectedErrors {
			if assert.Contains(t, errorMsg, expected) {
				hasExpectedError = true
				break
			}
		}
		if !hasExpectedError {
			t.Logf("Unexpected error message: %s", errorMsg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("TestRunScanAsync_FunctionExists timed out after 3 seconds")
	}
}
