/*
Copyright © 2025 changheonshin
*/
package utils

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSkipPatterns(t *testing.T) {
	patterns := GetSkipPatterns()

	// Ensure we get some patterns
	assert.Greater(t, len(patterns), 0, "Should return at least one skip pattern")

	// Check that patterns are appropriate for the current OS
	switch runtime.GOOS {
	case "darwin":
		assert.Contains(t, patterns, ".Trash", "macOS should include .Trash pattern")
		assert.Contains(t, patterns, ".fseventsd", "macOS should include .fseventsd pattern")
	case "linux":
		assert.Contains(t, patterns, "proc", "Linux should include proc pattern")
		assert.Contains(t, patterns, "sys", "Linux should include sys pattern")
	case "windows":
		assert.Contains(t, patterns, "$Recycle.Bin", "Windows should include $Recycle.Bin pattern")
	default:
		assert.Contains(t, patterns, ".Trash", "Default should include .Trash pattern")
	}
}

func TestShouldSkipPath(t *testing.T) {
	skipPatterns := []string{".Trash", "proc", "System/Library"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "should skip .Trash directory",
			path:     "/Users/test/.Trash",
			expected: true,
		},
		{
			name:     "should skip proc directory",
			path:     "/proc/1",
			expected: true,
		},
		{
			name:     "should skip System/Library path",
			path:     "/System/Library/Frameworks",
			expected: true,
		},
		{
			name:     "should skip hidden directory at root level",
			path:     "/Users/.hidden",
			expected: true,
		},
		{
			name:     "should not skip regular directory",
			path:     "/Users/test/Documents",
			expected: false,
		},
		{
			name:     "should not skip deep hidden file",
			path:     "/Users/test/Documents/deep/path/.hidden",
			expected: false,
		},
		{
			name:     "should skip shallow hidden directory",
			path:     "/Users/.config",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipPath(tt.path, skipPatterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}
