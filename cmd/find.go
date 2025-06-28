/*
Copyright © 2025 changheonshin

*/
package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/cobra"
)

// findCmd represents the find command
var findCmd = &cobra.Command{
	Use:   "find <name-pattern>",
	Short: "Finds indexed files by a name pattern.",
	Long: `Searches the fman database for files matching the given name pattern.
The pattern is case-insensitive and can be a partial name.
For example: 'fman find report' will find 'report.pdf', 'annual_report.docx', etc.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a new Database instance for each command execution
		// This ensures that each command gets its own DB connection.
		dbInstance := db.NewDatabase(nil)
		return runFind(cmd, args, dbInstance)
	},
}

func runFind(cmd *cobra.Command, args []string, database db.DBInterface) error {
	// Initialize DB
	if err := database.InitDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close() // Ensure DB connection is closed

	pattern := args[0]
	files, err := database.FindFilesByName(pattern)
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		fmt.Printf("No files found matching pattern: %s\n", pattern)
		return nil
	}

	// Print results in a table format
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Name\tSize (bytes)\tModified\tPath")
	fmt.Fprintln(w, "----\t------------\t--------\t----")
	for _, file := range files {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			file.Name,
			file.Size,
			file.ModifiedAt.Format("2006-01-02 15:04:05"),
			file.Path,
		)
	}
	w.Flush()

	return nil
}

func init() {
	rootCmd.AddCommand(findCmd)
}