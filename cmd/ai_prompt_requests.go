package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var aiPromptRequestsCmd = &cobra.Command{
	Use:   "ai-prompt-requests",
	Short: "List the account's AI prompt-request log",
	Long:  "List and filter the account's AI prompt-request audit log (GET /api/v1/ai/prompt-requests).",
	RunE:  runAIPromptRequestsList,
}

func init() {
	rootCmd.AddCommand(aiPromptRequestsCmd)

	aiPromptRequestsCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	aiPromptRequestsCmd.Flags().String("provider", "", "Filter by provider (e.g. openai, anthropic)")
	aiPromptRequestsCmd.Flags().String("model", "", "Filter by model")
	aiPromptRequestsCmd.Flags().String("status", "", "Filter by status")
	aiPromptRequestsCmd.Flags().String("search", "", "Free-text search over prompt requests")
	aiPromptRequestsCmd.Flags().String("start-date", "", "Start date for filtering (RFC3339 format)")
	aiPromptRequestsCmd.Flags().String("end-date", "", "End date for filtering (RFC3339 format)")
	aiPromptRequestsCmd.Flags().String("order", "", "Order direction (ASC or DESC)")
	aiPromptRequestsCmd.Flags().Uint64("limit", 0, "Maximum number of items to return (0 uses the server default)")
	aiPromptRequestsCmd.Flags().Uint64("offset", 0, "Number of items to skip")
}

func runAIPromptRequestsList(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	status, _ := cmd.Flags().GetString("status")
	search, _ := cmd.Flags().GetString("search")
	startDate, _ := cmd.Flags().GetString("start-date")
	endDate, _ := cmd.Flags().GetString("end-date")
	order, _ := cmd.Flags().GetString("order")
	limit, _ := cmd.Flags().GetUint64("limit")
	offset, _ := cmd.Flags().GetUint64("offset")

	params := scheduler0_client.ListPromptRequestsParams{
		Provider:  provider,
		Model:     model,
		Status:    status,
		Search:    search,
		StartDate: startDate,
		EndDate:   endDate,
		Order:     order,
		Limit:     limit,
		Offset:    offset,
	}

	result, err := cl.ListPromptRequests(params)
	if err != nil {
		return fmt.Errorf("failed to list AI prompt requests: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
