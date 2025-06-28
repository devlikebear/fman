/*
Copyright © 2025 changheonshin
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/devlikebear/fman/internal/db"
	"github.com/devlikebear/fman/internal/rules"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	rulesVerbose bool
	rulesDryRun  bool
	rulesConfirm bool
	rulesDir     string
)

// Helper function to get config directory
func getConfigDir() string {
	configDir := viper.GetString("config_dir")
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".fman")
	}
	return configDir
}

// rulesCmd represents the rules command
var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage file organization rules",
	Long: `Manage file organization rules for automated file management.

Rules allow you to define conditions and actions for automatic file organization.
For example, you can create rules to move old screenshots to an archive folder,
or delete temporary files that are larger than a certain size.

Available subcommands:
  list     - List all rules
  add      - Add a new rule (interactive)
  remove   - Remove a rule
  apply    - Apply rules to files
  enable   - Enable a rule
  disable  - Disable a rule
  init     - Initialize with example rules

Examples:
  fman rules list                    # List all rules
  fman rules apply --dry-run         # Preview what rules would do
  fman rules apply ~/Downloads       # Apply rules to Downloads folder
  fman rules enable screenshot-rule  # Enable a specific rule
  fman rules init                    # Create example rules`,
}

// rulesListCmd represents the rules list command
var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rules",
	Long:  `List all file organization rules with their status and descriptions.`,
	RunE:  runRulesList,
}

// rulesApplyCmd represents the rules apply command
var rulesApplyCmd = &cobra.Command{
	Use:   "apply [directory]",
	Short: "Apply rules to organize files",
	Long: `Apply enabled rules to organize files in the specified directory.
If no directory is specified, applies to all indexed files.

The --dry-run flag allows you to preview what actions would be taken
without actually modifying any files.`,
	RunE: runRulesApply,
}

// rulesRemoveCmd represents the rules remove command
var rulesRemoveCmd = &cobra.Command{
	Use:   "remove <rule-name>",
	Short: "Remove a rule",
	Long:  `Remove a file organization rule by name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRulesRemove,
}

// rulesEnableCmd represents the rules enable command
var rulesEnableCmd = &cobra.Command{
	Use:   "enable <rule-name>",
	Short: "Enable a rule",
	Long:  `Enable a file organization rule by name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRulesEnable,
}

// rulesDisableCmd represents the rules disable command
var rulesDisableCmd = &cobra.Command{
	Use:   "disable <rule-name>",
	Short: "Disable a rule",
	Long:  `Disable a file organization rule by name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRulesDisable,
}

// rulesInitCmd represents the rules init command
var rulesInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize with example rules",
	Long:  `Create example file organization rules to get started.`,
	RunE:  runRulesInit,
}

func init() {
	rootCmd.AddCommand(rulesCmd)

	// Add subcommands
	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesApplyCmd)
	rulesCmd.AddCommand(rulesRemoveCmd)
	rulesCmd.AddCommand(rulesEnableCmd)
	rulesCmd.AddCommand(rulesDisableCmd)
	rulesCmd.AddCommand(rulesInitCmd)

	// Add flags
	rulesApplyCmd.Flags().BoolVar(&rulesDryRun, "dry-run", false, "Preview actions without executing them")
	rulesApplyCmd.Flags().BoolVarP(&rulesVerbose, "verbose", "v", false, "Verbose output")
	rulesApplyCmd.Flags().BoolVar(&rulesConfirm, "confirm", false, "Ask for confirmation before each action")
	rulesApplyCmd.Flags().StringVar(&rulesDir, "dir", "", "Apply rules only to files in this directory")

	rulesListCmd.Flags().BoolVarP(&rulesVerbose, "verbose", "v", false, "Show detailed rule information")
}

// runRulesList lists all rules
func runRulesList(cmd *cobra.Command, args []string) error {
	manager := rules.NewManager(getConfigDir())
	if err := manager.LoadRules(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	allRules := manager.GetRules()
	if len(allRules) == 0 {
		fmt.Println("No rules found. Use 'fman rules init' to create example rules.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	if rulesVerbose {
		fmt.Fprintln(w, "NAME\tSTATUS\tCONDITIONS\tACTIONS\tDESCRIPTION\tUPDATED")
		fmt.Fprintln(w, "----\t------\t----------\t-------\t-----------\t-------")
	} else {
		fmt.Fprintln(w, "NAME\tSTATUS\tDESCRIPTION")
		fmt.Fprintln(w, "----\t------\t-----------")
	}

	for _, rule := range allRules {
		status := "disabled"
		if rule.Enabled {
			status = "enabled"
		}

		if rulesVerbose {
			conditionsStr := fmt.Sprintf("%d conditions", len(rule.Conditions))
			actionsStr := fmt.Sprintf("%d actions", len(rule.Actions))
			updatedStr := rule.UpdatedAt.Format("2006-01-02")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				rule.Name, status, conditionsStr, actionsStr, rule.Description, updatedStr)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n", rule.Name, status, rule.Description)
		}
	}

	return nil
}

// runRulesApply applies rules to organize files
func runRulesApply(cmd *cobra.Command, args []string) error {
	// Initialize database
	database := db.NewDatabase(nil)

	// Load rules
	manager := rules.NewManager(getConfigDir())
	if err := manager.LoadRules(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	enabledRules := manager.GetEnabledRules()
	if len(enabledRules) == 0 {
		fmt.Println("No enabled rules found. Use 'fman rules list' to see available rules.")
		return nil
	}

	// Determine target directory
	targetDir := ""
	if len(args) > 0 {
		targetDir = args[0]
	}
	if rulesDir != "" {
		targetDir = rulesDir
	}

	// Get files to process
	var files []db.File
	var err error

	if targetDir != "" {
		// Expand relative path
		absDir, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("failed to resolve directory path: %w", err)
		}

		// Find files in the specified directory using pattern matching
		criteria := db.SearchCriteria{
			NamePattern: "",
		}

		// Get all files and filter by path
		allFiles, err := database.FindFilesByAdvancedCriteria(criteria)
		if err != nil {
			return fmt.Errorf("failed to find files: %w", err)
		}

		// Filter files by directory
		for _, file := range allFiles {
			if strings.HasPrefix(file.Path, absDir) {
				files = append(files, file)
			}
		}
	} else {
		// Get all indexed files
		criteria := db.SearchCriteria{
			NamePattern: "", // Empty pattern to get all files
		}
		files, err = database.FindFilesByAdvancedCriteria(criteria)
		if err != nil {
			return fmt.Errorf("failed to get files: %w", err)
		}
	}

	if len(files) == 0 {
		fmt.Println("No files found to process.")
		return nil
	}

	// Initialize evaluator and executor
	evaluator := rules.NewEvaluator(rulesVerbose)
	executor := rules.NewExecutor(rulesDryRun, rulesVerbose, rulesConfirm)

	// Track execution summary
	summary := rules.ExecutionSummary{
		TotalFiles: len(files),
		Results:    []rules.ExecutionResult{},
		Errors:     []error{},
	}
	start := time.Now()

	fmt.Printf("Processing %d files with %d enabled rules...\n", len(files), len(enabledRules))
	if rulesDryRun {
		fmt.Println("DRY RUN MODE - No files will be modified")
	}
	fmt.Println()

	// Process each file
	for _, file := range files {
		processed := false

		// Check each rule
		for _, rule := range enabledRules {
			matches, err := evaluator.EvaluateRule(rule, file, targetDir)
			if err != nil {
				summary.Errors = append(summary.Errors, fmt.Errorf("rule '%s': %w", rule.Name, err))
				continue
			}

			if matches {
				if rulesVerbose {
					fmt.Printf("File %s matches rule '%s'\n", file.Path, rule.Name)
				}

				// Execute the rule
				result := executor.ExecuteRule(rule, file, targetDir)
				summary.Results = append(summary.Results, result)

				if result.Success {
					summary.SuccessfulActions += len(result.Actions)
					processed = true
				} else {
					summary.FailedActions += len(result.Actions)
					if result.Error != nil {
						summary.Errors = append(summary.Errors, result.Error)
					}
				}

				// Count skipped actions
				for _, action := range result.Actions {
					if action.Skipped {
						summary.SkippedActions++
					}
				}

				break // Only apply the first matching rule per file
			}
		}

		if processed {
			summary.ProcessedFiles++
		}
	}

	summary.Duration = time.Since(start)

	// Print summary
	fmt.Println()
	fmt.Println("=== Execution Summary ===")
	fmt.Printf("Total files: %d\n", summary.TotalFiles)
	fmt.Printf("Processed files: %d\n", summary.ProcessedFiles)
	fmt.Printf("Successful actions: %d\n", summary.SuccessfulActions)
	fmt.Printf("Failed actions: %d\n", summary.FailedActions)
	fmt.Printf("Skipped actions: %d\n", summary.SkippedActions)
	fmt.Printf("Duration: %v\n", summary.Duration)

	if len(summary.Errors) > 0 {
		fmt.Printf("\nErrors encountered:\n")
		for _, err := range summary.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	return nil
}

// runRulesRemove removes a rule
func runRulesRemove(cmd *cobra.Command, args []string) error {
	manager := rules.NewManager(getConfigDir())
	if err := manager.LoadRules(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	ruleName := args[0]
	if err := manager.RemoveRule(ruleName); err != nil {
		return fmt.Errorf("failed to remove rule: %w", err)
	}

	fmt.Printf("Rule '%s' removed successfully.\n", ruleName)
	return nil
}

// runRulesEnable enables a rule
func runRulesEnable(cmd *cobra.Command, args []string) error {
	manager := rules.NewManager(getConfigDir())
	if err := manager.LoadRules(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	ruleName := args[0]
	if err := manager.EnableRule(ruleName); err != nil {
		return fmt.Errorf("failed to enable rule: %w", err)
	}

	fmt.Printf("Rule '%s' enabled successfully.\n", ruleName)
	return nil
}

// runRulesDisable disables a rule
func runRulesDisable(cmd *cobra.Command, args []string) error {
	manager := rules.NewManager(getConfigDir())
	if err := manager.LoadRules(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	ruleName := args[0]
	if err := manager.DisableRule(ruleName); err != nil {
		return fmt.Errorf("failed to disable rule: %w", err)
	}

	fmt.Printf("Rule '%s' disabled successfully.\n", ruleName)
	return nil
}

// runRulesInit initializes with example rules
func runRulesInit(cmd *cobra.Command, args []string) error {
	manager := rules.NewManager(getConfigDir())
	if err := manager.LoadRules(); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	if err := manager.CreateExampleRules(); err != nil {
		return fmt.Errorf("failed to create example rules: %w", err)
	}

	fmt.Printf("Example rules created successfully in %s\n", manager.GetConfigPath())
	fmt.Println("Use 'fman rules list' to see the created rules.")
	fmt.Println("Rules are disabled by default for safety. Use 'fman rules enable <name>' to enable them.")
	return nil
}
