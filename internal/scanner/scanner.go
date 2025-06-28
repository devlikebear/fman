/*
Copyright © 2025 changheonshin
*/
package scanner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devlikebear/fman/internal/db"
	"github.com/devlikebear/fman/internal/utils"
	"github.com/spf13/afero"
)

// ScanStats holds statistics about the scanning process
type ScanStats struct {
	FilesIndexed       int
	DirectoriesSkipped int
	PermissionErrors   int
	SkippedPaths       []string
}

// ScanOptions contains options for scanning
type ScanOptions struct {
	Verbose   bool
	ForceSudo bool
}

// ScannerInterface defines the interface for file scanning operations
type ScannerInterface interface {
	ScanDirectory(ctx context.Context, rootDir string, options *ScanOptions) (*ScanStats, error)
}

// FileScanner implements the ScannerInterface
type FileScanner struct {
	fs       afero.Fs
	database db.DBInterface
}

// NewFileScanner creates a new FileScanner instance
func NewFileScanner(fs afero.Fs, database db.DBInterface) *FileScanner {
	return &FileScanner{
		fs:       fs,
		database: database,
	}
}

// ScanDirectory scans a directory and indexes file metadata into the database
func (s *FileScanner) ScanDirectory(ctx context.Context, rootDir string, options *ScanOptions) (*ScanStats, error) {
	// Initialize DB
	if err := s.database.InitDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	defer s.database.Close() // Ensure DB connection is closed

	skipPatterns := utils.GetSkipPatterns()
	stats := &ScanStats{}

	fmt.Printf("Starting scan of directory: %s\n", rootDir)
	if options.Verbose {
		fmt.Printf("Skip patterns: %v\n", skipPatterns)
	}

	if utils.IsRunningAsRoot() {
		fmt.Printf("🔐 Running with elevated privileges\n")
	}

	err := afero.Walk(s.fs, rootDir, func(path string, info os.FileInfo, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Handle permission errors gracefully
		if err != nil {
			if utils.IsPermissionError(err) {
				stats.PermissionErrors++
				stats.SkippedPaths = append(stats.SkippedPaths, path)
				if options.Verbose {
					fmt.Printf("⚠️  Permission denied, skipping: %s\n", path)
				}
				return nil // Continue walking, skip this path
			}
			return err // Return other errors
		}

		// Skip special directories (even when running as root, unless verbose)
		if info.IsDir() && utils.ShouldSkipPath(path, skipPatterns) {
			// When running as root, still skip system directories unless explicitly verbose
			if !utils.IsRunningAsRoot() || !options.Verbose {
				stats.DirectoriesSkipped++
				stats.SkippedPaths = append(stats.SkippedPaths, path)
				if options.Verbose {
					fmt.Printf("⏭️  Skipping special directory: %s\n", path)
				}
				return filepath.SkipDir
			}
		}

		// Process files
		if !info.IsDir() {
			if options.Verbose {
				fmt.Printf("📁 Indexing: %s\n", path)
			} else {
				fmt.Printf("Indexing: %s\n", path)
			}

			hash, err := s.calculateFileHash(path)
			if err != nil {
				// Log the error but continue scanning other files
				if utils.IsPermissionError(err) {
					stats.PermissionErrors++
					if options.Verbose {
						fmt.Printf("⚠️  Permission denied for file %s, skipping\n", path)
					}
				} else {
					fmt.Fprintf(os.Stderr, "Could not hash file %s: %v\n", path, err)
				}
				return nil
			}

			file := &db.File{
				Path:       path,
				Name:       info.Name(),
				Size:       info.Size(),
				ModifiedAt: info.ModTime(),
				FileHash:   hash,
			}

			if err := s.database.UpsertFile(file); err != nil {
				fmt.Fprintf(os.Stderr, "Could not index file %s: %v\n", path, err)
			} else {
				stats.FilesIndexed++
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %w", rootDir, err)
	}

	return stats, nil
}

// calculateFileHash calculates the SHA-256 hash of a file
func (s *FileScanner) calculateFileHash(filePath string) (string, error) {
	file, err := s.fs.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
