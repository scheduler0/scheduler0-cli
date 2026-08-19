package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	"github.com/spf13/cobra"
)

var featuresCmd = &cobra.Command{
	Use:   "features",
	Short: "List available features",
	Long:  "List every feature Scheduler0 supports (GET /api/v1/features). To grant one to an account, see 'scheduler0 accounts feature add'.",
	RunE:  runFeaturesList,
}

func init() {
	rootCmd.AddCommand(featuresCmd)
}

func runFeaturesList(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	result, err := cl.ListFeatures()
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
