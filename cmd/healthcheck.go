package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/config"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Check Scheduler0 cluster health",
	Long:  "Check the health of the Scheduler0 cluster and view Raft statistics",
	RunE:  runHealthcheck,
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
}

func runHealthcheck(cmd *cobra.Command, args []string) error {
	// Healthcheck doesn't require authentication, but we need the base URL
	cfg, err := GetClientConfig()
	if err != nil {
		// If no config, try to use base-url flag or default
		if baseURL == "" {
			return fmt.Errorf("base URL is required. Use --base-url flag or run 'scheduler0 init'")
		}
		cfg = &config.Config{BaseURL: baseURL}
	}

	// Create a minimal client for healthcheck (no auth needed)
	cl, err := scheduler0_client.NewClient(cfg.BaseURL, "v1")
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	result, err := cl.Healthcheck()
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

