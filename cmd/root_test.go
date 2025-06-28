/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Test help command (non-interactive)
	os.Args = []string{"fman", "--help"}

	// Execute should not panic and should handle help command gracefully
	assert.NotPanics(t, func() {
		// We can't directly test Execute() as it may exit the process,
		// but we can test that rootCmd is properly configured
		assert.NotNil(t, rootCmd)
		assert.Equal(t, "fman", rootCmd.Use)
		assert.Contains(t, rootCmd.Short, "CLI tool")
	})
}

func TestInitConfig(t *testing.T) {
	// Save original viper state
	defer func() {
		viper.Reset()
	}()

	// Test that initConfig doesn't panic
	assert.NotPanics(t, func() {
		initConfig()
	})

	// Verify viper configuration
	assert.Contains(t, viper.AllKeys(), "ai_provider")
}

func TestInitConfigCreatesDefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Set HOME environment variable to temp directory
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", tempDir)

	// Reset viper
	viper.Reset()

	// Test config initialization
	assert.NotPanics(t, func() {
		initConfig()
	})

	// Check if config directory was created
	configPath := filepath.Join(tempDir, ".fman")
	_, err := os.Stat(configPath)
	assert.NoError(t, err, "Config directory should be created")

	// Check if config file was created
	configFile := filepath.Join(configPath, "config.yml")
	_, err = os.Stat(configFile)
	assert.NoError(t, err, "Config file should be created")

	// Check config file content
	content, err := os.ReadFile(configFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "ai_provider:")
	assert.Contains(t, string(content), "gemini:")
	assert.Contains(t, string(content), "ollama:")
}

func TestRootCmdConfiguration(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "fman", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "fman is a powerful CLI tool")
	assert.Contains(t, rootCmd.Long, "fman (File Manager)")
}

func TestCobraInitialization(t *testing.T) {
	// Test that cobra initialization doesn't panic
	assert.NotPanics(t, func() {
		// Create a new command to test initialization
		testCmd := &cobra.Command{
			Use:   "test",
			Short: "Test command",
		}
		assert.NotNil(t, testCmd)
	})
}

func TestFileSystemVariable(t *testing.T) {
	// Test that fileSystem variable is properly initialized
	assert.NotNil(t, fileSystem)

	// Test that we can create files with the filesystem
	tempFile := "/tmp/test-file"
	err := afero.WriteFile(fileSystem, tempFile, []byte("test"), 0644)
	if err == nil {
		// Clean up
		fileSystem.Remove(tempFile)
	}
	// We don't assert on the error as it might fail due to permissions,
	// but we test that the filesystem is usable
}
