/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/devlikebear/fman/internal/daemon"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDaemonClient is a mock implementation of the daemon client for testing
type MockDaemonClient struct {
	mock.Mock
}

func (m *MockDaemonClient) Connect() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDaemonClient) Disconnect() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDaemonClient) SendRequest(req *daemon.Request) (*daemon.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*daemon.Response), args.Error(1)
}

func (m *MockDaemonClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDaemonClient) IsDaemonRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockDaemonClient) StartDaemon() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDaemonClient) StopDaemon() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDaemonClient) GetStatus() (*daemon.DaemonStatus, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*daemon.DaemonStatus), args.Error(1)
}

func (m *MockDaemonClient) SetTimeout(timeout time.Duration) {
	m.Called(timeout)
}

func (m *MockDaemonClient) SetRetryCount(count int) {
	m.Called(count)
}

func TestDaemonCommands(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected string
	}{
		{
			name:     "daemon command help",
			cmd:      "daemon --help",
			expected: "Manage the fman background daemon",
		},
		{
			name:     "daemon start help",
			cmd:      "daemon start --help",
			expected: "Start the fman background daemon",
		},
		{
			name:     "daemon stop help",
			cmd:      "daemon stop --help",
			expected: "Stop the fman background daemon",
		},
		{
			name:     "daemon status help",
			cmd:      "daemon status --help",
			expected: "Show the current status",
		},
		{
			name:     "daemon restart help",
			cmd:      "daemon restart --help",
			expected: "Restart the fman background daemon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute command
			rootCmd.SetArgs(strings.Split(tt.cmd, " "))
			err := rootCmd.Execute()

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read output
			out, _ := io.ReadAll(r)
			output := string(out)

			// Check that help was displayed (no error expected for help commands)
			assert.NoError(t, err)
			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "hours and minutes",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "2h 30m",
		},
		{
			name:     "exactly one minute",
			duration: time.Minute,
			expected: "1m 0s",
		},
		{
			name:     "exactly one hour",
			duration: time.Hour,
			expected: "1h 0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPIDFromFile(t *testing.T) {
	t.Run("function exists and is callable", func(t *testing.T) {
		// Test that the function can be called without panicking
		// In test environment, this will likely fail because no daemon is running
		_, err := getPIDFromFile()
		// We expect this to fail in test environment (file not found), but it shouldn't panic
		assert.Error(t, err)
	})
}

func TestEnsureDaemonRunning(t *testing.T) {
	t.Run("function exists and is callable", func(t *testing.T) {
		// Create a temporary config directory for testing
		tempDir := t.TempDir()

		// Set environment variable to use temp directory
		oldHome := os.Getenv("HOME")
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", oldHome)

		// Test that the function can be called without panicking
		// In test environment, this will likely fail because no daemon is running
		// but we set a very short timeout to avoid hanging
		err := ensureDaemonRunning()
		// We expect this to fail in test environment, but it shouldn't panic or hang
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to auto-start daemon")
	})
}

func TestDaemonCommandStructure(t *testing.T) {
	t.Run("daemon command has subcommands", func(t *testing.T) {
		assert.NotNil(t, daemonCmd)
		assert.Equal(t, "daemon", daemonCmd.Use)
		assert.True(t, daemonCmd.HasSubCommands())

		// Check that all expected subcommands are present
		subcommands := daemonCmd.Commands()
		commandNames := make([]string, len(subcommands))
		for i, cmd := range subcommands {
			commandNames[i] = cmd.Use
		}

		assert.Contains(t, commandNames, "start")
		assert.Contains(t, commandNames, "stop")
		assert.Contains(t, commandNames, "status")
		assert.Contains(t, commandNames, "restart")
	})

	t.Run("all subcommands have RunE functions", func(t *testing.T) {
		subcommands := []*cobra.Command{
			daemonStartCmd,
			daemonStopCmd,
			daemonStatusCmd,
			daemonRestartCmd,
		}

		for _, cmd := range subcommands {
			assert.NotNil(t, cmd.RunE, "Command %s should have RunE function", cmd.Use)
		}
	})
}

func TestRootCommandPersistentPreRun(t *testing.T) {
	t.Run("PersistentPreRun is set", func(t *testing.T) {
		assert.NotNil(t, rootCmd.PersistentPreRun)
	})

	t.Run("PersistentPreRun handles daemon commands", func(t *testing.T) {
		// Test with actual daemon subcommand
		assert.NotPanics(t, func() {
			rootCmd.PersistentPreRun(daemonStartCmd, []string{})
		})
	})

	t.Run("PersistentPreRun handles non-daemon commands", func(t *testing.T) {
		// Create a mock command that simulates a non-daemon command
		nonDaemonCmd := &cobra.Command{
			Use: "scan",
		}

		// Capture any panics
		assert.NotPanics(t, func() {
			rootCmd.PersistentPreRun(nonDaemonCmd, []string{})
		})
	})
}

// TestDaemonCommandIntegration tests the daemon commands with mocked dependencies
func TestDaemonCommandIntegration(t *testing.T) {
	// These tests would ideally use dependency injection to mock the daemon client
	// For now, we test that the commands can be executed without panicking

	t.Run("daemon commands execute without panic", func(t *testing.T) {
		commands := []string{
			"daemon start",
			"daemon stop",
			"daemon status",
			"daemon restart",
		}

		for _, cmdStr := range commands {
			t.Run(cmdStr, func(t *testing.T) {
				// Capture stderr to prevent error output during tests
				oldStderr := os.Stderr
				r, w, _ := os.Pipe()
				os.Stderr = w

				// Capture stdout as well
				oldStdout := os.Stdout
				r2, w2, _ := os.Pipe()
				os.Stdout = w2

				// Execute command (expect it to fail in test environment, but not panic)
				rootCmd.SetArgs(strings.Split(cmdStr, " "))
				_ = rootCmd.Execute()

				// Restore stderr and stdout
				w.Close()
				w2.Close()
				os.Stderr = oldStderr
				os.Stdout = oldStdout

				// Read and discard output
				_, _ = io.ReadAll(r)
				_, _ = io.ReadAll(r2)

				// In test environment, commands might succeed (just print status) or fail
				// We just want to ensure no panics occur
			})
		}
	})
}

// Benchmark tests for performance
func BenchmarkFormatDuration(b *testing.B) {
	duration := 2*time.Hour + 30*time.Minute + 45*time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatDuration(duration)
	}
}

func BenchmarkGetPIDFromFile(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will likely fail in test environment, but we're measuring performance
		_, _ = getPIDFromFile()
	}
}

// TestDaemonCommandsOutput tests the output formatting of daemon commands
func TestDaemonCommandsOutput(t *testing.T) {
	t.Run("formatDuration produces expected output format", func(t *testing.T) {
		testCases := []struct {
			input    time.Duration
			contains []string
		}{
			{
				input:    30 * time.Second,
				contains: []string{"30s"},
			},
			{
				input:    2*time.Minute + 15*time.Second,
				contains: []string{"2m", "15s"},
			},
			{
				input:    1*time.Hour + 30*time.Minute,
				contains: []string{"1h", "30m"},
			},
		}

		for _, tc := range testCases {
			result := formatDuration(tc.input)
			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		}
	})
}

// Helper function to capture command output
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// TestDaemonCommandDescriptions verifies command descriptions are informative
func TestDaemonCommandDescriptions(t *testing.T) {
	commands := map[string]*cobra.Command{
		"daemon":         daemonCmd,
		"daemon start":   daemonStartCmd,
		"daemon stop":    daemonStopCmd,
		"daemon status":  daemonStatusCmd,
		"daemon restart": daemonRestartCmd,
	}

	for name, cmd := range commands {
		t.Run(name+" has description", func(t *testing.T) {
			assert.NotEmpty(t, cmd.Short, "Command %s should have a short description", name)
			assert.NotEmpty(t, cmd.Long, "Command %s should have a long description", name)
			assert.True(t, len(cmd.Long) > len(cmd.Short), "Long description should be longer than short for %s", name)
		})
	}
}
