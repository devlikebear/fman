package rules

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devlikebear/fman/internal/db"
)

// Executor executes rule actions on files
type Executor struct {
	dryRun  bool
	verbose bool
	confirm bool
}

// NewExecutor creates a new rule executor
func NewExecutor(dryRun, verbose, confirm bool) *Executor {
	return &Executor{
		dryRun:  dryRun,
		verbose: verbose,
		confirm: confirm,
	}
}

// ExecuteRule executes all actions for a rule on a matching file
func (ex *Executor) ExecuteRule(rule Rule, file db.File, baseDir string) ExecutionResult {
	result := ExecutionResult{
		Rule:    &rule,
		File:    file,
		Actions: []ActionResult{},
		Success: true,
	}

	if ex.verbose {
		fmt.Printf("Executing rule '%s' on file: %s\n", rule.Name, file.Path)
	}

	// Execute all actions for this rule
	for _, action := range rule.Actions {
		actionResult := ex.executeAction(action, file, baseDir)
		result.Actions = append(result.Actions, actionResult)

		if !actionResult.Success && !actionResult.Skipped {
			result.Success = false
			result.Error = actionResult.Error
			if ex.verbose {
				fmt.Printf("Action failed: %v\n", actionResult.Error)
			}
			break // Stop on first failure
		}
	}

	return result
}

// executeAction executes a single action
func (ex *Executor) executeAction(action Action, file db.File, baseDir string) ActionResult {
	result := ActionResult{
		Action:  action,
		Source:  file.Path,
		Success: false,
	}

	// Handle confirmation if required
	if (action.Confirm || ex.confirm) && !ex.dryRun {
		if !ex.askConfirmation(action, file) {
			result.Skipped = true
			result.SkipReason = "user cancelled"
			return result
		}
	}

	switch action.Type {
	case ActionMove:
		return ex.executeMove(action, file, baseDir)
	case ActionCopy:
		return ex.executeCopy(action, file, baseDir)
	case ActionDelete:
		return ex.executeDelete(action, file)
	case ActionRename:
		return ex.executeRename(action, file, baseDir)
	case ActionLink:
		return ex.executeLink(action, file, baseDir)
	default:
		result.Error = fmt.Errorf("unsupported action type: %s", action.Type)
		return result
	}
}

// executeMove executes a move action
func (ex *Executor) executeMove(action Action, file db.File, baseDir string) ActionResult {
	result := ActionResult{
		Action: action,
		Source: file.Path,
	}

	destination, err := ex.resolveDestination(action, file, baseDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve destination: %w", err)
		return result
	}

	result.Destination = destination

	if ex.dryRun {
		if ex.verbose {
			fmt.Printf("DRY RUN: Would move %s to %s\n", file.Path, destination)
		}
		result.Success = true
		return result
	}

	// Create backup if requested
	if action.Backup {
		if err := ex.createBackup(file.Path); err != nil {
			result.Error = fmt.Errorf("failed to create backup: %w", err)
			return result
		}
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		result.Error = fmt.Errorf("failed to create destination directory: %w", err)
		return result
	}

	// Check if destination already exists
	if _, err := os.Stat(destination); err == nil {
		result.Error = fmt.Errorf("destination already exists: %s", destination)
		return result
	}

	// Perform the move
	if err := os.Rename(file.Path, destination); err != nil {
		result.Error = fmt.Errorf("failed to move file: %w", err)
		return result
	}

	if ex.verbose {
		fmt.Printf("Moved %s to %s\n", file.Path, destination)
	}

	result.Success = true
	return result
}

// executeCopy executes a copy action
func (ex *Executor) executeCopy(action Action, file db.File, baseDir string) ActionResult {
	result := ActionResult{
		Action: action,
		Source: file.Path,
	}

	destination, err := ex.resolveDestination(action, file, baseDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve destination: %w", err)
		return result
	}

	result.Destination = destination

	if ex.dryRun {
		if ex.verbose {
			fmt.Printf("DRY RUN: Would copy %s to %s\n", file.Path, destination)
		}
		result.Success = true
		return result
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		result.Error = fmt.Errorf("failed to create destination directory: %w", err)
		return result
	}

	// Check if destination already exists
	if _, err := os.Stat(destination); err == nil {
		result.Error = fmt.Errorf("destination already exists: %s", destination)
		return result
	}

	// Perform the copy
	if err := ex.copyFile(file.Path, destination); err != nil {
		result.Error = fmt.Errorf("failed to copy file: %w", err)
		return result
	}

	if ex.verbose {
		fmt.Printf("Copied %s to %s\n", file.Path, destination)
	}

	result.Success = true
	return result
}

// executeDelete executes a delete action
func (ex *Executor) executeDelete(action Action, file db.File) ActionResult {
	result := ActionResult{
		Action:      action,
		Source:      file.Path,
		Destination: "", // No destination for delete
	}

	if ex.dryRun {
		if ex.verbose {
			fmt.Printf("DRY RUN: Would delete %s\n", file.Path)
		}
		result.Success = true
		return result
	}

	// Create backup if requested
	if action.Backup {
		if err := ex.createBackup(file.Path); err != nil {
			result.Error = fmt.Errorf("failed to create backup: %w", err)
			return result
		}
	}

	// Perform the deletion
	if err := os.Remove(file.Path); err != nil {
		result.Error = fmt.Errorf("failed to delete file: %w", err)
		return result
	}

	if ex.verbose {
		fmt.Printf("Deleted %s\n", file.Path)
	}

	result.Success = true
	return result
}

// executeRename executes a rename action
func (ex *Executor) executeRename(action Action, file db.File, baseDir string) ActionResult {
	result := ActionResult{
		Action: action,
		Source: file.Path,
	}

	// For rename, destination should be just the new filename
	newName := action.Destination
	if action.Template != "" {
		var err error
		newName, err = ex.resolveTemplate(action.Template, file)
		if err != nil {
			result.Error = fmt.Errorf("failed to resolve template: %w", err)
			return result
		}
	}

	destination := filepath.Join(filepath.Dir(file.Path), newName)
	result.Destination = destination

	if ex.dryRun {
		if ex.verbose {
			fmt.Printf("DRY RUN: Would rename %s to %s\n", file.Path, destination)
		}
		result.Success = true
		return result
	}

	// Check if destination already exists
	if _, err := os.Stat(destination); err == nil {
		result.Error = fmt.Errorf("destination already exists: %s", destination)
		return result
	}

	// Create backup if requested
	if action.Backup {
		if err := ex.createBackup(file.Path); err != nil {
			result.Error = fmt.Errorf("failed to create backup: %w", err)
			return result
		}
	}

	// Perform the rename
	if err := os.Rename(file.Path, destination); err != nil {
		result.Error = fmt.Errorf("failed to rename file: %w", err)
		return result
	}

	if ex.verbose {
		fmt.Printf("Renamed %s to %s\n", file.Path, destination)
	}

	result.Success = true
	return result
}

// executeLink executes a link action (symbolic link)
func (ex *Executor) executeLink(action Action, file db.File, baseDir string) ActionResult {
	result := ActionResult{
		Action: action,
		Source: file.Path,
	}

	destination, err := ex.resolveDestination(action, file, baseDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve destination: %w", err)
		return result
	}

	result.Destination = destination

	if ex.dryRun {
		if ex.verbose {
			fmt.Printf("DRY RUN: Would create link from %s to %s\n", destination, file.Path)
		}
		result.Success = true
		return result
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		result.Error = fmt.Errorf("failed to create destination directory: %w", err)
		return result
	}

	// Check if destination already exists
	if _, err := os.Stat(destination); err == nil {
		result.Error = fmt.Errorf("destination already exists: %s", destination)
		return result
	}

	// Create symbolic link
	if err := os.Symlink(file.Path, destination); err != nil {
		result.Error = fmt.Errorf("failed to create symbolic link: %w", err)
		return result
	}

	if ex.verbose {
		fmt.Printf("Created symbolic link from %s to %s\n", destination, file.Path)
	}

	result.Success = true
	return result
}

// resolveDestination resolves the destination path for an action
func (ex *Executor) resolveDestination(action Action, file db.File, baseDir string) (string, error) {
	var destination string

	if action.Template != "" {
		var err error
		destination, err = ex.resolveTemplate(action.Template, file)
		if err != nil {
			return "", err
		}
	} else {
		destination = action.Destination
	}

	// Expand home directory
	if strings.HasPrefix(destination, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		destination = filepath.Join(homeDir, destination[2:])
	}

	// If destination is a directory, append the filename
	if strings.HasSuffix(destination, "/") || strings.HasSuffix(destination, "\\") {
		filename := filepath.Base(file.Path)
		destination = filepath.Join(destination, filename)
	}

	return destination, nil
}

// resolveTemplate resolves template variables in a string
func (ex *Executor) resolveTemplate(template string, file db.File) (string, error) {
	result := template

	// Replace common template variables
	replacements := map[string]string{
		"{filename}":  filepath.Base(file.Path),
		"{basename}":  strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path)),
		"{ext}":       filepath.Ext(file.Path),
		"{dir}":       filepath.Dir(file.Path),
		"{size}":      fmt.Sprintf("%d", file.Size),
		"{year}":      file.ModifiedAt.Format("2006"),
		"{month}":     file.ModifiedAt.Format("01"),
		"{day}":       file.ModifiedAt.Format("02"),
		"{date}":      file.ModifiedAt.Format("2006-01-02"),
		"{timestamp}": file.ModifiedAt.Format("20060102-150405"),
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// createBackup creates a backup of a file
func (ex *Executor) createBackup(filePath string) error {
	backupPath := filePath + ".backup." + time.Now().Format("20060102-150405")

	if ex.verbose {
		fmt.Printf("Creating backup: %s\n", backupPath)
	}

	return ex.copyFile(filePath, backupPath)
}

// copyFile copies a file from src to dst
func (ex *Executor) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents
	buffer := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := sourceFile.Read(buffer)
		if n > 0 {
			if _, writeErr := destFile.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
	}

	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// askConfirmation asks the user for confirmation
func (ex *Executor) askConfirmation(action Action, file db.File) bool {
	fmt.Printf("Execute %s action on %s? (y/N): ", action.Type, file.Path)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}

	return false // Default to no
}

// GetExecutorSettings returns the current executor settings for testing
func (ex *Executor) GetExecutorSettings() (bool, bool, bool) {
	return ex.dryRun, ex.verbose, ex.confirm
}

// ValidateActionType checks if an action type is valid
func ValidateActionType(actionType ActionType) bool {
	switch actionType {
	case ActionMove, ActionCopy, ActionDelete, ActionRename, ActionLink:
		return true
	default:
		return false
	}
}
