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

var (
	initUsername string
	initPassword string
	initAuthType string
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&baseURL, "base-url", "", "Scheduler0 API base URL (e.g., https://api.scheduler0.com)")
	initCmd.Flags().StringVar(&apiKey, "api-key", "", "Your API key (for API key authentication)")
	initCmd.Flags().StringVar(&apiSecret, "api-secret", "", "Your API secret (for API key authentication)")
	initCmd.Flags().StringVar(&accountID, "account-id", "", "Your account ID (for API key authentication)")
	initCmd.Flags().StringVar(&initUsername, "username", "", "Username (for basic authentication - self-hosted)")
	initCmd.Flags().StringVar(&initPassword, "password", "", "Password (for basic authentication - self-hosted)")
	initCmd.Flags().StringVar(&initAuthType, "auth-type", "", "Authentication type: 'api_key' or 'basic' (default: auto-detect)")
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
			prompt = "Base URL (e.g., https://api.scheduler0.com or http://localhost:7070): "
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

	// Determine authentication type
	determinedAuthType := initAuthType
	if determinedAuthType == "" {
		// Auto-detect from flags or existing config
		if (initUsername != "" || initPassword != "") || (cfg.Username != "" && cfg.Password != "") {
			determinedAuthType = "basic"
		} else if (apiKey != "" || apiSecret != "") || (cfg.APIKey != "" && cfg.APISecret != "") {
			determinedAuthType = "api_key"
		} else {
			// Ask user to choose
			fmt.Println("\nAuthentication Method:")
			fmt.Println("1. API Key + Secret (for managed/hosted Scheduler0)")
			fmt.Println("2. Username + Password (for self-hosted Scheduler0)")
			fmt.Print("Choose authentication method (1 or 2) [1]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "2" {
				determinedAuthType = "basic"
			} else {
				determinedAuthType = "api_key"
			}
		}
	}
	cfg.AuthType = determinedAuthType

	if cfg.AuthType == "basic" {
		// Basic authentication flow
		if initUsername == "" {
			prompt := fmt.Sprintf("Username [%s]: ", cfg.Username)
			if cfg.Username == "" {
				prompt = "Username (set during infrastructure setup): "
			}
			fmt.Print(prompt)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				cfg.Username = input
			} else if cfg.Username == "" {
				return fmt.Errorf("username is required for basic authentication")
			}
		} else {
			cfg.Username = initUsername
		}

		if initPassword == "" {
			prompt := fmt.Sprintf("Password [%s]: ", maskString(cfg.Password))
			if cfg.Password == "" {
				prompt = "Password (set during infrastructure setup): "
			}
			fmt.Print(prompt)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				cfg.Password = input
			} else if cfg.Password == "" {
				return fmt.Errorf("password is required for basic authentication")
			}
		} else {
			cfg.Password = initPassword
		}
	} else {
		// API Key authentication flow
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
				return fmt.Errorf("account ID is required for API key authentication")
			}
		} else {
			cfg.AccountID = accountID
		}
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

