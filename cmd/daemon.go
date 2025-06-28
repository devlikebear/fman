/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/devlikebear/fman/internal/daemon"
	"github.com/spf13/cobra"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the fman background daemon",
	Long: `Manage the fman background daemon that handles asynchronous file scanning operations.

The daemon runs in the background and processes scan requests through a queue system.
It automatically starts when needed and can be managed manually using these commands.`,
}

// daemonStartCmd represents the daemon start command
var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the fman daemon",
	Long: `Start the fman background daemon.

The daemon will run in the background and accept scan requests through Unix domain sockets.
If the daemon is already running, this command will show its current status.`,
	RunE: runDaemonStart,
}

// daemonStopCmd represents the daemon stop command
var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the fman daemon",
	Long: `Stop the fman background daemon gracefully.

This will terminate the daemon process and cancel any pending scan operations.
Running scans will be completed before the daemon shuts down.`,
	RunE: runDaemonStop,
}

// daemonStatusCmd represents the daemon status command
var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long: `Show the current status of the fman background daemon.

This includes information about:
- Whether the daemon is running
- Process ID (PID)
- Socket file location
- Number of jobs in queue
- Daemon uptime and statistics`,
	RunE: runDaemonStatus,
}

// daemonRestartCmd represents the daemon restart command
var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the fman daemon",
	Long: `Restart the fman background daemon.

This will stop the current daemon (if running) and start a new one.
Any pending operations will be preserved in the queue.`,
	RunE: runDaemonRestart,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
}

// runDaemonStart handles the daemon start command
func runDaemonStart(cmd *cobra.Command, args []string) error {
	client := daemon.NewDaemonClient(nil)
	defer client.Disconnect()

	// Check if daemon is already running
	if client.IsDaemonRunning() {
		fmt.Println("✅ Daemon is already running")
		return runDaemonStatus(cmd, args)
	}

	fmt.Println("🚀 Starting fman daemon...")

	err := client.StartDaemon()
	if err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait a moment for daemon to fully start
	time.Sleep(500 * time.Millisecond)

	// Verify daemon started successfully
	if !client.IsDaemonRunning() {
		return fmt.Errorf("daemon failed to start properly")
	}

	fmt.Println("✅ Daemon started successfully")
	return runDaemonStatus(cmd, args)
}

// runDaemonStop handles the daemon stop command
func runDaemonStop(cmd *cobra.Command, args []string) error {
	client := daemon.NewDaemonClient(nil)
	defer client.Disconnect()

	// Check if daemon is running
	if !client.IsDaemonRunning() {
		fmt.Println("ℹ️  Daemon is not running")
		return nil
	}

	fmt.Println("🛑 Stopping fman daemon...")

	err := client.StopDaemon()
	if err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}

	// Wait for daemon to shut down
	timeout := time.Now().Add(5 * time.Second)
	for time.Now().Before(timeout) {
		if !client.IsDaemonRunning() {
			fmt.Println("✅ Daemon stopped successfully")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon did not stop within timeout")
}

// runDaemonStatus handles the daemon status command
func runDaemonStatus(cmd *cobra.Command, args []string) error {
	client := daemon.NewDaemonClient(nil)
	defer client.Disconnect()

	if !client.IsDaemonRunning() {
		fmt.Println("❌ Daemon is not running")
		return nil
	}

	// Connect to daemon and get status
	err := client.Connect()
	if err != nil {
		fmt.Println("❌ Daemon process exists but is not responding")
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}

	status, err := client.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}

	// Calculate uptime
	uptime := time.Since(status.StartedAt)

	// Display daemon status
	fmt.Println("✅ Daemon is running")
	fmt.Printf("📍 PID: %d\n", status.PID)
	fmt.Printf("⏰ Uptime: %s\n", formatDuration(uptime))
	fmt.Printf("📊 Queue Status:\n")
	fmt.Printf("   - Pending: %d jobs\n", status.QueuedJobs)
	fmt.Printf("   - Running: %d jobs\n", status.ActiveJobs)
	fmt.Printf("   - Completed: %d jobs\n", status.CompletedJobs)
	fmt.Printf("   - Failed: %d jobs\n", status.FailedJobs)
	fmt.Printf("👥 Workers: %d active\n", status.Workers)

	return nil
}

// runDaemonRestart handles the daemon restart command
func runDaemonRestart(cmd *cobra.Command, args []string) error {
	client := daemon.NewDaemonClient(nil)
	defer client.Disconnect()

	wasRunning := client.IsDaemonRunning()

	if wasRunning {
		fmt.Println("🛑 Stopping current daemon...")
		err := runDaemonStop(cmd, args)
		if err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		// Give it a moment to fully shut down
		time.Sleep(1 * time.Second)
	}

	fmt.Println("🚀 Starting daemon...")
	return runDaemonStart(cmd, args)
}

// ensureDaemonRunning ensures the daemon is running, starting it if necessary
func ensureDaemonRunning() error {
	client := daemon.NewDaemonClient(nil)
	defer client.Disconnect()

	if client.IsDaemonRunning() {
		return nil // Already running
	}

	// Start daemon silently
	err := client.StartDaemon()
	if err != nil {
		return fmt.Errorf("failed to auto-start daemon: %w", err)
	}

	// Wait for daemon to start - reduced timeout for tests
	timeout := time.Now().Add(500 * time.Millisecond) // Reduced from 3 seconds
	for time.Now().Before(timeout) {
		if client.IsDaemonRunning() {
			return nil
		}
		time.Sleep(50 * time.Millisecond) // Reduced from 100ms
	}

	return fmt.Errorf("daemon auto-start timeout")
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) - 60*minutes
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) - 60*hours
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// getPIDFromFile reads PID from the daemon PID file
func getPIDFromFile() (int, error) {
	config := daemon.GetDefaultConfig()
	pidFile := config.PIDPath

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}
