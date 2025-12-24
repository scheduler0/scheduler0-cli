package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage jobs",
	Long:  "Commands for managing Scheduler0 jobs",
}

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs",
	Long:  "List all jobs with pagination",
	RunE:  runJobsList,
}

var jobsGetCmd = &cobra.Command{
	Use:   "get [job-id]",
	Short: "Get job details",
	Long:  "Get details of a specific job",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsGet,
}

var jobsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new job",
	Long:  "Create a new scheduled job",
	RunE:  runJobsCreate,
}

var jobsUpdateCmd = &cobra.Command{
	Use:   "update [job-id]",
	Short: "Update a job",
	Long:  "Update an existing job",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsUpdate,
}

var jobsDeleteCmd = &cobra.Command{
	Use:   "delete [job-id]",
	Short: "Delete a job",
	Long:  "Delete a job",
	Args:  cobra.ExactArgs(1),
	RunE:  runJobsDelete,
}

func init() {
	rootCmd.AddCommand(jobsCmd)
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsGetCmd)
	jobsCmd.AddCommand(jobsCreateCmd)
	jobsCmd.AddCommand(jobsUpdateCmd)
	jobsCmd.AddCommand(jobsDeleteCmd)

	jobsListCmd.Flags().String("project-id", "", "Filter by project ID")
	jobsListCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	jobsListCmd.Flags().Int("offset", 0, "Number of items to skip")
	jobsListCmd.Flags().String("order-by", "date_created", "Field to order by")
	jobsListCmd.Flags().String("order-direction", "desc", "Order direction (asc/desc)")

	jobsCreateCmd.Flags().Int64("project-id", 0, "Project ID (required)")
	jobsCreateCmd.Flags().String("timezone", "UTC", "Timezone (required)")
	jobsCreateCmd.Flags().Int64("executor-id", 0, "Executor ID")
	jobsCreateCmd.Flags().String("data", "", "Job payload data")
	jobsCreateCmd.Flags().String("spec", "", "Cron specification")
	jobsCreateCmd.Flags().String("start-date", "", "Start date (RFC3339)")
	jobsCreateCmd.Flags().String("end-date", "", "End date (RFC3339)")
	jobsCreateCmd.Flags().Int64("timezone-offset", 0, "Timezone offset in seconds")
	jobsCreateCmd.Flags().Int("retry-max", 0, "Maximum retries")
	jobsCreateCmd.Flags().String("status", "active", "Job status (active/inactive)")
	jobsCreateCmd.Flags().String("created-by", "", "User who created the job (required)")
	jobsCreateCmd.MarkFlagRequired("project-id")
	jobsCreateCmd.MarkFlagRequired("created-by")

	jobsUpdateCmd.Flags().Int64("project-id", 0, "Project ID")
	jobsUpdateCmd.Flags().Int64("executor-id", 0, "Executor ID")
	jobsUpdateCmd.Flags().String("data", "", "Job payload data")
	jobsUpdateCmd.Flags().String("spec", "", "Cron specification")
	jobsUpdateCmd.Flags().String("start-date", "", "Start date (RFC3339)")
	jobsUpdateCmd.Flags().String("end-date", "", "End date (RFC3339)")
	jobsUpdateCmd.Flags().String("timezone", "", "Timezone")
	jobsUpdateCmd.Flags().Int64("timezone-offset", 0, "Timezone offset in seconds")
	jobsUpdateCmd.Flags().Int("retry-max", 0, "Maximum retries")
	jobsUpdateCmd.Flags().String("status", "", "Job status (active/inactive)")
	jobsUpdateCmd.Flags().String("modified-by", "", "User who modified the job (required)")
	jobsUpdateCmd.MarkFlagRequired("modified-by")

	jobsDeleteCmd.Flags().String("deleted-by", "", "User who deleted the job (required)")
	jobsDeleteCmd.MarkFlagRequired("deleted-by")
}

func runJobsList(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	projectID, _ := cmd.Flags().GetString("project-id")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	orderBy, _ := cmd.Flags().GetString("order-by")
	orderDirection, _ := cmd.Flags().GetString("order-direction")

	params := scheduler0_client.ListJobsParams{
		ProjectID:        projectID,
		Limit:            limit,
		Offset:           offset,
		OrderBy:          orderBy,
		OrderByDirection: orderDirection,
	}

	result, err := cl.ListJobs(params)
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runJobsGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	jobID := args[0]
	result, err := cl.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runJobsCreate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	projectID, _ := cmd.Flags().GetInt64("project-id")
	timezone, _ := cmd.Flags().GetString("timezone")
	executorID, _ := cmd.Flags().GetInt64("executor-id")
	data, _ := cmd.Flags().GetString("data")
	spec, _ := cmd.Flags().GetString("spec")
	startDate, _ := cmd.Flags().GetString("start-date")
	endDate, _ := cmd.Flags().GetString("end-date")
	timezoneOffset, _ := cmd.Flags().GetInt64("timezone-offset")
	retryMax, _ := cmd.Flags().GetInt("retry-max")
	status, _ := cmd.Flags().GetString("status")
	createdBy, _ := cmd.Flags().GetString("created-by")

	job := &scheduler0_client.JobRequestBody{
		ProjectID:      projectID,
		Timezone:       timezone,
		Data:           data,
		Spec:           spec,
		StartDate:      startDate,
		EndDate:        endDate,
		TimezoneOffset: timezoneOffset,
		RetryMax:       retryMax,
		Status:         status,
		CreatedBy:      createdBy,
	}

	if executorID > 0 {
		job.ExecutorID = &executorID
	}

	result, err := cl.CreateJob(job)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	fmt.Println("\nNote: Job creation is async. Use 'scheduler0 async-tasks get <request-id>' to check status.")
	return nil
}

func runJobsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	jobID := args[0]
	projectID, _ := cmd.Flags().GetInt64("project-id")
	executorID, _ := cmd.Flags().GetInt64("executor-id")
	data, _ := cmd.Flags().GetString("data")
	spec, _ := cmd.Flags().GetString("spec")
	startDate, _ := cmd.Flags().GetString("start-date")
	endDate, _ := cmd.Flags().GetString("end-date")
	timezone, _ := cmd.Flags().GetString("timezone")
	timezoneOffset, _ := cmd.Flags().GetInt64("timezone-offset")
	retryMax, _ := cmd.Flags().GetInt("retry-max")
	status, _ := cmd.Flags().GetString("status")
	modifiedBy, _ := cmd.Flags().GetString("modified-by")

	update := &scheduler0_client.JobUpdateRequestBody{
		Data:           data,
		Spec:           spec,
		StartDate:      startDate,
		EndDate:        endDate,
		Timezone:       timezone,
		TimezoneOffset: timezoneOffset,
		RetryMax:       retryMax,
		Status:         status,
		ModifiedBy:     modifiedBy,
	}

	if projectID > 0 {
		update.ProjectID = projectID
	}
	if executorID > 0 {
		update.ExecutorID = &executorID
	}

	result, err := cl.UpdateJob(jobID, update)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runJobsDelete(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	jobID := args[0]
	deletedBy, _ := cmd.Flags().GetString("deleted-by")

	deleteReq := &scheduler0_client.JobDeleteRequestBody{
		DeletedBy: deletedBy,
	}

	err = cl.DeleteJob(jobID, deleteReq)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	fmt.Printf("Job %s deleted successfully\n", jobID)
	return nil
}

