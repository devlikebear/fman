package cmd

import (
	"github.com/devlikebear/fman/internal/db"
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

func (m *MockDBInterface) FindFilesWithHashes(searchDir string, minSize int64) ([]db.File, error) {
	args := m.Called(searchDir, minSize)
	return args.Get(0).([]db.File), args.Error(1)
}

func (m *MockDBInterface) FindFilesByAdvancedCriteria(criteria db.SearchCriteria) ([]db.File, error) {
	args := m.Called(criteria)
	return args.Get(0).([]db.File), args.Error(1)
}

func (m *MockDBInterface) Close() error {
	args := m.Called()
	return args.Error(0)
}
