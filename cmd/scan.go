/*
Copyright © 2025 changheonshin

*/
package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan <directory>",
	Short: "Scans a directory and indexes file metadata into the database.",
	Long: `Recursively scans the specified directory, calculates metadata and a content hash
for each file, and stores this information in the fman database. This index is later
used by other commands like 'find' and 'organize'.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := db.InitDB(); err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		rootDir := args[0]
		fmt.Printf("Starting scan of directory: %s\n", rootDir)

		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fmt.Printf("Indexing: %s\n", path)

				hash, err := calculateFileHash(path)
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

				if err := db.UpsertFile(file); err != nil {
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
	},
}

func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
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
