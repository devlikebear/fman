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
	"runtime"
	"time"

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
	Verbose       bool
	ForceSudo     bool
	ThrottleDelay time.Duration // 파일 간 처리 지연시간
	MaxFileSize   int64         // 해시 계산 최대 파일 크기 (바이트)
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
			// CPU 사용량 제어를 위한 주기적인 지연
			if options.ThrottleDelay > 0 && stats.FilesIndexed%100 == 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(options.ThrottleDelay):
				}
			}

			// 메모리 사용량 모니터링 및 가비지 컬렉션
			if stats.FilesIndexed%1000 == 0 {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.Alloc > 100*1024*1024 { // 100MB 초과시 GC 강제 실행
					runtime.GC()
				}
			}

			if options.Verbose {
				fmt.Printf("📁 Indexing: %s\n", path)
			} else {
				fmt.Printf("Indexing: %s\n", path)
			}

			// 파일 크기 확인 후 해시 계산
			var hash string
			if options.MaxFileSize > 0 && info.Size() > options.MaxFileSize {
				// 큰 파일은 해시 계산 건너뛰기
				hash = "large_file_skipped"
				if options.Verbose {
					fmt.Printf("⏭️  File too large for hashing: %s (%d bytes)\n", path, info.Size())
				}
			} else {
				var err error
				hash, err = s.calculateFileHash(path)
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

// calculateFileHash calculates the SHA-256 hash of a file with optimized reading
func (s *FileScanner) calculateFileHash(filePath string) (string, error) {
	file, err := s.fs.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	// CPU 부하를 줄이기 위해 청크 단위로 읽기 (32KB 버퍼)
	buffer := make([]byte, 32*1024)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// 큰 파일의 경우 CPU 사용량을 줄이기 위해 잠시 대기
		runtime.Gosched() // 다른 고루틴에게 실행 기회 제공
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
