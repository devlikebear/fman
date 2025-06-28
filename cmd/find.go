/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/spf13/cobra"
)

var (
	findSize     string
	findModified string
	findType     string
	findDir      string
)

// findCmd represents the find command
var findCmd = &cobra.Command{
	Use:   "find [name-pattern]",
	Short: "Finds indexed files with advanced search criteria.",
	Long: `Searches the fman database for files using various criteria.
You can search by name pattern, file size, modification date, file type, and directory.

Examples:
  fman find report                    # Find files with 'report' in name
  fman find --size +100M              # Find files larger than 100MB
  fman find --size 1M-10M             # Find files between 1MB and 10MB
  fman find --modified -30d           # Find files modified in last 30 days
  fman find --modified +7d            # Find files older than 7 days
  fman find --type image              # Find image files (.jpg, .png, .gif, etc.)
  fman find --type .pdf               # Find PDF files
  fman find --dir /Users/john/Documents # Find files in specific directory
  fman find report --size +1M --type .pdf # Combined search

Size format: [+/-]<number>[K|M|G] (e.g., +100M, -1G, 500K-2M)
Date format: [+/-]<number>[d|w|m|y] or YYYY-MM-DD (e.g., -30d, +1w, 2024-01-01)
Type format: image|video|audio|document|archive or file extension (e.g., .pdf, .jpg)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create a new Database instance for each command execution
		dbInstance := db.NewDatabase(nil)
		return runFind(cmd, args, dbInstance)
	},
}

func runFind(cmd *cobra.Command, args []string, database db.DBInterface) error {
	// Initialize DB
	if err := database.InitDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	// Build search criteria
	criteria := db.SearchCriteria{}

	// Name pattern
	if len(args) > 0 {
		criteria.NamePattern = args[0]
	}

	// Parse size filter
	if findSize != "" {
		minSize, maxSize, err := parseSize(findSize)
		if err != nil {
			return fmt.Errorf("invalid size format: %w", err)
		}
		criteria.MinSize = minSize
		criteria.MaxSize = maxSize
	}

	// Parse modified date filter
	if findModified != "" {
		after, before, err := parseModified(findModified)
		if err != nil {
			return fmt.Errorf("invalid modified date format: %w", err)
		}
		criteria.ModifiedAfter = after
		criteria.ModifiedBefore = before
	}

	// Parse file type filter
	if findType != "" {
		types, err := parseFileType(findType)
		if err != nil {
			return fmt.Errorf("invalid file type: %w", err)
		}
		criteria.FileTypes = types
	}

	// Directory filter
	if findDir != "" {
		absDir, err := filepath.Abs(findDir)
		if err != nil {
			return fmt.Errorf("invalid directory path: %w", err)
		}
		criteria.SearchDir = absDir
	}

	// Search files
	files, err := database.FindFilesByAdvancedCriteria(criteria)
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No files found matching the specified criteria.")
		return nil
	}

	// Print results in a table format
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

	fmt.Printf("\nFound %d file(s)\n", len(files))
	return nil
}

// parseSize parses size filter strings like "+100M", "-1G", "500K-2M"
func parseSize(sizeStr string) (*int64, *int64, error) {
	// Handle range format (e.g., "1M-10M")
	if strings.Contains(sizeStr, "-") && !strings.HasPrefix(sizeStr, "-") {
		parts := strings.Split(sizeStr, "-")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid range format")
		}
		min, err := parseSingleSize(parts[0])
		if err != nil {
			return nil, nil, err
		}
		max, err := parseSingleSize(parts[1])
		if err != nil {
			return nil, nil, err
		}
		return &min, &max, nil
	}

	// Handle single size with +/- prefix
	if strings.HasPrefix(sizeStr, "+") {
		size, err := parseSingleSize(sizeStr[1:])
		if err != nil {
			return nil, nil, err
		}
		return &size, nil, nil
	}

	if strings.HasPrefix(sizeStr, "-") {
		size, err := parseSingleSize(sizeStr[1:])
		if err != nil {
			return nil, nil, err
		}
		zero := int64(0)
		return &zero, &size, nil
	}

	// Handle exact size
	size, err := parseSingleSize(sizeStr)
	if err != nil {
		return nil, nil, err
	}
	return &size, &size, nil
}

// parseSingleSize parses a single size string like "100M", "1G", "500K"
func parseSingleSize(sizeStr string) (int64, error) {
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)(K|M|G|B)?$`)
	matches := re.FindStringSubmatch(strings.ToUpper(sizeStr))
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	num, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	unit := matches[2]
	switch unit {
	case "K":
		return int64(num * 1024), nil
	case "M":
		return int64(num * 1024 * 1024), nil
	case "G":
		return int64(num * 1024 * 1024 * 1024), nil
	case "B", "":
		return int64(num), nil
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}
}

// parseModified parses modified date filter strings like "-30d", "+1w", "2024-01-01"
func parseModified(modStr string) (*time.Time, *time.Time, error) {
	now := time.Now()

	// Handle relative dates with +/- prefix
	if strings.HasPrefix(modStr, "+") || strings.HasPrefix(modStr, "-") {
		isOlder := strings.HasPrefix(modStr, "+")
		durStr := modStr[1:]

		duration, err := parseRelativeDuration(durStr)
		if err != nil {
			return nil, nil, err
		}

		if isOlder {
			// Files older than duration
			cutoff := now.Add(-duration)
			return nil, &cutoff, nil
		} else {
			// Files newer than duration
			cutoff := now.Add(-duration)
			return &cutoff, nil, nil
		}
	}

	// Handle absolute date (YYYY-MM-DD)
	date, err := time.Parse("2006-01-02", modStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid date format, use YYYY-MM-DD or relative format like -30d")
	}

	// Return files from that specific day
	start := date
	end := date.Add(24 * time.Hour)
	return &start, &end, nil
}

// parseRelativeDuration parses duration strings like "30d", "1w", "2m", "1y"
func parseRelativeDuration(durStr string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)(d|w|m|y)$`)
	matches := re.FindStringSubmatch(strings.ToLower(durStr))
	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid duration format: %s", durStr)
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	unit := matches[2]
	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "w":
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(num) * 30 * 24 * time.Hour, nil
	case "y":
		return time.Duration(num) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}

// parseFileType parses file type filter strings
func parseFileType(typeStr string) ([]string, error) {
	typeStr = strings.ToLower(typeStr)

	// Handle predefined types
	switch typeStr {
	case "image":
		return []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg"}, nil
	case "video":
		return []string{".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v"}, nil
	case "audio":
		return []string{".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a"}, nil
	case "document":
		return []string{".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".pages"}, nil
	case "archive":
		return []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz"}, nil
	default:
		// Handle specific extension
		if strings.HasPrefix(typeStr, ".") {
			return []string{typeStr}, nil
		}
		// Try to add dot prefix
		return []string{"." + typeStr}, nil
	}
}

// formatSize formats file size in human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(findCmd)

	// Add flags for advanced search
	findCmd.Flags().StringVar(&findSize, "size", "", "Filter by file size (e.g., +100M, -1G, 500K-2M)")
	findCmd.Flags().StringVar(&findModified, "modified", "", "Filter by modification date (e.g., -30d, +1w, 2024-01-01)")
	findCmd.Flags().StringVar(&findType, "type", "", "Filter by file type (image, video, audio, document, archive, .pdf, .jpg)")
	findCmd.Flags().StringVar(&findDir, "dir", "", "Search within specific directory")
}
