package client

import (
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
)

// NewClient creates a new Scheduler0 client from config
func NewClient(cfg *config.Config) (*scheduler0_client.Client, error) {
	// Determine auth type if not explicitly set
	authType := cfg.AuthType
	if authType == "" {
		if cfg.Username != "" && cfg.Password != "" {
			authType = "basic"
		} else if cfg.APIKey != "" && cfg.APISecret != "" {
			authType = "api_key"
		}
	}

	var client *scheduler0_client.Client
	var err error

	if authType == "basic" {
		// Basic authentication for self-hosted instances
		client, err = scheduler0_client.NewBasicAuthClient(
			cfg.BaseURL,
			"v1",
			cfg.Username,
			cfg.Password,
		)
	} else {
		// API Key authentication for managed/hosted instances
		client, err = scheduler0_client.NewAPIClientWithAccount(
			cfg.BaseURL,
			"v1",
			cfg.APIKey,
			cfg.APISecret,
			cfg.AccountID,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}
