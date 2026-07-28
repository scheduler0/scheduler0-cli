package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	"github.com/spf13/cobra"
)

var aiModelsRootCmd = &cobra.Command{
	Use:   "ai-models",
	Short: "List the approved AI models catalog",
	Long:  "Get the per-provider approved model catalog (GET /api/v1/ai/models). The catalog lists every model that Scheduler0 accepts for each provider.",
	RunE:  runAIModelsRoot,
}

func init() {
	rootCmd.AddCommand(aiModelsRootCmd)
}

func runAIModelsRoot(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	result, err := cl.GetAIModels()
	if err != nil {
		return fmt.Errorf("failed to get AI models: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
