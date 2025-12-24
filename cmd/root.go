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
	username  string
	password  string
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
	// Check for basic auth flags first
	if baseURL != "" && username != "" && password != "" {
		return &config.Config{
			BaseURL:  baseURL,
			Username: username,
			Password: password,
			AuthType: "basic",
		}, nil
	}
	
	// Check for API key auth flags
	if baseURL != "" && apiKey != "" && apiSecret != "" && accountID != "" {
		return &config.Config{
			BaseURL:   baseURL,
			APIKey:    apiKey,
			APISecret: apiSecret,
			AccountID: accountID,
			AuthType:  "api_key",
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
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (overrides config, for API key auth)")
	rootCmd.PersistentFlags().StringVar(&apiSecret, "api-secret", "", "API secret (overrides config, for API key auth)")
	rootCmd.PersistentFlags().StringVar(&accountID, "account-id", "", "Account ID (overrides config, for API key auth)")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Username (overrides config, for basic auth - self-hosted)")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Password (overrides config, for basic auth - self-hosted)")
}

