/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/devlikebear/fman/internal/db"
	"github.com/devlikebear/fman/internal/scanner"
	"github.com/devlikebear/fman/internal/utils"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

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
		if forceSudo && !utils.IsRunningAsRoot() {
			return utils.RunWithSudo(cmd, args)
		}

		// Create a new Database instance for each command execution
		// This ensures that each command gets its own DB connection.
		dbInstance := db.NewDatabase(nil)
		return runScan(cmd, args, fileSystem, dbInstance)
	},
}

func runScan(cmd *cobra.Command, args []string, fs afero.Fs, database db.DBInterface) error {
	// Get flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	forceSudo, _ := cmd.Flags().GetBool("force-sudo")

	// Create scanner options
	options := &scanner.ScanOptions{
		Verbose:   verbose,
		ForceSudo: forceSudo,
	}

	// Create scanner instance
	fileScanner := scanner.NewFileScanner(fs, database)

	// Create context for cancellation support
	ctx := context.Background()

	// Perform the scan
	stats, err := fileScanner.ScanDirectory(ctx, args[0], options)
	if err != nil {
		return err
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
		if !forceSudo && !utils.IsRunningAsRoot() {
			fmt.Println("   Use --force-sudo flag if you need to scan protected directories (use with caution).")
		}
	}

	fmt.Println("✅ Scan complete.")
	return nil
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Add flags for advanced options
	scanCmd.Flags().Bool("force-sudo", false, "Force scanning with elevated privileges (use with caution)")
	scanCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output showing skipped paths")
}
