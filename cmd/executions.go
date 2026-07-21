package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var executionsCmd = &cobra.Command{
	Use:   "executions",
	Short: "List job executions",
	Long:  "List job execution logs with date filtering",
	RunE:  runExecutionsList,
}

func init() {
	rootCmd.AddCommand(executionsCmd)

	executionsCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	executionsCmd.Flags().String("start-date", "", "Start date for filtering (RFC3339 format, required)")
	executionsCmd.Flags().String("end-date", "", "End date for filtering (RFC3339 format, required)")
	executionsCmd.Flags().Int64("project-id", 0, "Filter by project ID (0 for all)")
	executionsCmd.Flags().Int64("job-id", 0, "Filter by job ID (0 for all)")
	executionsCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	executionsCmd.Flags().Int("offset", 0, "Number of items to skip")

	_ = executionsCmd.MarkFlagRequired("start-date")
	_ = executionsCmd.MarkFlagRequired("end-date")
}

func runExecutionsList(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	startDate, _ := cmd.Flags().GetString("start-date")
	endDate, _ := cmd.Flags().GetString("end-date")
	projectID, _ := cmd.Flags().GetInt64("project-id")
	jobID, _ := cmd.Flags().GetInt64("job-id")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")

	params := scheduler0_client.ListExecutionsParams{
		StartDate: startDate,
		EndDate:   endDate,
		ProjectID: projectID,
		JobID:     jobID,
		Limit:     limit,
		Offset:    offset,
	}

	result, err := cl.ListExecutions(params)
	if err != nil {
		return fmt.Errorf("failed to list executions: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

