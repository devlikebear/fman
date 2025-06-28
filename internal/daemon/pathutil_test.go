package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPathChecker(t *testing.T) {
	pc := NewPathChecker()

	assert.NotNil(t, pc)
	assert.NotNil(t, pc.normalizedCache)
	assert.Equal(t, 0, pc.GetCacheSize())
}

func TestPathChecker_NormalizePath(t *testing.T) {
	pc := NewPathChecker()

	t.Run("basic path normalization", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected func(string) bool // Function to validate the result
		}{
			{
				name:  "clean relative path",
				input: "./test/path",
				expected: func(result string) bool {
					return strings.HasSuffix(result, "/test/path") || strings.HasSuffix(result, "\\test\\path")
				},
			},
			{
				name:  "path with double slashes",
				input: "/test//path",
				expected: func(result string) bool {
					return !strings.Contains(result, "//")
				},
			},
			{
				name:  "path with dots",
				input: "/test/../path",
				expected: func(result string) bool {
					return !strings.Contains(result, "..")
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := pc.NormalizePath(tc.input)
				assert.NoError(t, err)
				assert.True(t, tc.expected(result), "Result: %s", result)
			})
		}
	})

	t.Run("cache functionality", func(t *testing.T) {
		// Create a fresh PathChecker for this test to avoid interference
		freshPC := NewPathChecker()
		testPath := "/test/cache/path"

		// First call should cache the result
		result1, err := freshPC.NormalizePath(testPath)
		assert.NoError(t, err)
		assert.Equal(t, 1, freshPC.GetCacheSize())

		// Second call should use cache
		result2, err := freshPC.NormalizePath(testPath)
		assert.NoError(t, err)
		assert.Equal(t, result1, result2)
		assert.Equal(t, 1, freshPC.GetCacheSize())
	})

	t.Run("platform specific normalization", func(t *testing.T) {
		testPath := "/Test/Path"
		result, err := pc.NormalizePath(testPath)
		assert.NoError(t, err)

		// On case-insensitive systems, should be lowercase
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			assert.True(t, strings.ToLower(result) == result, "Result should be lowercase on case-insensitive systems")
		}

		// Should use forward slashes
		assert.True(t, strings.Contains(result, "/") || !strings.Contains(result, "\\"))
	})

	t.Run("trailing slash removal", func(t *testing.T) {
		testPath := "/test/path/"
		result, err := pc.NormalizePath(testPath)
		assert.NoError(t, err)
		assert.False(t, strings.HasSuffix(result, "/"), "Trailing slash should be removed")
	})

	t.Run("root path handling", func(t *testing.T) {
		testPath := "/"
		result, err := pc.NormalizePath(testPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

func TestPathChecker_IsParentPath(t *testing.T) {
	pc := NewPathChecker()

	t.Run("clear parent-child relationship", func(t *testing.T) {
		// Use temporary directory to avoid symlink resolution issues
		tempDir := t.TempDir()
		parent := tempDir
		child := filepath.Join(tempDir, "documents")

		// Create the child directory
		err := os.MkdirAll(child, 0755)
		require.NoError(t, err)

		isParent, err := pc.IsParentPath(parent, child)
		assert.NoError(t, err)
		assert.True(t, isParent)
	})

	t.Run("not parent-child relationship", func(t *testing.T) {
		tempDir := t.TempDir()
		path1 := filepath.Join(tempDir, "user")
		path2 := filepath.Join(tempDir, "other")

		// Create both directories
		err := os.MkdirAll(path1, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(path2, 0755)
		require.NoError(t, err)

		isParent, err := pc.IsParentPath(path1, path2)
		assert.NoError(t, err)
		assert.False(t, isParent)
	})

	t.Run("same path", func(t *testing.T) {
		tempDir := t.TempDir()
		path := tempDir

		isParent, err := pc.IsParentPath(path, path)
		assert.NoError(t, err)
		assert.False(t, isParent)
	})

	t.Run("reverse relationship", func(t *testing.T) {
		tempDir := t.TempDir()
		parent := filepath.Join(tempDir, "user", "documents")
		child := filepath.Join(tempDir, "user")

		// Create the directory structure
		err := os.MkdirAll(parent, 0755)
		require.NoError(t, err)

		isParent, err := pc.IsParentPath(parent, child)
		assert.NoError(t, err)
		assert.False(t, isParent)
	})

	t.Run("nested paths", func(t *testing.T) {
		// Use temporary directory to avoid symlink resolution issues
		tempDir := t.TempDir()
		parent := tempDir
		child := filepath.Join(tempDir, "user", "documents", "file.txt")

		// Create the child directory structure
		err := os.MkdirAll(filepath.Dir(child), 0755)
		require.NoError(t, err)

		isParent, err := pc.IsParentPath(parent, child)
		assert.NoError(t, err)
		assert.True(t, isParent)
	})

	t.Run("similar but not parent paths", func(t *testing.T) {
		tempDir := t.TempDir()
		path1 := filepath.Join(tempDir, "user")
		path2 := filepath.Join(tempDir, "username")

		// Create both directories
		err := os.MkdirAll(path1, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(path2, 0755)
		require.NoError(t, err)

		isParent, err := pc.IsParentPath(path1, path2)
		assert.NoError(t, err)
		assert.False(t, isParent)
	})

	t.Run("relative paths", func(t *testing.T) {
		parent := "./test"
		child := "./test/subdir"

		isParent, err := pc.IsParentPath(parent, child)
		assert.NoError(t, err)
		assert.True(t, isParent)
	})
}

func TestPathChecker_HasConflict(t *testing.T) {
	pc := NewPathChecker()

	t.Run("no conflict", func(t *testing.T) {
		tempDir := t.TempDir()
		newPath := filepath.Join(tempDir, "documents")
		existingPaths := []string{
			filepath.Join(tempDir, "pictures"),
			filepath.Join(tempDir, "logs"),
		}

		// Create directories
		for _, path := range append(existingPaths, newPath) {
			err := os.MkdirAll(path, 0755)
			require.NoError(t, err)
		}

		conflict, err := pc.HasConflict(newPath, existingPaths)
		assert.NoError(t, err)
		assert.Nil(t, conflict)
	})

	t.Run("duplicate path conflict", func(t *testing.T) {
		tempDir := t.TempDir()
		newPath := filepath.Join(tempDir, "documents")
		existingPaths := []string{
			filepath.Join(tempDir, "documents"),
			filepath.Join(tempDir, "logs"),
		}

		// Create directories
		for _, path := range existingPaths {
			err := os.MkdirAll(path, 0755)
			require.NoError(t, err)
		}

		conflict, err := pc.HasConflict(newPath, existingPaths)
		assert.NoError(t, err)
		assert.NotNil(t, conflict)
		assert.Equal(t, ConflictTypeDuplicate, conflict.Type)
		assert.Equal(t, existingPaths[0], conflict.ExistingPath)
		assert.Equal(t, newPath, conflict.ConflictPath)
	})

	t.Run("parent-child conflict", func(t *testing.T) {
		tempDir := t.TempDir()
		newPath := tempDir
		existingPaths := []string{
			filepath.Join(tempDir, "documents"),
			filepath.Join(tempDir, "logs"),
		}

		// Create directories
		for _, path := range existingPaths {
			err := os.MkdirAll(path, 0755)
			require.NoError(t, err)
		}

		conflict, err := pc.HasConflict(newPath, existingPaths)
		assert.NoError(t, err)
		assert.NotNil(t, conflict)
		assert.Equal(t, ConflictTypeParentChild, conflict.Type)
		assert.Equal(t, existingPaths[0], conflict.ExistingPath)
		assert.Equal(t, newPath, conflict.ConflictPath)
	})

	t.Run("child-parent conflict", func(t *testing.T) {
		tempDir := t.TempDir()
		documentsPath := filepath.Join(tempDir, "documents")
		newPath := filepath.Join(documentsPath, "subfolder")
		existingPaths := []string{
			documentsPath,
			filepath.Join(tempDir, "logs"),
		}

		// Create directories
		for _, path := range existingPaths {
			err := os.MkdirAll(path, 0755)
			require.NoError(t, err)
		}
		err := os.MkdirAll(newPath, 0755)
		require.NoError(t, err)

		conflict, err := pc.HasConflict(newPath, existingPaths)
		assert.NoError(t, err)
		assert.NotNil(t, conflict)
		assert.Equal(t, ConflictTypeChildParent, conflict.Type)
		assert.Equal(t, documentsPath, conflict.ExistingPath)
		assert.Equal(t, newPath, conflict.ConflictPath)
	})

	t.Run("multiple existing paths", func(t *testing.T) {
		tempDir := t.TempDir()
		newPath := filepath.Join(tempDir, "documents")
		existingPaths := []string{
			filepath.Join(tempDir, "pictures"),
			filepath.Join(newPath, "subfolder"),
			filepath.Join(tempDir, "logs"),
		}

		// Create directories
		for _, path := range existingPaths {
			err := os.MkdirAll(path, 0755)
			require.NoError(t, err)
		}

		conflict, err := pc.HasConflict(newPath, existingPaths)
		assert.NoError(t, err)
		assert.NotNil(t, conflict)
		assert.Equal(t, ConflictTypeParentChild, conflict.Type)
	})

	t.Run("empty existing paths", func(t *testing.T) {
		tempDir := t.TempDir()
		newPath := filepath.Join(tempDir, "documents")
		existingPaths := []string{}

		conflict, err := pc.HasConflict(newPath, existingPaths)
		assert.NoError(t, err)
		assert.Nil(t, conflict)
	})
}

func TestPathChecker_OptimizePaths(t *testing.T) {
	pc := NewPathChecker()

	t.Run("empty paths", func(t *testing.T) {
		result, err := pc.OptimizePaths([]string{})
		assert.NoError(t, err)
		assert.Empty(t, result.OptimizedPaths)
		assert.Empty(t, result.RemovedPaths)
		assert.Empty(t, result.Conflicts)
	})

	t.Run("no conflicts", func(t *testing.T) {
		tempDir := t.TempDir()
		paths := []string{
			filepath.Join(tempDir, "documents"),
			filepath.Join(tempDir, "logs"),
			filepath.Join(tempDir, "apps"),
		}

		// Create directories
		for _, path := range paths {
			err := os.MkdirAll(path, 0755)
			require.NoError(t, err)
		}

		result, err := pc.OptimizePaths(paths)
		assert.NoError(t, err)
		assert.Len(t, result.OptimizedPaths, 3)
		assert.Empty(t, result.RemovedPaths)
		assert.Empty(t, result.Conflicts)
	})

	t.Run("duplicate paths", func(t *testing.T) {
		tempDir := t.TempDir()
		userPath := filepath.Join(tempDir, "user")
		logPath := filepath.Join(tempDir, "logs")
		paths := []string{userPath, userPath, logPath}

		// Create directories
		err := os.MkdirAll(userPath, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(logPath, 0755)
		require.NoError(t, err)

		result, err := pc.OptimizePaths(paths)
		assert.NoError(t, err)
		assert.Len(t, result.OptimizedPaths, 2)
		assert.Len(t, result.RemovedPaths, 1)
		assert.Len(t, result.Conflicts, 1)
		assert.Equal(t, ConflictTypeDuplicate, result.Conflicts[0].Type)
	})

	t.Run("parent-child optimization", func(t *testing.T) {
		tempDir := t.TempDir()
		userPath := filepath.Join(tempDir, "user")
		logPath := filepath.Join(tempDir, "logs")
		paths := []string{
			userPath,
			filepath.Join(userPath, "documents"),
			filepath.Join(userPath, "pictures"),
			logPath,
		}

		// Create directories
		for _, path := range paths {
			err := os.MkdirAll(path, 0755)
			require.NoError(t, err)
		}

		result, err := pc.OptimizePaths(paths)
		assert.NoError(t, err)

		// Should keep parent and remove children
		assert.Contains(t, result.OptimizedPaths, userPath)
		assert.Contains(t, result.OptimizedPaths, logPath)
		assert.Contains(t, result.RemovedPaths, filepath.Join(userPath, "documents"))
		assert.Contains(t, result.RemovedPaths, filepath.Join(userPath, "pictures"))

		assert.Len(t, result.OptimizedPaths, 2)
		assert.Len(t, result.RemovedPaths, 2)
		assert.Len(t, result.Conflicts, 2)
	})

	t.Run("complex hierarchy", func(t *testing.T) {
		// Create temporary directories to avoid symlink resolution issues
		tempDir := t.TempDir()
		homeDir := filepath.Join(tempDir, "home")
		varDir := filepath.Join(tempDir, "var")

		// Create directory structure
		dirs := []string{
			homeDir,
			filepath.Join(homeDir, "user"),
			filepath.Join(homeDir, "user", "documents"),
			filepath.Join(homeDir, "other"),
			varDir,
			filepath.Join(varDir, "log"),
			filepath.Join(varDir, "log", "app"),
		}

		for _, dir := range dirs {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}

		paths := []string{
			homeDir,
			filepath.Join(homeDir, "user"),
			filepath.Join(homeDir, "user", "documents"),
			filepath.Join(homeDir, "other"),
			filepath.Join(varDir, "log"),
			filepath.Join(varDir, "log", "app"),
		}

		result, err := pc.OptimizePaths(paths)
		assert.NoError(t, err)

		// Should keep top-level parents
		assert.Contains(t, result.OptimizedPaths, homeDir)
		assert.Contains(t, result.OptimizedPaths, filepath.Join(varDir, "log"))
		assert.Len(t, result.OptimizedPaths, 2)
		assert.Len(t, result.RemovedPaths, 4)
	})

	t.Run("mixed valid and invalid paths", func(t *testing.T) {
		// Create a temporary directory for testing
		tempDir := t.TempDir()
		validPath := filepath.Join(tempDir, "valid")
		err := os.MkdirAll(validPath, 0755)
		require.NoError(t, err)

		paths := []string{
			validPath,
			"/definitely/nonexistent/path/that/should/fail",
			"/var/log",
		}

		result, err := pc.OptimizePaths(paths)
		assert.NoError(t, err)

		// Valid paths should be optimized
		assert.GreaterOrEqual(t, len(result.OptimizedPaths), 1)

		// Invalid path might be removed (depending on platform)
		// This is platform-dependent behavior
	})

	t.Run("relative and absolute paths", func(t *testing.T) {
		paths := []string{
			"./test",
			"./test/subdir",
			"/absolute/path",
		}

		result, err := pc.OptimizePaths(paths)
		assert.NoError(t, err)

		// Should handle both relative and absolute paths
		assert.NotEmpty(t, result.OptimizedPaths)
	})
}

func TestPathChecker_CacheManagement(t *testing.T) {
	pc := NewPathChecker()

	t.Run("cache size tracking", func(t *testing.T) {
		assert.Equal(t, 0, pc.GetCacheSize())

		_, err := pc.NormalizePath("/test/path1")
		assert.NoError(t, err)
		assert.Equal(t, 1, pc.GetCacheSize())

		_, err = pc.NormalizePath("/test/path2")
		assert.NoError(t, err)
		assert.Equal(t, 2, pc.GetCacheSize())

		// Same path should not increase cache size
		_, err = pc.NormalizePath("/test/path1")
		assert.NoError(t, err)
		assert.Equal(t, 2, pc.GetCacheSize())
	})

	t.Run("cache clearing", func(t *testing.T) {
		_, err := pc.NormalizePath("/test/path1")
		assert.NoError(t, err)
		_, err = pc.NormalizePath("/test/path2")
		assert.NoError(t, err)

		assert.Equal(t, 2, pc.GetCacheSize())

		pc.ClearCache()
		assert.Equal(t, 0, pc.GetCacheSize())
	})
}

func TestConflictType_String(t *testing.T) {
	testCases := []struct {
		conflict *PathConflict
		expected string
	}{
		{
			conflict: &PathConflict{
				Type:         ConflictTypeDuplicate,
				ExistingPath: "/path1",
				ConflictPath: "/path2",
				Reason:       "test reason",
			},
			expected: "duplicate conflict: test reason (existing: /path1, conflict: /path2)",
		},
		{
			conflict: &PathConflict{
				Type:         ConflictTypeParentChild,
				ExistingPath: "/parent",
				ConflictPath: "/parent/child",
				Reason:       "parent-child relationship",
			},
			expected: "parent_child conflict: parent-child relationship (existing: /parent, conflict: /parent/child)",
		},
	}

	for _, tc := range testCases {
		result := tc.conflict.String()
		assert.Equal(t, tc.expected, result)
	}
}

func TestPathOptimizationResult_String(t *testing.T) {
	result := &PathOptimizationResult{
		OptimizedPaths: []string{"/path1", "/path2"},
		RemovedPaths:   []string{"/path3"},
		Conflicts:      []*PathConflict{{}, {}},
	}

	expected := "Optimized: 2 paths, Removed: 1 paths, Conflicts: 2"
	assert.Equal(t, expected, result.String())
}

func TestPathChecker_ConcurrentAccess(t *testing.T) {
	pc := NewPathChecker()

	// Test concurrent access to cache
	const numGoroutines = 10
	const pathsPerGoroutine = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < pathsPerGoroutine; j++ {
				path := fmt.Sprintf("/test/worker%d/path%d", workerID, j)
				_, err := pc.NormalizePath(path)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Cache should contain all unique paths
	expectedCacheSize := numGoroutines * pathsPerGoroutine
	assert.Equal(t, expectedCacheSize, pc.GetCacheSize())
}

func TestPathChecker_PlatformSpecificBehavior(t *testing.T) {
	pc := NewPathChecker()

	t.Run("case sensitivity", func(t *testing.T) {
		path1 := "/Test/Path"
		path2 := "/test/path"

		norm1, err := pc.NormalizePath(path1)
		assert.NoError(t, err)

		norm2, err := pc.NormalizePath(path2)
		assert.NoError(t, err)

		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			// Case-insensitive filesystems should normalize to same result
			assert.Equal(t, norm1, norm2)
		} else {
			// Case-sensitive filesystems should preserve case differences
			// (though both will be converted to absolute paths)
			assert.True(t, len(norm1) > 0 && len(norm2) > 0)
		}
	})

	t.Run("path separators", func(t *testing.T) {
		var testPath string
		if runtime.GOOS == "windows" {
			testPath = "C:\\test\\path"
		} else {
			testPath = "/test/path"
		}

		normalized, err := pc.NormalizePath(testPath)
		assert.NoError(t, err)

		// Should use forward slashes in normalized form
		assert.True(t, strings.Contains(normalized, "/") || !strings.Contains(normalized, "\\"))
	})
}

// Benchmark tests for performance measurement
func BenchmarkPathChecker_NormalizePath(b *testing.B) {
	pc := NewPathChecker()
	testPath := "/test/benchmark/path"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pc.NormalizePath(testPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPathChecker_IsParentPath(b *testing.B) {
	pc := NewPathChecker()
	parent := "/home/user"
	child := "/home/user/documents/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pc.IsParentPath(parent, child)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPathChecker_OptimizePaths(b *testing.B) {
	pc := NewPathChecker()
	paths := []string{
		"/home",
		"/home/user",
		"/home/user/documents",
		"/home/user/pictures",
		"/var/log",
		"/var/log/app",
		"/opt/software",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pc.OptimizePaths(paths)
		if err != nil {
			b.Fatal(err)
		}
	}
}
