/*
Copyright © 2025 changheonshin
*/
package utils

import (
	"path/filepath"
	"runtime"
	"strings"
)

// GetSkipPatterns returns a list of directory patterns to skip based on the OS
func GetSkipPatterns() []string {
	switch runtime.GOOS {
	case "darwin": // macOS
		return []string{
			".Trash", ".Trashes",
			".fseventsd",
			".Spotlight-V100",
			".DocumentRevisions-V100",
			".TemporaryItems",
			".DS_Store",
			"System/Library",
			"Library/Caches",
			"private/var/vm",
		}
	case "linux":
		return []string{
			".cache", ".local/share/Trash",
			"proc", "sys", "dev",
			"tmp", "var/tmp",
			"run", "mnt",
		}
	case "windows":
		return []string{
			"$Recycle.Bin",
			"System Volume Information",
			"pagefile.sys",
			"hiberfil.sys",
			"swapfile.sys",
		}
	default:
		return []string{".Trash", ".cache", "tmp"}
	}
}

// ShouldSkipPath checks if a path should be skipped based on patterns
func ShouldSkipPath(path string, skipPatterns []string) bool {
	pathBase := filepath.Base(path)

	for _, pattern := range skipPatterns {
		if strings.Contains(path, pattern) || pathBase == pattern {
			return true
		}
	}

	// Skip hidden directories in root level scans (but allow deeper hidden files)
	if strings.HasPrefix(pathBase, ".") && strings.Count(path, string(filepath.Separator)) <= 3 {
		return true
	}

	return false
}
