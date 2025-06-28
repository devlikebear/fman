/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan <directory>",
	Short: "Scans a directory and indexes file metadata into the database.",
	Long: `Recursively scans the specified directory, calculates metadata and a content hash
for each file, and stores this information in the fman database. This index is later
used by other commands like 'find' and 'organize'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a new Database instance for each command execution
		// This ensures that each command gets its own DB connection.
		dbInstance := db.NewDatabase(nil)
		return runScan(cmd, args, fileSystem, dbInstance)
	},
}

func runScan(cmd *cobra.Command, args []string, fs afero.Fs, database db.DBInterface) error {
	// Initialize DB
	if err := database.InitDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close() // Ensure DB connection is closed

	rootDir := args[0]
	fmt.Printf("Starting scan of directory: %s\n", rootDir)

	err := afero.Walk(fs, rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fmt.Printf("Indexing: %s\n", path)

			hash, err := calculateFileHash(fs, path)
			if err != nil {
				// Log the error but continue scanning other files
				fmt.Fprintf(os.Stderr, "Could not hash file %s: %v\n", path, err)
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
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking the path %s: %w", rootDir, err)
	}

	fmt.Println("Scan complete.")
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
}
