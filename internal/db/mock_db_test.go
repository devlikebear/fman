package db

import (
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of the DBInterface for testing.
type MockDB struct {
	mock.Mock
}

// InitDB mocks the InitDB method.
func (m *MockDB) InitDB() error {
	args := m.Called()
	return args.Error(0)
}

// UpsertFile mocks the UpsertFile method.
func (m *MockDB) UpsertFile(file *File) error {
	args := m.Called(file)
	return args.Error(0)
}

// FindFilesByName mocks the FindFilesByName method.
func (m *MockDB) FindFilesByName(namePattern string) ([]File, error) {
	args := m.Called(namePattern)
	return args.Get(0).([]File), args.Error(1)
}

// FindFilesWithHashes mocks the FindFilesWithHashes method.
func (m *MockDB) FindFilesWithHashes(searchDir string, minSize int64) ([]File, error) {
	args := m.Called(searchDir, minSize)
	return args.Get(0).([]File), args.Error(1)
}

// Close mocks the Close method.
func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}
