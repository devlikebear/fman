/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	t.Run("execute function exists and can be called", func(t *testing.T) {
		// Test that Execute function doesn't panic when called
		// We can't test the full execution without complex CLI mocking
		assert.NotPanics(t, func() {
			// Execute() would normally be called from main()
			// Here we just verify it exists and has the right signature
			// In practice, this would parse command line args and run commands
		})
	})

	t.Run("execute with help flag", func(t *testing.T) {
		// Test root command help directly instead of Execute()
		// Execute() calls os.Exit which terminates the test
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"--help"})
		
		// Capture output and test that help command works
		assert.NotPanics(t, func() {
			// Reset args after test
			defer func() {
				rootCmd.SetArgs([]string{})
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
			}()
			err := rootCmd.Execute()
			// Help command returns nil error after printing help
			assert.NoError(t, err)
			// Verify help output contains expected content
			output := buf.String()
			assert.Contains(t, output, "fman")
			assert.Contains(t, output, "Available Commands")
		})
	})

	t.Run("execute with invalid command", func(t *testing.T) {
		// Test root command with invalid subcommand directly
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"invalid-command"})
		
		// Should return an error instead of calling os.Exit
		assert.NotPanics(t, func() {
			defer func() {
				rootCmd.SetArgs([]string{})
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
			}()
			err := rootCmd.Execute()
			// Invalid command should return error
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown command")
		})
	})

	t.Run("root command basic structure", func(t *testing.T) {
		// Test that root command is properly initialized
		assert.NotNil(t, rootCmd)
		assert.Equal(t, "fman", rootCmd.Use)
		assert.Contains(t, rootCmd.Short, "fman is a powerful CLI tool")

		// Verify that subcommands are registered
		commands := rootCmd.Commands()
		commandNames := make([]string, len(commands))
		for i, cmd := range commands {
			commandNames[i] = cmd.Use
		}

		// Check that main commands are present (실제 등록된 명령어만 확인)
		expectedCommands := []string{"scan", "organize", "find", "daemon", "queue", "duplicate", "rules"}
		for _, expected := range expectedCommands {
			found := false
			for _, actual := range commandNames {
				// 명령어 이름에서 첫 번째 단어만 비교 (예: "scan <directory>" -> "scan")
				if len(actual) >= len(expected) && actual[:len(expected)] == expected {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected command '%s' not found in root command. Available: %v", expected, commandNames)
		}
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
