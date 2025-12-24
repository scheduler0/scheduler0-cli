package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Scheduler0 CLI with credentials",
	Long: `Initialize the Scheduler0 CLI by configuring your API credentials.
This will save your credentials locally for use in subsequent commands.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&baseURL, "base-url", "", "Scheduler0 API base URL (e.g., https://api.scheduler0.com)")
	initCmd.Flags().StringVar(&apiKey, "api-key", "", "Your API key")
	initCmd.Flags().StringVar(&apiSecret, "api-secret", "", "Your API secret")
	initCmd.Flags().StringVar(&accountID, "account-id", "", "Your account ID")
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{}

	// Try to load existing config
	existingConfig, err := config.LoadConfig()
	if err == nil {
		cfg = existingConfig
		fmt.Println("Found existing configuration. You can update it now.")
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)

	// Get base URL
	if baseURL == "" {
		prompt := fmt.Sprintf("Base URL [%s]: ", cfg.BaseURL)
		if cfg.BaseURL == "" {
			prompt = "Base URL (e.g., https://api.scheduler0.com): "
		}
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.BaseURL = input
		} else if cfg.BaseURL == "" {
			return fmt.Errorf("base URL is required")
		}
	} else {
		cfg.BaseURL = baseURL
	}

	// Get API Key
	if apiKey == "" {
		prompt := fmt.Sprintf("API Key [%s]: ", maskString(cfg.APIKey))
		if cfg.APIKey == "" {
			prompt = "API Key: "
		}
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.APIKey = input
		} else if cfg.APIKey == "" {
			return fmt.Errorf("API key is required")
		}
	} else {
		cfg.APIKey = apiKey
	}

	// Get API Secret
	if apiSecret == "" {
		prompt := fmt.Sprintf("API Secret [%s]: ", maskString(cfg.APISecret))
		if cfg.APISecret == "" {
			prompt = "API Secret: "
		}
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.APISecret = input
		} else if cfg.APISecret == "" {
			return fmt.Errorf("API secret is required")
		}
	} else {
		cfg.APISecret = apiSecret
	}

	// Get Account ID
	if accountID == "" {
		prompt := fmt.Sprintf("Account ID [%s]: ", cfg.AccountID)
		if cfg.AccountID == "" {
			prompt = "Account ID: "
		}
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			cfg.AccountID = input
		} else if cfg.AccountID == "" {
			return fmt.Errorf("account ID is required")
		}
	} else {
		cfg.AccountID = accountID
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save config
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
	fmt.Println("You can now use scheduler0 commands!")

	return nil
}

func maskString(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

