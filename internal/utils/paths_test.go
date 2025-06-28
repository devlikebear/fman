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

func TestGetSkipPatternsAllOSTypes(t *testing.T) {
	// Test that we have patterns defined for all major OS types
	// by temporarily simulating different OS values

	// We can't change runtime.GOOS during runtime, but we can test
	// that the function handles different cases properly by checking
	// the patterns contain expected values for current OS

	patterns := GetSkipPatterns()

	// All OS should have at least some patterns
	assert.Greater(t, len(patterns), 0)

	// Test that patterns are strings and not empty
	for _, pattern := range patterns {
		assert.NotEmpty(t, pattern, "Pattern should not be empty")
		assert.IsType(t, "", pattern, "Pattern should be string")
	}

	// Test specific patterns based on current OS
	currentOS := runtime.GOOS
	switch currentOS {
	case "darwin":
		// macOS specific patterns
		expectedPatterns := []string{".Trash", ".fseventsd", ".Spotlight-V100"}
		for _, expected := range expectedPatterns {
			assert.Contains(t, patterns, expected, "macOS should include %s", expected)
		}
	case "linux":
		// Linux specific patterns
		expectedPatterns := []string{"proc", "sys", "dev", "tmp"}
		for _, expected := range expectedPatterns {
			assert.Contains(t, patterns, expected, "Linux should include %s", expected)
		}
	case "windows":
		// Windows specific patterns
		expectedPatterns := []string{"$Recycle.Bin", "System Volume Information"}
		for _, expected := range expectedPatterns {
			assert.Contains(t, patterns, expected, "Windows should include %s", expected)
		}
	}
}

func TestGetSkipPatternsConsistency(t *testing.T) {
	// Test that calling GetSkipPatterns multiple times returns consistent results
	patterns1 := GetSkipPatterns()
	patterns2 := GetSkipPatterns()

	assert.Equal(t, patterns1, patterns2, "GetSkipPatterns should return consistent results")
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
		{
			name:     "should skip pattern in middle of path",
			path:     "/home/user/proc/test",
			expected: true,
		},
		{
			name:     "should skip exact pattern match",
			path:     "proc",
			expected: true,
		},
		{
			name:     "should not skip non-matching path",
			path:     "/home/user/documents/file.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipPath(tt.path, skipPatterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipPathEmptyPatterns(t *testing.T) {
	// Test with empty patterns
	result := ShouldSkipPath("/some/path", []string{})
	// Should still skip hidden directories at root level
	assert.False(t, result, "Should not skip regular path with empty patterns")

	// Test hidden directory with empty patterns
	result = ShouldSkipPath("/Users/.hidden", []string{})
	assert.True(t, result, "Should still skip hidden directories even with empty patterns")
}

func TestShouldSkipPathEdgeCases(t *testing.T) {
	patterns := []string{"test"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "empty path",
			path:     "",
			expected: true, // Empty path should be skipped (has prefix ".")
		},
		{
			name:     "root path",
			path:     "/",
			expected: false,
		},
		{
			name:     "hidden file in deep path",
			path:     "/very/deep/path/structure/that/goes/many/levels/.hidden",
			expected: false,
		},
		{
			name:     "pattern as basename",
			path:     "/some/path/test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipPath(tt.path, patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}
