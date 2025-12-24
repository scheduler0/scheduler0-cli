package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	configDirName  = ".scheduler0"
	configFileName = "config.json"
)

type Config struct {
	BaseURL   string `json:"base_url"`
	// API Key authentication (for managed/hosted instances)
	APIKey    string `json:"api_key,omitempty"`
	APISecret string `json:"api_secret,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	// Basic authentication (for self-hosted instances)
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// Auth type: "api_key" or "basic"
	AuthType string `json:"auth_type,omitempty"`
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, configDirName)
	return filepath.Join(configDir, configFileName), nil
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found. Please run 'scheduler0 init' to configure credentials")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to the config file
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates that all required fields are set
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	
	// Determine auth type if not explicitly set
	if c.AuthType == "" {
		if c.Username != "" && c.Password != "" {
			c.AuthType = "basic"
		} else if c.APIKey != "" && c.APISecret != "" {
			c.AuthType = "api_key"
		}
	}
	
	// Validate based on auth type
	if c.AuthType == "basic" {
		if c.Username == "" {
			return fmt.Errorf("username is required for basic authentication")
		}
		if c.Password == "" {
			return fmt.Errorf("password is required for basic authentication")
		}
	} else if c.AuthType == "api_key" {
		if c.APIKey == "" {
			return fmt.Errorf("api_key is required for API key authentication")
		}
		if c.APISecret == "" {
			return fmt.Errorf("api_secret is required for API key authentication")
		}
		if c.AccountID == "" {
			return fmt.Errorf("account_id is required for API key authentication")
		}
	} else {
		return fmt.Errorf("authentication required: provide either username/password (basic) or api_key/api_secret/account_id (api_key)")
	}
	
	return nil
}

// GetConfigFromViper loads config from viper (for init command)
func GetConfigFromViper() (*Config, error) {
	viper.SetConfigType("json")
	config := &Config{
		BaseURL:   viper.GetString("base_url"),
		APIKey:    viper.GetString("api_key"),
		APISecret: viper.GetString("api_secret"),
		AccountID: viper.GetString("account_id"),
		Username:  viper.GetString("username"),
		Password:  viper.GetString("password"),
		AuthType:  viper.GetString("auth_type"),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

