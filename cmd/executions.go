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

var executionsAnalyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Get execution counts grouped by minute buckets for a date range",
	Long:  "Retrieves execution counts grouped by minute buckets for a date range (GET /api/v1/executions/analytics). All dates and times should be in UTC.",
	RunE:  runExecutionsAnalytics,
}

var executionsTotalsCmd = &cobra.Command{
	Use:   "totals",
	Short: "Get total scheduled/success/failed execution counts",
	Long:  "Retrieves total counts of scheduled, success, and failed executions for an account (GET /api/v1/executions/totals).",
	RunE:  runExecutionsTotals,
}

var executionsCleanupCmd = &cobra.Command{
	Use:   "cleanup-old-logs",
	Short: "Delete execution logs older than a retention period",
	Long:  "Deletes execution logs older than the given retention period (POST /api/v1/executions/cleanup-old-logs).",
	RunE:  runExecutionsCleanup,
}

func init() {
	rootCmd.AddCommand(executionsCmd)
	executionsCmd.AddCommand(executionsAnalyticsCmd)
	executionsCmd.AddCommand(executionsTotalsCmd)
	executionsCmd.AddCommand(executionsCleanupCmd)

	executionsCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	executionsCmd.Flags().String("start-date", "", "Start date for filtering (RFC3339 format, required)")
	executionsCmd.Flags().String("end-date", "", "End date for filtering (RFC3339 format, required)")
	executionsCmd.Flags().Int64("project-id", 0, "Filter by project ID (0 for all)")
	executionsCmd.Flags().Int64("job-id", 0, "Filter by job ID (0 for all)")
	executionsCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	executionsCmd.Flags().Int("offset", 0, "Number of items to skip")

	_ = executionsCmd.MarkFlagRequired("start-date")
	_ = executionsCmd.MarkFlagRequired("end-date")

	executionsAnalyticsCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	executionsAnalyticsCmd.Flags().String("start-date", "", "Start date, UTC (RFC3339 date, required)")
	executionsAnalyticsCmd.Flags().String("start-time", "", "Start time, UTC (required)")
	_ = executionsAnalyticsCmd.MarkFlagRequired("start-date")
	_ = executionsAnalyticsCmd.MarkFlagRequired("start-time")

	executionsTotalsCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")

	executionsCleanupCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command, required)")
	executionsCleanupCmd.Flags().Int("retention-months", 0, "Delete logs older than this many months (required)")
	_ = executionsCleanupCmd.MarkFlagRequired("account-id")
	_ = executionsCleanupCmd.MarkFlagRequired("retention-months")
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

func runExecutionsAnalytics(cmd *cobra.Command, args []string) error {
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
	startTime, _ := cmd.Flags().GetString("start-time")

	params := scheduler0_client.GetDateRangeAnalyticsParams{
		StartDate: startDate,
		StartTime: startTime,
	}

	result, err := cl.GetDateRangeAnalytics(params)
	if err != nil {
		return fmt.Errorf("failed to get execution analytics: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runExecutionsTotals(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	result, err := cl.GetExecutionTotals(0)
	if err != nil {
		return fmt.Errorf("failed to get execution totals: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runExecutionsCleanup(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	retentionMonths, _ := cmd.Flags().GetInt("retention-months")

	result, err := cl.CleanupOldExecutionLogs(cfg.AccountID, retentionMonths)
	if err != nil {
		return fmt.Errorf("failed to clean up old execution logs: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

