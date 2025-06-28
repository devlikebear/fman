package cmd

import (
	"fmt"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBInterface is a mock implementation of db.DBInterface for testing
type MockDBInterface struct {
	mock.Mock
}

func (m *MockDBInterface) InitDB() error {
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

func (m *MockDBInterface) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestFindCommand(t *testing.T) {
	// Create a new mock DB instance for each test
	mockDB := new(MockDBInterface)

	// Expect InitDB to be called and succeed
	mockDB.On("InitDB").Return(nil).Once()
	mockDB.On("Close").Return(nil).Once()

	// Prepare mock data for FindFilesByName
	mockFiles := []db.File{
		{
			Path:       "/path/to/document.pdf",
			Name:       "document.pdf",
			Size:       1024,
			ModifiedAt: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			FileHash:   "hash1",
		},
		{
			Path:       "/another/report.docx",
			Name:       "report.docx",
			Size:       2048,
			ModifiedAt: time.Date(2023, 2, 15, 11, 30, 0, 0, time.UTC),
			FileHash:   "hash2",
		},
	}
	mockDB.On("FindFilesByName", "doc").Return(mockFiles, nil).Once()

	// Execute the command
	err := runFind(nil, []string{"doc"}, mockDB)
	assert.NoError(t, err)

	// Assert that mock expectations were met
	mockDB.AssertExpectations(t)
}

func TestFindCommand_NoFilesFound(t *testing.T) {
	mockDB := new(MockDBInterface)

	mockDB.On("InitDB").Return(nil).Once()
	mockDB.On("Close").Return(nil).Once()
	mockDB.On("FindFilesByName", "nonexistent").Return([]db.File{}, nil).Once()

	err := runFind(nil, []string{"nonexistent"}, mockDB)
	assert.NoError(t, err)

	mockDB.AssertExpectations(t)
}

func TestFindCommand_InitDBError(t *testing.T) {
	mockDB := new(MockDBInterface)

	mockDB.On("InitDB").Return(fmt.Errorf("db init error")).Once()

	err := runFind(nil, []string{"test"}, mockDB)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize database")

	mockDB.AssertExpectations(t)
}
