/*
Copyright © 2025 changheonshin

*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// fileSystem is the filesystem abstraction, defaults to osFs
var fileSystem = afero.NewOsFs()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fman",
	Short: "fman is a powerful CLI tool to manage your local files intelligently with AI.",
	Long: `fman (File Manager) is a command-line interface (CLI) tool developed in Go.
It helps you organize and manage local files intelligently, utilizing AI to suggest
and perform file operations.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Find home directory.
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	configPath := filepath.Join(home, ".fman")
	configName := "config"
	configType := "yml"
	configFile := filepath.Join(configPath, fmt.Sprintf("%s.%s", configName, configType))

	viper.AddConfigPath(configPath)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		// If config file not found, create it
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := os.MkdirAll(configPath, os.ModePerm); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating config directory: %s\n", err)
				os.Exit(1)
			}

			defaultConfig := `# ~/.fman/config.yml
# 사용할 AI 공급자를 선택합니다. (gemini 또는 ollama)
ai_provider: "gemini"

gemini:
  # Gemini API 키를 입력하세요.
  api_key: "YOUR_GEMINI_API_KEY"
  # 사용할 모델을 지정합니다.
  model: "gemini-1.5-flash"

ollama:
  # Ollama 서버 주소를 입력하세요.
  base_url: "http://localhost:11434"
  # 사용할 모델을 지정합니다.
  model: "llama3"
`
			if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating config file: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("Configuration file created at:", configFile)
			fmt.Println("Please edit it to add your API key.")

		} else {
			// Config file was found but another error was produced
			fmt.Fprintf(os.Stderr, "Error reading config file: %s\n", err)
			os.Exit(1)
		}
	}
}
