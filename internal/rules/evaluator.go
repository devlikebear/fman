package rules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/devlikebear/fman/internal/db"
)

// Evaluator evaluates rules against files
type Evaluator struct {
	verbose bool
}

// NewEvaluator creates a new rule evaluator
func NewEvaluator(verbose bool) *Evaluator {
	return &Evaluator{
		verbose: verbose,
	}
}

// EvaluateRule evaluates if a file matches all conditions of a rule
func (e *Evaluator) EvaluateRule(rule Rule, file db.File, baseDir string) (bool, error) {
	if !rule.Enabled {
		return false, nil
	}

	// All conditions must be satisfied (AND logic)
	for _, condition := range rule.Conditions {
		matches, err := e.evaluateCondition(condition, file, baseDir)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate condition: %w", err)
		}
		if !matches {
			if e.verbose {
				fmt.Printf("File %s does not match condition: %s %s %s\n",
					file.Path, condition.Type, condition.Operator, condition.Value)
			}
			return false, nil
		}
	}

	return true, nil
}

// evaluateCondition evaluates a single condition against a file
func (e *Evaluator) evaluateCondition(condition Condition, file db.File, baseDir string) (bool, error) {
	switch condition.Type {
	case ConditionNamePattern:
		return e.evaluateNamePattern(condition, file)
	case ConditionExtension:
		return e.evaluateExtension(condition, file)
	case ConditionSize:
		return e.evaluateSize(condition, file)
	case ConditionAge:
		return e.evaluateAge(condition, file)
	case ConditionModified:
		return e.evaluateModified(condition, file)
	case ConditionPath:
		return e.evaluatePath(condition, file, baseDir)
	case ConditionFileType:
		return e.evaluateFileType(condition, file)
	case ConditionMimeType:
		return e.evaluateMimeType(condition, file)
	default:
		return false, fmt.Errorf("unsupported condition type: %s", condition.Type)
	}
}

// evaluateNamePattern evaluates name pattern conditions
func (e *Evaluator) evaluateNamePattern(condition Condition, file db.File) (bool, error) {
	filename := filepath.Base(file.Path)

	switch condition.Operator {
	case OpContains, "":
		return strings.Contains(filename, condition.Value), nil
	case OpEqual:
		return filename == condition.Value, nil
	case OpNotEqual:
		return filename != condition.Value, nil
	case OpStartsWith:
		return strings.HasPrefix(filename, condition.Value), nil
	case OpEndsWith:
		return strings.HasSuffix(filename, condition.Value), nil
	case OpMatches:
		matched, err := regexp.MatchString(condition.Value, filename)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern '%s': %w", condition.Value, err)
		}
		return matched, nil
	default:
		return false, fmt.Errorf("unsupported operator '%s' for name_pattern", condition.Operator)
	}
}

// evaluateExtension evaluates file extension conditions
func (e *Evaluator) evaluateExtension(condition Condition, file db.File) (bool, error) {
	ext := strings.ToLower(filepath.Ext(file.Path))
	expectedExt := strings.ToLower(condition.Value)

	// Add dot if not present
	if !strings.HasPrefix(expectedExt, ".") {
		expectedExt = "." + expectedExt
	}

	switch condition.Operator {
	case OpEqual, "":
		return ext == expectedExt, nil
	case OpNotEqual:
		return ext != expectedExt, nil
	default:
		return false, fmt.Errorf("unsupported operator '%s' for extension", condition.Operator)
	}
}

// evaluateSize evaluates file size conditions
func (e *Evaluator) evaluateSize(condition Condition, file db.File) (bool, error) {
	targetSize, err := e.parseSize(condition.Value)
	if err != nil {
		return false, fmt.Errorf("invalid size value '%s': %w", condition.Value, err)
	}

	switch condition.Operator {
	case OpGreaterThan, "":
		return file.Size > targetSize, nil
	case OpLessThan:
		return file.Size < targetSize, nil
	case OpGreaterThanOrEqual:
		return file.Size >= targetSize, nil
	case OpLessThanOrEqual:
		return file.Size <= targetSize, nil
	case OpEqual:
		return file.Size == targetSize, nil
	case OpNotEqual:
		return file.Size != targetSize, nil
	default:
		return false, fmt.Errorf("unsupported operator '%s' for size", condition.Operator)
	}
}

// evaluateAge evaluates file age conditions
func (e *Evaluator) evaluateAge(condition Condition, file db.File) (bool, error) {
	targetAge, err := e.parseDuration(condition.Value)
	if err != nil {
		return false, fmt.Errorf("invalid age value '%s': %w", condition.Value, err)
	}

	fileAge := time.Since(file.ModifiedAt)

	switch condition.Operator {
	case OpGreaterThan, "":
		return fileAge > targetAge, nil
	case OpLessThan:
		return fileAge < targetAge, nil
	case OpGreaterThanOrEqual:
		return fileAge >= targetAge, nil
	case OpLessThanOrEqual:
		return fileAge <= targetAge, nil
	case OpEqual:
		return fileAge == targetAge, nil
	case OpNotEqual:
		return fileAge != targetAge, nil
	default:
		return false, fmt.Errorf("unsupported operator '%s' for age", condition.Operator)
	}
}

// evaluateModified evaluates file modification date conditions
func (e *Evaluator) evaluateModified(condition Condition, file db.File) (bool, error) {
	targetTime, err := e.parseTime(condition.Value)
	if err != nil {
		return false, fmt.Errorf("invalid modified value '%s': %w", condition.Value, err)
	}

	switch condition.Operator {
	case OpGreaterThan, "":
		return file.ModifiedAt.After(targetTime), nil
	case OpLessThan:
		return file.ModifiedAt.Before(targetTime), nil
	case OpGreaterThanOrEqual:
		return file.ModifiedAt.After(targetTime) || file.ModifiedAt.Equal(targetTime), nil
	case OpLessThanOrEqual:
		return file.ModifiedAt.Before(targetTime) || file.ModifiedAt.Equal(targetTime), nil
	case OpEqual:
		return file.ModifiedAt.Equal(targetTime), nil
	case OpNotEqual:
		return !file.ModifiedAt.Equal(targetTime), nil
	default:
		return false, fmt.Errorf("unsupported operator '%s' for modified", condition.Operator)
	}
}

// evaluatePath evaluates file path conditions
func (e *Evaluator) evaluatePath(condition Condition, file db.File, baseDir string) (bool, error) {
	path := file.Path
	if baseDir != "" {
		// Make path relative to base directory if possible
		if strings.HasPrefix(path, baseDir) {
			path = strings.TrimPrefix(path, baseDir)
			path = strings.TrimPrefix(path, "/")
		}
	}

	switch condition.Operator {
	case OpContains, "":
		return strings.Contains(path, condition.Value), nil
	case OpEqual:
		return path == condition.Value, nil
	case OpNotEqual:
		return path != condition.Value, nil
	case OpStartsWith:
		return strings.HasPrefix(path, condition.Value), nil
	case OpEndsWith:
		return strings.HasSuffix(path, condition.Value), nil
	case OpMatches:
		matched, err := regexp.MatchString(condition.Value, path)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern '%s': %w", condition.Value, err)
		}
		return matched, nil
	default:
		return false, fmt.Errorf("unsupported operator '%s' for path", condition.Operator)
	}
}

// evaluateFileType evaluates file type conditions
func (e *Evaluator) evaluateFileType(condition Condition, file db.File) (bool, error) {
	ext := strings.ToLower(filepath.Ext(file.Path))

	// Define file type mappings
	fileTypes := map[string][]string{
		"image":    {".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg"},
		"video":    {".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v"},
		"audio":    {".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a"},
		"document": {".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".pages"},
		"archive":  {".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz"},
		"code":     {".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".h"},
	}

	targetType := strings.ToLower(condition.Value)

	// Check if it's a predefined type
	if extensions, exists := fileTypes[targetType]; exists {
		for _, validExt := range extensions {
			if ext == validExt {
				switch condition.Operator {
				case OpEqual, "":
					return true, nil
				case OpNotEqual:
					return false, nil
				}
			}
		}
		switch condition.Operator {
		case OpEqual, "":
			return false, nil
		case OpNotEqual:
			return true, nil
		}
	}

	// Check if it's a specific extension
	if strings.HasPrefix(targetType, ".") {
		switch condition.Operator {
		case OpEqual, "":
			return ext == targetType, nil
		case OpNotEqual:
			return ext != targetType, nil
		}
	}

	return false, fmt.Errorf("unknown file type: %s", condition.Value)
}

// evaluateMimeType evaluates MIME type conditions (placeholder for future implementation)
func (e *Evaluator) evaluateMimeType(condition Condition, file db.File) (bool, error) {
	// This would require file content analysis
	// For now, return false as it's not implemented
	return false, fmt.Errorf("mime_type condition not yet implemented")
}

// parseSize parses size strings like "100M", "1G", etc.
func (e *Evaluator) parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Remove any whitespace
	sizeStr = strings.TrimSpace(sizeStr)

	// Handle units
	multiplier := int64(1)
	unit := strings.ToUpper(sizeStr[len(sizeStr)-1:])

	switch unit {
	case "K":
		multiplier = 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case "M":
		multiplier = 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case "G":
		multiplier = 1024 * 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	}

	// Parse the numeric part
	value, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %w", err)
	}

	return int64(value * float64(multiplier)), nil
}

// parseDuration parses duration strings like "30d", "1w", etc.
func (e *Evaluator) parseDuration(durationStr string) (time.Duration, error) {
	if durationStr == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Remove any whitespace
	durationStr = strings.TrimSpace(durationStr)

	// Extract unit
	unit := strings.ToLower(durationStr[len(durationStr)-1:])
	valueStr := durationStr[:len(durationStr)-1]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}

	switch unit {
	case "s":
		return time.Duration(value * float64(time.Second)), nil
	case "m":
		return time.Duration(value * float64(time.Minute)), nil
	case "h":
		return time.Duration(value * float64(time.Hour)), nil
	case "d":
		return time.Duration(value * float64(24*time.Hour)), nil
	case "w":
		return time.Duration(value * float64(7*24*time.Hour)), nil
	case "y":
		return time.Duration(value * float64(365*24*time.Hour)), nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}

// parseTime parses time strings in various formats
func (e *Evaluator) parseTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	// Try different time formats
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	// Try relative time (like "+30d" or "-1w")
	if strings.HasPrefix(timeStr, "+") || strings.HasPrefix(timeStr, "-") {
		sign := 1
		if strings.HasPrefix(timeStr, "-") {
			sign = -1
		}

		durationStr := timeStr[1:]
		duration, err := e.parseDuration(durationStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative time: %w", err)
		}

		return time.Now().Add(time.Duration(sign) * duration), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}
