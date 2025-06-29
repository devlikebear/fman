package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildSearchPrompt(t *testing.T) {
	query := "find large images modified last week"
	prompt := buildSearchPrompt(query)

	assert.Contains(t, prompt, "file search query parser")
	assert.Contains(t, prompt, query)
	assert.Contains(t, prompt, "JSON object")
	assert.Contains(t, prompt, "namePattern")
	assert.Contains(t, prompt, "fileTypes")
	assert.Contains(t, prompt, time.Now().Format(time.RFC3339)[:10]) // Check current date is included
}

func TestParseSearchCriteriaFromJSON(t *testing.T) {
	t.Run("valid JSON with all fields", func(t *testing.T) {
		jsonStr := `{
			"namePattern": "test",
			"minSize": 1048576,
			"maxSize": 10485760,
			"modifiedAfter": "2024-01-15T00:00:00Z",
			"modifiedBefore": "2024-01-20T00:00:00Z",
			"searchDir": "/test/dir",
			"fileTypes": [".jpg", ".png"]
		}`

		criteria, err := parseSearchCriteriaFromJSON(jsonStr)

		assert.NoError(t, err)
		assert.Equal(t, "test", criteria.NamePattern)
		assert.Equal(t, int64(1048576), *criteria.MinSize)
		assert.Equal(t, int64(10485760), *criteria.MaxSize)
		assert.Equal(t, "/test/dir", criteria.SearchDir)
		assert.Equal(t, []string{".jpg", ".png"}, criteria.FileTypes)

		expectedAfter, _ := time.Parse(time.RFC3339, "2024-01-15T00:00:00Z")
		expectedBefore, _ := time.Parse(time.RFC3339, "2024-01-20T00:00:00Z")
		assert.Equal(t, expectedAfter, *criteria.ModifiedAfter)
		assert.Equal(t, expectedBefore, *criteria.ModifiedBefore)
	})

	t.Run("minimal JSON", func(t *testing.T) {
		jsonStr := `{"namePattern": "test"}`

		criteria, err := parseSearchCriteriaFromJSON(jsonStr)

		assert.NoError(t, err)
		assert.Equal(t, "test", criteria.NamePattern)
		assert.Nil(t, criteria.MinSize)
		assert.Nil(t, criteria.MaxSize)
		assert.Empty(t, criteria.SearchDir)
		assert.Empty(t, criteria.FileTypes)
	})

	t.Run("empty JSON", func(t *testing.T) {
		jsonStr := `{}`

		criteria, err := parseSearchCriteriaFromJSON(jsonStr)

		assert.NoError(t, err)
		assert.Empty(t, criteria.NamePattern)
		assert.Nil(t, criteria.MinSize)
		assert.Nil(t, criteria.MaxSize)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonStr := `{invalid json`

		criteria, err := parseSearchCriteriaFromJSON(jsonStr)

		assert.Error(t, err)
		assert.Nil(t, criteria)
		assert.Contains(t, err.Error(), "failed to parse AI response as JSON")
	})

	t.Run("JSON with markdown code blocks", func(t *testing.T) {
		jsonStr := "```json\n{\"namePattern\": \"test\"}\n```"

		criteria, err := parseSearchCriteriaFromJSON(jsonStr)

		assert.NoError(t, err)
		assert.Equal(t, "test", criteria.NamePattern)
	})
}

func TestCleanJSONString(t *testing.T) {
	t.Run("plain JSON", func(t *testing.T) {
		input := `{"test": "value"}`
		result := cleanJSONString(input)
		assert.Equal(t, input, result)
	})

	t.Run("JSON with backticks", func(t *testing.T) {
		input := "`{\"test\": \"value\"}`"
		result := cleanJSONString(input)
		assert.Equal(t, "{\"test\": \"value\"}", result)
	})

	t.Run("complex string with JSON", func(t *testing.T) {
		input := "Here is the JSON: {\"test\": \"value\"} and some more text"
		result := cleanJSONString(input)
		assert.Equal(t, "{\"test\": \"value\"}", result)
	})

	t.Run("no JSON found", func(t *testing.T) {
		input := "No JSON here"
		result := cleanJSONString(input)
		assert.Equal(t, input, result)
	})
}
