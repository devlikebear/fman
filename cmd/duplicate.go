/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/cobra"
)

// duplicateCmd represents the duplicate command
var duplicateCmd = &cobra.Command{
	Use:   "duplicate [directory]",
	Short: "Find and manage duplicate files",
	Long: `Find duplicate files by comparing file hashes.
This command scans the database for files with identical content (same hash)
and provides options to review and remove duplicates.

If no directory is specified, searches all indexed files.
If a directory is specified, only searches within that directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDuplicate,
}

var (
	duplicateRemove      bool
	duplicateInteractive bool
	duplicateMinSize     int64
)

func runDuplicate(cmd *cobra.Command, args []string) error {
	// Initialize database
	database := db.NewDatabase(nil)
	if err := database.InitDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Get search directory if specified
	var searchDir string
	if len(args) > 0 {
		var err error
		searchDir, err = filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("failed to resolve directory path: %w", err)
		}
	}

	// Find duplicate files
	duplicates, err := findDuplicateFiles(database, searchDir, duplicateMinSize)
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(duplicates) == 0 {
		fmt.Println("No duplicate files found.")
		return nil
	}

	// Display results
	totalDuplicates, totalSize := displayDuplicates(duplicates)

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("Found %d duplicate file groups\n", len(duplicates))
	fmt.Printf("Total duplicate files: %d\n", totalDuplicates)
	fmt.Printf("Total wasted space: %.2f MB\n", float64(totalSize)/(1024*1024))

	// Handle removal if requested
	if duplicateRemove {
		return handleDuplicateRemoval(duplicates, duplicateInteractive)
	}

	fmt.Println("\nUse --remove flag to delete duplicates, or --interactive for selective removal.")
	return nil
}

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	Hash  string
	Files []db.File
	Size  int64
}

func findDuplicateFiles(database db.DBInterface, searchDir string, minSize int64) ([]DuplicateGroup, error) {
	// Get all files with hashes
	files, err := database.FindFilesWithHashes(searchDir, minSize)
	if err != nil {
		return nil, err
	}

	// Group files by hash
	hashGroups := make(map[string][]db.File)
	for _, file := range files {
		if file.FileHash != "" {
			hashGroups[file.FileHash] = append(hashGroups[file.FileHash], file)
		}
	}

	// Find groups with more than one file (duplicates)
	var duplicates []DuplicateGroup
	for hash, files := range hashGroups {
		if len(files) > 1 {
			// Sort files by path for consistent ordering
			sort.Slice(files, func(i, j int) bool {
				return files[i].Path < files[j].Path
			})

			duplicates = append(duplicates, DuplicateGroup{
				Hash:  hash,
				Files: files,
				Size:  files[0].Size, // All files in group have same size
			})
		}
	}

	// Sort duplicate groups by size (largest first)
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Size > duplicates[j].Size
	})

	return duplicates, nil
}

func displayDuplicates(duplicates []DuplicateGroup) (int, int64) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)

	totalDuplicates := 0
	totalSize := int64(0)

	for i, group := range duplicates {
		fmt.Printf("\n🔍 Duplicate Group %d (Hash: %s):\n", i+1, group.Hash[:12]+"...")
		fmt.Printf("File Size: %.2f MB\n", float64(group.Size)/(1024*1024))
		fmt.Printf("Duplicate Count: %d files\n", len(group.Files))
		fmt.Printf("Wasted Space: %.2f MB\n\n", float64(group.Size*(int64(len(group.Files))-1))/(1024*1024))

		fmt.Fprintln(w, "Path\tModified\tDirectory")
		fmt.Fprintln(w, "----\t--------\t---------")

		for _, file := range group.Files {
			dir := filepath.Dir(file.Path)
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				file.Path,
				file.ModifiedAt.Format("2006-01-02 15:04:05"),
				dir,
			)
		}
		w.Flush()

		totalDuplicates += len(group.Files)
		totalSize += group.Size * (int64(len(group.Files)) - 1) // Wasted space
	}

	return totalDuplicates, totalSize
}

func handleDuplicateRemoval(duplicates []DuplicateGroup, interactive bool) error {
	fmt.Println("\n🗑️  Removing duplicate files...")

	removedCount := 0
	savedSpace := int64(0)

	for i, group := range duplicates {
		fmt.Printf("\nProcessing group %d/%d...\n", i+1, len(duplicates))

		if interactive {
			fmt.Printf("Files in this group:\n")
			for j, file := range group.Files {
				fmt.Printf("  %d. %s (modified: %s)\n", j+1, file.Path,
					file.ModifiedAt.Format("2006-01-02 15:04:05"))
			}

			fmt.Print("Keep which file? (1-" + fmt.Sprintf("%d", len(group.Files)) + ", or 's' to skip): ")
			var choice string
			fmt.Scanln(&choice)

			if choice == "s" || choice == "S" {
				fmt.Println("Skipped.")
				continue
			}

			// Parse choice and remove others
			var keepIndex int
			if _, err := fmt.Sscanf(choice, "%d", &keepIndex); err != nil || keepIndex < 1 || keepIndex > len(group.Files) {
				fmt.Println("Invalid choice, skipping group.")
				continue
			}
			keepIndex-- // Convert to 0-based index

			// Remove all files except the chosen one
			for j, file := range group.Files {
				if j != keepIndex {
					if err := os.Remove(file.Path); err != nil {
						fmt.Printf("Failed to remove %s: %v\n", file.Path, err)
					} else {
						fmt.Printf("Removed: %s\n", file.Path)
						removedCount++
						savedSpace += group.Size
					}
				}
			}
		} else {
			// Non-interactive: keep the first file (usually oldest), remove others
			fmt.Printf("Keeping: %s\n", group.Files[0].Path)
			for j := 1; j < len(group.Files); j++ {
				file := group.Files[j]
				if err := os.Remove(file.Path); err != nil {
					fmt.Printf("Failed to remove %s: %v\n", file.Path, err)
				} else {
					fmt.Printf("Removed: %s\n", file.Path)
					removedCount++
					savedSpace += group.Size
				}
			}
		}
	}

	fmt.Printf("\n✅ Cleanup complete!\n")
	fmt.Printf("Files removed: %d\n", removedCount)
	fmt.Printf("Space saved: %.2f MB\n", float64(savedSpace)/(1024*1024))

	return nil
}

func init() {
	rootCmd.AddCommand(duplicateCmd)

	duplicateCmd.Flags().BoolVar(&duplicateRemove, "remove", false, "Remove duplicate files automatically")
	duplicateCmd.Flags().BoolVar(&duplicateInteractive, "interactive", false, "Interactive mode for selective removal")
	duplicateCmd.Flags().Int64Var(&duplicateMinSize, "min-size", 1024, "Minimum file size in bytes to consider for duplicates")
}
