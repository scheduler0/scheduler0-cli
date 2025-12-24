package cmd

import (
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	baseURL   string
	apiKey    string
	apiSecret string
	accountID string
)

var rootCmd = &cobra.Command{
	Use:   "scheduler0",
	Short: "Scheduler0 CLI - A command-line interface for Scheduler0",
	Long: `Scheduler0 CLI is a command-line tool for interacting with the Scheduler0 API.
	
Use 'scheduler0 init' to configure your credentials before using other commands.`,
	Version: "1.0.0",
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Config is loaded per-command as needed
}

// GetClientConfig returns the client configuration, loading from saved config or flags
func GetClientConfig() (*config.Config, error) {
	// If flags are provided, use them
	if baseURL != "" && apiKey != "" && apiSecret != "" && accountID != "" {
		return &config.Config{
			BaseURL:   baseURL,
			APIKey:    apiKey,
			APISecret: apiSecret,
			AccountID: accountID,
		}, nil
	}

	// Otherwise, load from saved config
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w\nRun 'scheduler0 init' to configure credentials", err)
	}

	return cfg, nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Scheduler0 API base URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().StringVar(&apiSecret, "api-secret", "", "API secret (overrides config)")
	rootCmd.PersistentFlags().StringVar(&accountID, "account-id", "", "Account ID (overrides config)")
}

