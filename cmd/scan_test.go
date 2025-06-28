package cmd

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestScanCommand(t *testing.T) {
	// Create a new mock DB instance for each test
	mockDB := new(MockDBInterface)

	// Initialize afero filesystem
	fs := afero.NewMemMapFs()

	// Create a dummy directory structure in the mock filesystem
	_ = fs.MkdirAll("/testdir/subdir", 0755)
	_ = afero.WriteFile(fs, "/testdir/file1.txt", []byte("content1"), 0644)
	_ = afero.WriteFile(fs, "/testdir/subdir/file2.txt", []byte("content2"), 0644)

	// Expect InitDB to be called and succeed
	mockDB.On("InitDB").Return(nil).Once()
	mockDB.On("Close").Return(nil).Once()

	// Expect UpsertFile to be called for each file
	mockDB.On("UpsertFile", mock.AnythingOfType("*db.File")).Return(nil).Times(2)

	// Execute the runScan function directly
	err := runScan(nil, []string{"/testdir"}, fs, mockDB)
	assert.NoError(t, err)

	// Assert that mock expectations were met
	mockDB.AssertExpectations(t)
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
