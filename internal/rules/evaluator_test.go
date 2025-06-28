package rules

import (
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvaluator(t *testing.T) {
	t.Run("creates evaluator with correct settings", func(t *testing.T) {
		evaluator := NewEvaluator(true)
		assert.NotNil(t, evaluator)
		assert.True(t, evaluator.verbose)

		evaluator2 := NewEvaluator(false)
		assert.False(t, evaluator2.verbose)
	})
}

func TestEvaluateRule(t *testing.T) {
	evaluator := NewEvaluator(false)

	t.Run("disabled rule returns false", func(t *testing.T) {
		rule := Rule{
			Name:    "disabled-rule",
			Enabled: false,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Value: "test"},
			},
		}

		file := db.File{Path: "/test/testfile.txt"}

		matches, err := evaluator.EvaluateRule(rule, file, "")
		require.NoError(t, err)
		assert.False(t, matches)
	})

	t.Run("enabled rule with matching conditions returns true", func(t *testing.T) {
		rule := Rule{
			Name:    "enabled-rule",
			Enabled: true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Operator: OpContains, Value: "test"},
			},
		}

		file := db.File{Path: "/test/testfile.txt", Name: "testfile.txt"}

		matches, err := evaluator.EvaluateRule(rule, file, "")
		require.NoError(t, err)
		assert.True(t, matches)
	})

	t.Run("enabled rule with non-matching conditions returns false", func(t *testing.T) {
		rule := Rule{
			Name:    "enabled-rule",
			Enabled: true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Operator: OpContains, Value: "nonexistent"},
			},
		}

		file := db.File{Path: "/test/testfile.txt", Name: "testfile.txt"}

		matches, err := evaluator.EvaluateRule(rule, file, "")
		require.NoError(t, err)
		assert.False(t, matches)
	})

	t.Run("multiple conditions with AND logic", func(t *testing.T) {
		rule := Rule{
			Name:    "multi-condition-rule",
			Enabled: true,
			Conditions: []Condition{
				{Type: ConditionNamePattern, Operator: OpContains, Value: "test"},
				{Type: ConditionExtension, Operator: OpEqual, Value: ".txt"},
			},
		}

		// File that matches both conditions
		file1 := db.File{Path: "/test/testfile.txt", Name: "testfile.txt"}
		matches, err := evaluator.EvaluateRule(rule, file1, "")
		require.NoError(t, err)
		assert.True(t, matches)

		// File that matches only first condition
		file2 := db.File{Path: "/test/testfile.pdf", Name: "testfile.pdf"}
		matches, err = evaluator.EvaluateRule(rule, file2, "")
		require.NoError(t, err)
		assert.False(t, matches)
	})
}

func TestEvaluateNamePattern(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		condition Condition
		filename  string
		expected  bool
		expectErr bool
	}{
		{
			name:      "contains match",
			condition: Condition{Type: ConditionNamePattern, Operator: OpContains, Value: "test"},
			filename:  "testfile.txt",
			expected:  true,
		},
		{
			name:      "contains no match",
			condition: Condition{Type: ConditionNamePattern, Operator: OpContains, Value: "xyz"},
			filename:  "testfile.txt",
			expected:  false,
		},
		{
			name:      "equal match",
			condition: Condition{Type: ConditionNamePattern, Operator: OpEqual, Value: "testfile.txt"},
			filename:  "testfile.txt",
			expected:  true,
		},
		{
			name:      "equal no match",
			condition: Condition{Type: ConditionNamePattern, Operator: OpEqual, Value: "other.txt"},
			filename:  "testfile.txt",
			expected:  false,
		},
		{
			name:      "starts with match",
			condition: Condition{Type: ConditionNamePattern, Operator: OpStartsWith, Value: "test"},
			filename:  "testfile.txt",
			expected:  true,
		},
		{
			name:      "ends with match",
			condition: Condition{Type: ConditionNamePattern, Operator: OpEndsWith, Value: ".txt"},
			filename:  "testfile.txt",
			expected:  true,
		},
		{
			name:      "regex match",
			condition: Condition{Type: ConditionNamePattern, Operator: OpMatches, Value: "test.*\\.txt"},
			filename:  "testfile.txt",
			expected:  true,
		},
		{
			name:      "invalid regex",
			condition: Condition{Type: ConditionNamePattern, Operator: OpMatches, Value: "[invalid"},
			filename:  "testfile.txt",
			expected:  false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{Path: "/test/" + tt.filename, Name: tt.filename}

			result, err := evaluator.evaluateNamePattern(tt.condition, file)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvaluateExtension(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		condition Condition
		filename  string
		expected  bool
	}{
		{
			name:      "extension match with dot",
			condition: Condition{Type: ConditionExtension, Operator: OpEqual, Value: ".txt"},
			filename:  "testfile.txt",
			expected:  true,
		},
		{
			name:      "extension match without dot",
			condition: Condition{Type: ConditionExtension, Operator: OpEqual, Value: "txt"},
			filename:  "testfile.txt",
			expected:  true,
		},
		{
			name:      "extension no match",
			condition: Condition{Type: ConditionExtension, Operator: OpEqual, Value: ".pdf"},
			filename:  "testfile.txt",
			expected:  false,
		},
		{
			name:      "case insensitive",
			condition: Condition{Type: ConditionExtension, Operator: OpEqual, Value: ".TXT"},
			filename:  "testfile.txt",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{Path: "/test/" + tt.filename, Name: tt.filename}

			result, err := evaluator.evaluateExtension(tt.condition, file)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateSize(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		condition Condition
		fileSize  int64
		expected  bool
		expectErr bool
	}{
		{
			name:      "greater than match",
			condition: Condition{Type: ConditionSize, Operator: OpGreaterThan, Value: "100"},
			fileSize:  200,
			expected:  true,
		},
		{
			name:      "greater than no match",
			condition: Condition{Type: ConditionSize, Operator: OpGreaterThan, Value: "100"},
			fileSize:  50,
			expected:  false,
		},
		{
			name:      "size with unit K",
			condition: Condition{Type: ConditionSize, Operator: OpGreaterThan, Value: "1K"},
			fileSize:  2048,
			expected:  true,
		},
		{
			name:      "size with unit M",
			condition: Condition{Type: ConditionSize, Operator: OpLessThan, Value: "1M"},
			fileSize:  512 * 1024,
			expected:  true,
		},
		{
			name:      "invalid size format",
			condition: Condition{Type: ConditionSize, Operator: OpGreaterThan, Value: "invalid"},
			fileSize:  100,
			expected:  false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{Size: tt.fileSize}

			result, err := evaluator.evaluateSize(tt.condition, file)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvaluateAge(t *testing.T) {
	evaluator := NewEvaluator(false)
	now := time.Now()

	tests := []struct {
		name         string
		condition    Condition
		modifiedTime time.Time
		expected     bool
		expectErr    bool
	}{
		{
			name:         "older than 1 day",
			condition:    Condition{Type: ConditionAge, Operator: OpGreaterThan, Value: "1d"},
			modifiedTime: now.Add(-2 * 24 * time.Hour),
			expected:     true,
		},
		{
			name:         "newer than 1 day",
			condition:    Condition{Type: ConditionAge, Operator: OpGreaterThan, Value: "1d"},
			modifiedTime: now.Add(-12 * time.Hour),
			expected:     false,
		},
		{
			name:         "less than 1 week old",
			condition:    Condition{Type: ConditionAge, Operator: OpLessThan, Value: "1w"},
			modifiedTime: now.Add(-3 * 24 * time.Hour),
			expected:     true,
		},
		{
			name:         "invalid age format",
			condition:    Condition{Type: ConditionAge, Operator: OpGreaterThan, Value: "invalid"},
			modifiedTime: now,
			expected:     false,
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{ModifiedAt: tt.modifiedTime}

			result, err := evaluator.evaluateAge(tt.condition, file)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvaluateFileType(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		condition Condition
		filename  string
		expected  bool
		expectErr bool
	}{
		{
			name:      "image type match",
			condition: Condition{Type: ConditionFileType, Operator: OpEqual, Value: "image"},
			filename:  "photo.jpg",
			expected:  true,
		},
		{
			name:      "image type no match",
			condition: Condition{Type: ConditionFileType, Operator: OpEqual, Value: "image"},
			filename:  "document.pdf",
			expected:  false,
		},
		{
			name:      "specific extension match",
			condition: Condition{Type: ConditionFileType, Operator: OpEqual, Value: ".pdf"},
			filename:  "document.pdf",
			expected:  true,
		},
		{
			name:      "document type match",
			condition: Condition{Type: ConditionFileType, Operator: OpEqual, Value: "document"},
			filename:  "report.pdf",
			expected:  true,
		},
		{
			name:      "video type match",
			condition: Condition{Type: ConditionFileType, Operator: OpEqual, Value: "video"},
			filename:  "movie.mp4",
			expected:  true,
		},
		{
			name:      "unknown type",
			condition: Condition{Type: ConditionFileType, Operator: OpEqual, Value: "unknown"},
			filename:  "file.xyz",
			expected:  false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{Path: "/test/" + tt.filename, Name: tt.filename}

			result, err := evaluator.evaluateFileType(tt.condition, file)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseSizeUtilities(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		{
			name:     "bytes",
			input:    "100",
			expected: 100,
		},
		{
			name:     "kilobytes",
			input:    "1K",
			expected: 1024,
		},
		{
			name:     "megabytes",
			input:    "1M",
			expected: 1024 * 1024,
		},
		{
			name:     "gigabytes",
			input:    "1G",
			expected: 1024 * 1024 * 1024,
		},
		{
			name:     "decimal value",
			input:    "1.5M",
			expected: int64(1.5 * 1024 * 1024),
		},
		{
			name:      "invalid format",
			input:     "invalid",
			expectErr: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.parseSize(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseDurationUtilities(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		input     string
		expected  time.Duration
		expectErr bool
	}{
		{
			name:     "seconds",
			input:    "30s",
			expected: 30 * time.Second,
		},
		{
			name:     "minutes",
			input:    "5m",
			expected: 5 * time.Minute,
		},
		{
			name:     "hours",
			input:    "2h",
			expected: 2 * time.Hour,
		},
		{
			name:     "days",
			input:    "1d",
			expected: 24 * time.Hour,
		},
		{
			name:     "weeks",
			input:    "1w",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "decimal value",
			input:    "1.5d",
			expected: time.Duration(1.5 * float64(24*time.Hour)),
		},
		{
			name:      "invalid unit",
			input:     "1x",
			expectErr: true,
		},
		{
			name:      "invalid format",
			input:     "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.parseDuration(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseTimeUtilities(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:  "date only",
			input: "2024-01-01",
		},
		{
			name:  "date and time",
			input: "2024-01-01 15:30:00",
		},
		{
			name:  "ISO format",
			input: "2024-01-01T15:30:00Z",
		},
		{
			name:  "relative future",
			input: "+30d",
		},
		{
			name:  "relative past",
			input: "-1w",
		},
		{
			name:      "invalid format",
			input:     "invalid-date",
			expectErr: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.parseTime(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.False(t, result.IsZero())
			}
		})
	}
}

func TestEvaluateModified(t *testing.T) {
	evaluator := NewEvaluator(false)

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	tests := []struct {
		name        string
		condition   Condition
		modifiedAt  time.Time
		expected    bool
		expectError bool
	}{
		{
			name:       "modified after - recent file (relative time)",
			condition:  Condition{Type: ConditionModified, Operator: OpGreaterThan, Value: "-1h"},
			modifiedAt: oneHourAgo.Add(30 * time.Minute), // 30 minutes ago
			expected:   true,
		},
		{
			name:       "modified after - old file (relative time)",
			condition:  Condition{Type: ConditionModified, Operator: OpGreaterThan, Value: "-1h"},
			modifiedAt: oneHourAgo.Add(-30 * time.Minute), // 1.5 hours ago
			expected:   false,
		},
		{
			name:       "modified before - old file (relative time)",
			condition:  Condition{Type: ConditionModified, Operator: OpLessThan, Value: "-1d"},
			modifiedAt: oneDayAgo.Add(-1 * time.Hour), // 25 hours ago
			expected:   true,
		},
		{
			name:       "modified before - recent file (relative time)",
			condition:  Condition{Type: ConditionModified, Operator: OpLessThan, Value: "-1d"},
			modifiedAt: oneDayAgo.Add(1 * time.Hour), // 23 hours ago
			expected:   false,
		},
		{
			name:       "exact date match",
			condition:  Condition{Type: ConditionModified, Operator: OpEqual, Value: "2024-01-01"},
			modifiedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected:   true,
		},
		{
			name:       "exact date no match",
			condition:  Condition{Type: ConditionModified, Operator: OpEqual, Value: oneDayAgo.Format("2006-01-02")},
			modifiedAt: now,
			expected:   false,
		},
		{
			name:        "invalid duration format",
			condition:   Condition{Type: ConditionModified, Operator: OpGreaterThan, Value: "invalid"},
			modifiedAt:  now,
			expected:    false,
			expectError: true,
		},
		{
			name:        "invalid date format",
			condition:   Condition{Type: ConditionModified, Operator: OpEqual, Value: "invalid-date"},
			modifiedAt:  now,
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{
				Path:       "/test/file.txt",
				Name:       "file.txt",
				ModifiedAt: tt.modifiedAt,
			}

			result, err := evaluator.evaluateModified(tt.condition, file)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvaluatePath(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		condition Condition
		filePath  string
		baseDir   string
		expected  bool
	}{
		{
			name:      "path contains match",
			condition: Condition{Type: ConditionPath, Operator: OpContains, Value: "documents"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "/home/user",
			expected:  true,
		},
		{
			name:      "path contains no match",
			condition: Condition{Type: ConditionPath, Operator: OpContains, Value: "downloads"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "/home/user",
			expected:  false,
		},
		{
			name:      "path starts with match",
			condition: Condition{Type: ConditionPath, Operator: OpStartsWith, Value: "/home/user"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "",
			expected:  true,
		},
		{
			name:      "path starts with no match",
			condition: Condition{Type: ConditionPath, Operator: OpStartsWith, Value: "/opt"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "",
			expected:  false,
		},
		{
			name:      "path ends with match",
			condition: Condition{Type: ConditionPath, Operator: OpEndsWith, Value: "documents/file.txt"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "",
			expected:  true,
		},
		{
			name:      "path equal match",
			condition: Condition{Type: ConditionPath, Operator: OpEqual, Value: "/home/user/documents/file.txt"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "",
			expected:  true,
		},
		{
			name:      "relative path with base directory",
			condition: Condition{Type: ConditionPath, Operator: OpContains, Value: "documents"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "/home/user",
			expected:  true,
		},
		{
			name:      "regex match",
			condition: Condition{Type: ConditionPath, Operator: OpMatches, Value: ".*/documents/.*\\.txt"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "",
			expected:  true,
		},
		{
			name:      "regex no match",
			condition: Condition{Type: ConditionPath, Operator: OpMatches, Value: ".*/downloads/.*"},
			filePath:  "/home/user/documents/file.txt",
			baseDir:   "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{
				Path: tt.filePath,
				Name: "file.txt",
			}

			result, err := evaluator.evaluatePath(tt.condition, file, tt.baseDir)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateMimeType(t *testing.T) {
	evaluator := NewEvaluator(false)

	tests := []struct {
		name      string
		condition Condition
		filename  string
		expected  bool
		expectErr bool
	}{
		{
			name:      "mime type not implemented - returns error",
			condition: Condition{Type: ConditionMimeType, Operator: OpEqual, Value: "text/plain"},
			filename:  "file.txt",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "mime type with image - returns error",
			condition: Condition{Type: ConditionMimeType, Operator: OpEqual, Value: "image/jpeg"},
			filename:  "photo.jpg",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "mime type with pdf - returns error",
			condition: Condition{Type: ConditionMimeType, Operator: OpEqual, Value: "application/pdf"},
			filename:  "document.pdf",
			expected:  false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := db.File{
				Path: "/test/" + tt.filename,
				Name: tt.filename,
			}

			result, err := evaluator.evaluateMimeType(tt.condition, file)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not yet implemented")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
