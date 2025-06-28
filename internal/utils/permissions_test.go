/*
Copyright © 2025 changheonshin
*/
package utils

import (
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRunningAsRoot(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "Check if running as root",
			expected: os.Geteuid() == 0 && runtime.GOOS != "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRunningAsRoot()
			if runtime.GOOS == "windows" {
				assert.False(t, result, "Windows should always return false")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "EACCES error",
			err:      syscall.EACCES,
			expected: true,
		},
		{
			name:     "EPERM error",
			err:      syscall.EPERM,
			expected: true,
		},
		{
			name:     "permission denied string",
			err:      &os.PathError{Op: "open", Path: "/test", Err: syscall.EACCES},
			expected: true,
		},
		{
			name:     "other error",
			err:      syscall.ENOENT,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPermissionError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: RunWithSudo is difficult to test in unit tests as it requires
// interactive input and sudo execution. In a real testing environment,
// you would typically test this with integration tests or mock the
// interactive parts.
