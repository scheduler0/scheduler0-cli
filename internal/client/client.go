package client

import (
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
)

// NewClient creates a new Scheduler0 client from config
func NewClient(cfg *config.Config) (*scheduler0_client.Client, error) {
	client, err := scheduler0_client.NewAPIClientWithAccount(
		cfg.BaseURL,
		"v1",
		cfg.APIKey,
		cfg.APISecret,
		cfg.AccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

