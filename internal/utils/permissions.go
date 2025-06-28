/*
Copyright © 2025 changheonshin
*/
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// IsRunningAsRoot checks if the current process is running with root privileges
func IsRunningAsRoot() bool {
	if runtime.GOOS == "windows" {
		// On Windows, check if running as administrator
		// This is a simplified check - in production you might want more robust detection
		return false // For now, we'll skip sudo functionality on Windows
	}
	return os.Geteuid() == 0
}

// IsPermissionError checks if an error is a permission error
func IsPermissionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for syscall.Errno permission errors
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.EACCES || errno == syscall.EPERM
	}

	// Check for string-based permission errors
	errStr := err.Error()
	return strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "operation not permitted") ||
		strings.Contains(errStr, "access is denied")
}

// RunWithSudo re-executes the current command with sudo privileges
func RunWithSudo(cmd *cobra.Command, args []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("sudo functionality is not supported on Windows")
	}

	fmt.Println("🔐 Requesting elevated privileges...")
	fmt.Println("⚠️  WARNING: You are about to run fman with sudo privileges.")
	fmt.Print("   Continue? (y/N): ")

	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Operation cancelled.")
		return nil
	}

	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build the sudo command
	sudoArgs := []string{executable, "scan"}
	sudoArgs = append(sudoArgs, args...)

	// Add other flags except --force-sudo to avoid infinite recursion
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		sudoArgs = append(sudoArgs, "--verbose")
	}

	// Execute with sudo
	sudoCmd := exec.Command("sudo", sudoArgs...)
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	sudoCmd.Stdin = os.Stdin

	return sudoCmd.Run()
}
