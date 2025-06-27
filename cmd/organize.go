/*
Copyright © 2025 changheonshin

*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devlikebear/fman/internal/ai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// organizeCmd represents the organize command
var organizeCmd = &cobra.Command{
	Use:   "organize <directory>",
	Short: "Organizes files in a directory, with suggestions from AI.",
	Long: `Organizes files within a specified directory. 
When used with the --ai flag, it leverages an AI provider (like Gemini or Ollama)
to get suggestions for how to best organize the files. It then presents the suggested
shell commands to the user for approval before executing them.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		aiFlag, _ := cmd.Flags().GetBool("ai")
		if !aiFlag {
			return fmt.Errorf("the --ai flag is required to use AI-based organization")
		}

		providerName := viper.GetString("ai_provider")
		var provider ai.AIProvider

		switch providerName {
		case "gemini":
			provider = ai.NewGeminiProvider()
		case "ollama":
			provider = ai.NewOllamaProvider()
		default:
			return fmt.Errorf("unknown AI provider: %s. Please check your config file", providerName)
		}

		dir := args[0]
		fmt.Printf("Getting AI suggestions to organize directory: %s (using %s)\n", dir, providerName)

		var filePaths []string
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				filePaths = append(filePaths, filepath.Join(dir, entry.Name()))
			}
		}

		if len(filePaths) == 0 {
			fmt.Println("No files to organize in the specified directory.")
			return nil
		}

		suggestions, err := provider.SuggestOrganization(context.Background(), filePaths)
		if err != nil {
			return fmt.Errorf("failed to get suggestions from AI provider: %w", err)
		}

		fmt.Println("\nAI-suggested commands:")
		fmt.Println("------------------------")
		fmt.Println(suggestions)
		fmt.Println("------------------------")
		fmt.Print("Do you want to execute these commands? (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		if strings.ToLower(strings.TrimSpace(response)) == "y" {
			fmt.Println("Executing commands...")
			// We will execute the script in the context of the target directory
			cmd := exec.Command("bash", "-c", suggestions)
			cmd.Dir = dir // Run the command in the directory we are organizing
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to execute suggested commands: %w", err)
			}
			fmt.Println("Commands executed successfully.")
		} else {
			fmt.Println("Organization cancelled.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(organizeCmd)
	organizeCmd.Flags().Bool("ai", false, "Use AI to get organization suggestions")
}
