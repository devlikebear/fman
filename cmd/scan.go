/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// ScanStats holds statistics about the scanning process
type ScanStats struct {
	FilesIndexed       int
	DirectoriesSkipped int
	PermissionErrors   int
	SkippedPaths       []string
}

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan <directory>",
	Short: "Scans a directory and indexes file metadata into the database.",
	Long: `Recursively scans the specified directory, calculates metadata and a content hash
for each file, and stores this information in the fman database. This index is later
used by other commands like 'find' and 'organize'.

The scanner automatically skips system directories and handles permission errors gracefully.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		forceSudo, _ := cmd.Flags().GetBool("force-sudo")

		// If force-sudo is requested and we're not already running as root
		if forceSudo && !isRunningAsRoot() {
			return runWithSudo(cmd, args)
		}

		// Create a new Database instance for each command execution
		// This ensures that each command gets its own DB connection.
		dbInstance := db.NewDatabase(nil)
		return runScan(cmd, args, fileSystem, dbInstance)
	},
}

// isRunningAsRoot checks if the current process is running with root privileges
func isRunningAsRoot() bool {
	if runtime.GOOS == "windows" {
		// On Windows, check if running as administrator
		// This is a simplified check - in production you might want more robust detection
		return false // For now, we'll skip sudo functionality on Windows
	}
	return os.Geteuid() == 0
}

// runWithSudo re-executes the current command with sudo privileges
func runWithSudo(cmd *cobra.Command, args []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("sudo functionality is not supported on Windows")
	}

	fmt.Println("🔐 Requesting elevated privileges...")
	fmt.Println("⚠️  WARNING: You are about to run fman with sudo privileges.")
	fmt.Print("   Continue? (y/N): ")

	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Operation cancelled.")
		return nil
	}

	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build the sudo command
	sudoArgs := []string{executable, "scan"}
	sudoArgs = append(sudoArgs, args...)

	// Add other flags except --force-sudo to avoid infinite recursion
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		sudoArgs = append(sudoArgs, "--verbose")
	}

	// Execute with sudo
	sudoCmd := exec.Command("sudo", sudoArgs...)
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	sudoCmd.Stdin = os.Stdin

	return sudoCmd.Run()
}

// getSkipPatterns returns a list of directory patterns to skip based on the OS
func getSkipPatterns() []string {
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

// shouldSkipPath checks if a path should be skipped based on patterns
func shouldSkipPath(path string, skipPatterns []string) bool {
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

// isPermissionError checks if an error is a permission error
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for syscall.Errno permission errors
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.EACCES || errno == syscall.EPERM
	}

	// Check for string-based permission errors
	errStr := err.Error()
	return strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "operation not permitted") ||
		strings.Contains(errStr, "access is denied")
}

func runScan(cmd *cobra.Command, args []string, fs afero.Fs, database db.DBInterface) error {
	// Initialize DB
	if err := database.InitDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close() // Ensure DB connection is closed

	rootDir := args[0]
	skipPatterns := getSkipPatterns()
	stats := &ScanStats{}

	// Get flags
	forceSudo, _ := cmd.Flags().GetBool("force-sudo")
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Printf("Starting scan of directory: %s\n", rootDir)
	if verbose {
		fmt.Printf("Skip patterns: %v\n", skipPatterns)
	}

	if isRunningAsRoot() {
		fmt.Printf("🔐 Running with elevated privileges\n")
	}

	err := afero.Walk(fs, rootDir, func(path string, info os.FileInfo, err error) error {
		// Handle permission errors gracefully
		if err != nil {
			if isPermissionError(err) {
				stats.PermissionErrors++
				stats.SkippedPaths = append(stats.SkippedPaths, path)
				if verbose {
					fmt.Printf("⚠️  Permission denied, skipping: %s\n", path)
				}
				return nil // Continue walking, skip this path
			}
			return err // Return other errors
		}

		// Skip special directories (even when running as root, unless verbose)
		if info.IsDir() && shouldSkipPath(path, skipPatterns) {
			// When running as root, still skip system directories unless explicitly verbose
			if !isRunningAsRoot() || !verbose {
				stats.DirectoriesSkipped++
				stats.SkippedPaths = append(stats.SkippedPaths, path)
				if verbose {
					fmt.Printf("⏭️  Skipping special directory: %s\n", path)
				}
				return filepath.SkipDir
			}
		}

		// Process files
		if !info.IsDir() {
			if verbose {
				fmt.Printf("📁 Indexing: %s\n", path)
			} else {
				fmt.Printf("Indexing: %s\n", path)
			}

			hash, err := calculateFileHash(fs, path)
			if err != nil {
				// Log the error but continue scanning other files
				if isPermissionError(err) {
					stats.PermissionErrors++
					if verbose {
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

			if err := database.UpsertFile(file); err != nil {
				fmt.Fprintf(os.Stderr, "Could not index file %s: %v\n", path, err)
			} else {
				stats.FilesIndexed++
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking the path %s: %w", rootDir, err)
	}

	// Print scan statistics
	fmt.Println("\n📊 Scan Statistics:")
	fmt.Printf("✅ Files indexed: %d\n", stats.FilesIndexed)
	fmt.Printf("⏭️  Directories skipped: %d\n", stats.DirectoriesSkipped)
	fmt.Printf("⚠️  Permission errors: %d\n", stats.PermissionErrors)

	if len(stats.SkippedPaths) > 0 && verbose {
		fmt.Println("\n📋 Skipped paths:")
		for _, path := range stats.SkippedPaths {
			fmt.Printf("  - %s\n", path)
		}
	}

	if stats.PermissionErrors > 0 {
		fmt.Printf("\n💡 Tip: %d paths were skipped due to permission errors.\n", stats.PermissionErrors)
		if !forceSudo && !isRunningAsRoot() {
			fmt.Println("   Use --force-sudo flag if you need to scan protected directories (use with caution).")
		}
	}

	fmt.Println("✅ Scan complete.")
	return nil
}

func calculateFileHash(fs afero.Fs, filePath string) (string, error) {
	file, err := fs.Open(filePath)
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

func init() {
	rootCmd.AddCommand(scanCmd)

	// Add flags for advanced options
	scanCmd.Flags().Bool("force-sudo", false, "Force scanning with elevated privileges (use with caution)")
	scanCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output showing skipped paths")
}
