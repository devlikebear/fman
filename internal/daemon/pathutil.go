package daemon

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// PathChecker provides path normalization and conflict detection functionality
type PathChecker struct {
	// Cache for normalized paths to improve performance
	normalizedCache map[string]string
	cacheMu         sync.RWMutex
}

// NewPathChecker creates a new PathChecker instance
func NewPathChecker() *PathChecker {
	return &PathChecker{
		normalizedCache: make(map[string]string),
	}
}

// NormalizePath normalizes a path by cleaning it and resolving symbolic links
func (pc *PathChecker) NormalizePath(path string) (string, error) {
	// Check cache first
	pc.cacheMu.RLock()
	if normalized, exists := pc.normalizedCache[path]; exists {
		pc.cacheMu.RUnlock()
		return normalized, nil
	}
	pc.cacheMu.RUnlock()

	// Clean the path first
	cleaned := filepath.Clean(path)

	// Convert to absolute path
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	// Resolve symbolic links
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If symlink resolution fails, use the absolute path
		// This can happen if the path doesn't exist yet or permission denied
		resolved = abs
	}

	// For consistency, try to resolve symlinks for the parent directory
	// if the full path resolution failed
	if resolved == abs && err != nil {
		dir := filepath.Dir(abs)
		resolvedDir, dirErr := filepath.EvalSymlinks(dir)
		if dirErr == nil {
			resolved = filepath.Join(resolvedDir, filepath.Base(abs))
		}
	}

	// Normalize path separators and case (for case-insensitive filesystems)
	normalized := pc.normalizePlatformSpecific(resolved)

	// Cache the result
	pc.cacheMu.Lock()
	pc.normalizedCache[path] = normalized
	pc.cacheMu.Unlock()

	return normalized, nil
}

// normalizePlatformSpecific handles platform-specific path normalization
func (pc *PathChecker) normalizePlatformSpecific(path string) string {
	// Convert to forward slashes for consistency
	normalized := filepath.ToSlash(path)

	// On Windows and macOS (case-insensitive filesystems), convert to lowercase
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		normalized = strings.ToLower(normalized)
	}

	// Ensure trailing slash is removed for consistency
	if len(normalized) > 1 && strings.HasSuffix(normalized, "/") {
		normalized = strings.TrimSuffix(normalized, "/")
	}

	return normalized
}

// IsParentPath checks if parentPath is a parent directory of childPath
func (pc *PathChecker) IsParentPath(parentPath, childPath string) (bool, error) {
	normalizedParent, err := pc.NormalizePath(parentPath)
	if err != nil {
		return false, fmt.Errorf("failed to normalize parent path %s: %w", parentPath, err)
	}

	normalizedChild, err := pc.NormalizePath(childPath)
	if err != nil {
		return false, fmt.Errorf("failed to normalize child path %s: %w", childPath, err)
	}

	// Check if child path starts with parent path
	if normalizedParent == normalizedChild {
		return false, nil // Same path, not parent-child relationship
	}

	// Ensure parent path ends with separator for accurate comparison
	parentWithSep := normalizedParent
	if !strings.HasSuffix(parentWithSep, "/") {
		parentWithSep += "/"
	}

	return strings.HasPrefix(normalizedChild, parentWithSep), nil
}

// HasConflict checks if a new path conflicts with existing job paths
func (pc *PathChecker) HasConflict(newPath string, existingPaths []string) (*PathConflict, error) {
	normalizedNew, err := pc.NormalizePath(newPath)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize new path %s: %w", newPath, err)
	}

	for _, existingPath := range existingPaths {
		normalizedExisting, err := pc.NormalizePath(existingPath)
		if err != nil {
			// Skip paths that can't be normalized
			continue
		}

		// Check if paths are identical
		if normalizedNew == normalizedExisting {
			return &PathConflict{
				Type:         ConflictTypeDuplicate,
				ExistingPath: existingPath,
				ConflictPath: newPath,
				Reason:       "identical paths",
			}, nil
		}

		// Check if new path is a parent of existing path
		isParent, err := pc.IsParentPath(newPath, existingPath)
		if err == nil && isParent {
			return &PathConflict{
				Type:         ConflictTypeParentChild,
				ExistingPath: existingPath,
				ConflictPath: newPath,
				Reason:       "new path is parent of existing path",
			}, nil
		}

		// Check if existing path is a parent of new path
		isChild, err := pc.IsParentPath(existingPath, newPath)
		if err == nil && isChild {
			return &PathConflict{
				Type:         ConflictTypeChildParent,
				ExistingPath: existingPath,
				ConflictPath: newPath,
				Reason:       "new path is child of existing path",
			}, nil
		}
	}

	return nil, nil // No conflict found
}

// OptimizePaths removes redundant paths from a list, keeping only the most efficient set
func (pc *PathChecker) OptimizePaths(paths []string) (*PathOptimizationResult, error) {
	if len(paths) == 0 {
		return &PathOptimizationResult{
			OptimizedPaths: []string{},
			RemovedPaths:   []string{},
			Conflicts:      []*PathConflict{},
		}, nil
	}

	result := &PathOptimizationResult{
		OptimizedPaths: []string{},
		RemovedPaths:   []string{},
		Conflicts:      []*PathConflict{},
	}

	// Normalize all paths first
	normalizedPaths := make(map[string]string) // normalized -> original
	for _, path := range paths {
		normalized, err := pc.NormalizePath(path)
		if err != nil {
			// Skip paths that can't be normalized
			result.RemovedPaths = append(result.RemovedPaths, path)
			result.Conflicts = append(result.Conflicts, &PathConflict{
				Type:         ConflictTypeInvalid,
				ExistingPath: "",
				ConflictPath: path,
				Reason:       fmt.Sprintf("normalization failed: %v", err),
			})
			continue
		}

		// Check for exact duplicates
		if existingOriginal, exists := normalizedPaths[normalized]; exists {
			result.RemovedPaths = append(result.RemovedPaths, path)
			result.Conflicts = append(result.Conflicts, &PathConflict{
				Type:         ConflictTypeDuplicate,
				ExistingPath: existingOriginal,
				ConflictPath: path,
				Reason:       "duplicate after normalization",
			})
			continue
		}

		normalizedPaths[normalized] = path
	}

	// Convert to slices for easier processing
	uniquePaths := make([]string, 0, len(normalizedPaths))
	for _, original := range normalizedPaths {
		uniquePaths = append(uniquePaths, original)
	}

	// Build a map to track which paths should be kept
	shouldKeep := make(map[string]bool)
	for _, path := range uniquePaths {
		shouldKeep[path] = true
	}

	// Find parent-child relationships and mark children for removal
	for i, path1 := range uniquePaths {
		if !shouldKeep[path1] {
			continue // Already marked for removal
		}

		for j, path2 := range uniquePaths {
			if i == j || !shouldKeep[path2] {
				continue
			}

			isParent, err := pc.IsParentPath(path1, path2)
			if err != nil {
				continue
			}

			if isParent {
				// path1 is parent of path2, so remove path2
				shouldKeep[path2] = false
				result.RemovedPaths = append(result.RemovedPaths, path2)
				result.Conflicts = append(result.Conflicts, &PathConflict{
					Type:         ConflictTypeChildParent,
					ExistingPath: path1,
					ConflictPath: path2,
					Reason:       "child path removed in favor of parent",
				})
			}
		}
	}

	// Add remaining paths that should be kept
	for _, path := range uniquePaths {
		if shouldKeep[path] {
			result.OptimizedPaths = append(result.OptimizedPaths, path)
		}
	}

	return result, nil
}

// ClearCache clears the normalized path cache
func (pc *PathChecker) ClearCache() {
	pc.cacheMu.Lock()
	defer pc.cacheMu.Unlock()
	pc.normalizedCache = make(map[string]string)
}

// GetCacheSize returns the current cache size
func (pc *PathChecker) GetCacheSize() int {
	pc.cacheMu.RLock()
	defer pc.cacheMu.RUnlock()
	return len(pc.normalizedCache)
}

// ConflictType represents the type of path conflict
type ConflictType string

const (
	// ConflictTypeDuplicate indicates identical paths
	ConflictTypeDuplicate ConflictType = "duplicate"
	// ConflictTypeParentChild indicates new path is parent of existing
	ConflictTypeParentChild ConflictType = "parent_child"
	// ConflictTypeChildParent indicates new path is child of existing
	ConflictTypeChildParent ConflictType = "child_parent"
	// ConflictTypeInvalid indicates invalid path
	ConflictTypeInvalid ConflictType = "invalid"
)

// PathConflict represents a conflict between paths
type PathConflict struct {
	Type         ConflictType `json:"type"`
	ExistingPath string       `json:"existing_path"`
	ConflictPath string       `json:"conflict_path"`
	Reason       string       `json:"reason"`
}

// PathOptimizationResult contains the result of path optimization
type PathOptimizationResult struct {
	OptimizedPaths []string        `json:"optimized_paths"`
	RemovedPaths   []string        `json:"removed_paths"`
	Conflicts      []*PathConflict `json:"conflicts"`
}

// String returns a string representation of the conflict
func (pc *PathConflict) String() string {
	return fmt.Sprintf("%s conflict: %s (existing: %s, conflict: %s)",
		pc.Type, pc.Reason, pc.ExistingPath, pc.ConflictPath)
}

// String returns a string representation of the optimization result
func (por *PathOptimizationResult) String() string {
	return fmt.Sprintf("Optimized: %d paths, Removed: %d paths, Conflicts: %d",
		len(por.OptimizedPaths), len(por.RemovedPaths), len(por.Conflicts))
}
