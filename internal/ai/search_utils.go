package ai

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/devlikebear/fman/internal/db"
)

// buildSearchPrompt creates a prompt for converting natural language to search criteria
func buildSearchPrompt(query string) string {
	return fmt.Sprintf(`You are a file search query parser. Convert the following natural language query into a JSON object that represents file search criteria.

Rules:
1. Return ONLY a valid JSON object, no explanation or additional text.
2. Use these field names exactly: namePattern, minSize, maxSize, modifiedAfter, modifiedBefore, searchDir, fileTypes
3. For file sizes, use bytes (e.g., 1048576 for 1MB, 1073741824 for 1GB)
4. For dates, use RFC3339 format (e.g., "2024-01-15T00:00:00Z")
5. For file types, use extensions with dots (e.g., [".jpg", ".png", ".pdf"])
6. Only include fields that are relevant to the query
7. For relative dates like "last week", "yesterday", calculate from current time: %s

Common file type mappings:
- images: [".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg"]
- videos: [".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v"]
- documents: [".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".pages"]
- audio: [".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a"]
- archives: [".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz"]

Size interpretations:
- "large files" or "big files": minSize of 100MB (104857600 bytes)
- "small files": maxSize of 1MB (1048576 bytes)
- "huge files": minSize of 1GB (1073741824 bytes)

Examples:
Query: "find large images modified last week"
Output: {"fileTypes": [".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", ".svg"], "minSize": 104857600, "modifiedAfter": "2024-01-08T00:00:00Z"}

Query: "show me PDF documents from Downloads folder"
Output: {"fileTypes": [".pdf"], "searchDir": "/Users/username/Downloads"}

Query: "files with report in name bigger than 10MB"
Output: {"namePattern": "report", "minSize": 10485760}

Now convert this query:
"%s"`, time.Now().Format(time.RFC3339), query)
}

// parseSearchCriteriaFromJSON parses the AI response JSON into SearchCriteria
func parseSearchCriteriaFromJSON(jsonStr string) (*db.SearchCriteria, error) {
	// Clean up the JSON string (remove markdown code blocks if present)
	jsonStr = cleanJSONString(jsonStr)

	var rawCriteria map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawCriteria); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
	}

	criteria := &db.SearchCriteria{}

	// Parse string fields
	if namePattern, ok := rawCriteria["namePattern"].(string); ok {
		criteria.NamePattern = namePattern
	}
	if searchDir, ok := rawCriteria["searchDir"].(string); ok {
		criteria.SearchDir = searchDir
	}

	// Parse size fields
	if minSize, ok := rawCriteria["minSize"].(float64); ok {
		size := int64(minSize)
		criteria.MinSize = &size
	}
	if maxSize, ok := rawCriteria["maxSize"].(float64); ok {
		size := int64(maxSize)
		criteria.MaxSize = &size
	}

	// Parse date fields
	if modifiedAfter, ok := rawCriteria["modifiedAfter"].(string); ok {
		if t, err := time.Parse(time.RFC3339, modifiedAfter); err == nil {
			criteria.ModifiedAfter = &t
		}
	}
	if modifiedBefore, ok := rawCriteria["modifiedBefore"].(string); ok {
		if t, err := time.Parse(time.RFC3339, modifiedBefore); err == nil {
			criteria.ModifiedBefore = &t
		}
	}

	// Parse file types
	if fileTypes, ok := rawCriteria["fileTypes"].([]interface{}); ok {
		for _, ft := range fileTypes {
			if typeStr, ok := ft.(string); ok {
				criteria.FileTypes = append(criteria.FileTypes, typeStr)
			}
		}
	}

	return criteria, nil
}

// cleanJSONString removes markdown code blocks and extra whitespace
func cleanJSONString(jsonStr string) string {
	// Remove markdown code blocks and clean up response
	lines := []string{}
	inCodeBlock := false

	for _, char := range jsonStr {
		line := string(char)
		if line == "`" {
			continue
		}
		if len(line) > 0 && (line[0] == '{' || inCodeBlock) {
			inCodeBlock = true
			lines = append(lines, line)
			if line[len(line)-1:] == "}" {
				break
			}
		}
	}

	if len(lines) > 0 {
		result := ""
		for _, line := range lines {
			result += line
		}
		return result
	}

	// If no code block found, try to extract JSON from the response
	start := -1
	end := -1
	for i, char := range jsonStr {
		if char == '{' && start == -1 {
			start = i
		}
		if char == '}' {
			end = i + 1
		}
	}

	if start != -1 && end != -1 && end > start {
		return jsonStr[start:end]
	}

	return jsonStr
}
