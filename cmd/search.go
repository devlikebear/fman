/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/devlikebear/fman/internal/ai"
	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [natural-language-query]",
	Short: "Search files using natural language queries powered by AI",
	Long: `Search files using natural language queries that are converted to structured search criteria by AI.

This command uses your configured AI provider (Gemini or Ollama) to interpret natural language
and convert it into precise search parameters.

Examples:
  fman search "find large images modified last week"
  fman search "show me PDF documents from Downloads folder"
  fman search "files with report in name bigger than 10MB"
  fman search "videos created yesterday"
  fman search "small text files in my home directory"
  fman search "huge files taking up space"

The AI will interpret your query and search for files based on:
- File names and patterns
- File sizes (large, small, huge, specific sizes)
- Modification dates (last week, yesterday, specific dates)
- File types (images, videos, documents, specific extensions)
- Directory locations

Make sure your AI provider is properly configured in ~/.fman/config.yml`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a new Database instance for each command execution
		dbInstance := db.NewDatabase(nil)
		return runSearch(cmd, args, dbInstance)
	},
}

func runSearch(cmd *cobra.Command, args []string, database db.DBInterface) error {
	// Initialize DB
	if err := database.InitDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Get the natural language query
	query := args[0]

	// Get AI provider from config
	provider, err := getAIProvider()
	if err != nil {
		return fmt.Errorf("failed to get AI provider: %w", err)
	}

	// Convert natural language query to search criteria using AI
	fmt.Printf("🤖 Interpreting query: \"%s\"\n", query)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	criteria, err := provider.ParseSearchQuery(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to parse search query: %w", err)
	}

	// Show what the AI understood
	fmt.Println("🔍 AI interpreted your query as:")
	printSearchCriteria(criteria)

	// Search files using the criteria
	files, err := database.FindFilesByAdvancedCriteria(*criteria)
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("\n❌ No files found matching your query.")
		fmt.Println("💡 Try rephrasing your query or check if the files are indexed with 'fman scan <directory>'")
		return nil
	}

	// Print results in a table format
	fmt.Printf("\n📁 Found %d file(s):\n\n", len(files))
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Name\tSize\tModified\tPath")
	fmt.Fprintln(w, "----\t----\t--------\t----")
	for _, file := range files {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			file.Name,
			formatSize(file.Size),
			file.ModifiedAt.Format("2006-01-02 15:04"),
			file.Path,
		)
	}
	w.Flush()

	return nil
}

// getAIProvider creates an AI provider based on the configuration
func getAIProvider() (ai.AIProvider, error) {
	aiProvider := viper.GetString("ai_provider")
	if aiProvider == "" {
		return nil, fmt.Errorf("ai_provider is not set in the configuration")
	}

	switch aiProvider {
	case "gemini":
		return ai.NewGeminiProvider(), nil
	case "ollama":
		return ai.NewOllamaProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", aiProvider)
	}
}

// printSearchCriteria shows what the AI understood from the query
func printSearchCriteria(criteria *db.SearchCriteria) {
	if criteria.NamePattern != "" {
		fmt.Printf("  • Name contains: %s\n", criteria.NamePattern)
	}
	if criteria.MinSize != nil {
		fmt.Printf("  • Minimum size: %s\n", formatSize(*criteria.MinSize))
	}
	if criteria.MaxSize != nil {
		fmt.Printf("  • Maximum size: %s\n", formatSize(*criteria.MaxSize))
	}
	if criteria.ModifiedAfter != nil {
		fmt.Printf("  • Modified after: %s\n", criteria.ModifiedAfter.Format("2006-01-02 15:04"))
	}
	if criteria.ModifiedBefore != nil {
		fmt.Printf("  • Modified before: %s\n", criteria.ModifiedBefore.Format("2006-01-02 15:04"))
	}
	if criteria.SearchDir != "" {
		fmt.Printf("  • Search directory: %s\n", criteria.SearchDir)
	}
	if len(criteria.FileTypes) > 0 {
		fmt.Printf("  • File types: %v\n", criteria.FileTypes)
	}
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
